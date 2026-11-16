// Package voice gives each session a shape of its own, and - if asked - a
// small model to speak for it.
//
// The tree gives the pet one shape a life, the same in every window, and the
// repertoire five lines a form. Here every open session wears the line of the
// tree its work calls for, read off what the session has done by fixed rules
// (rules.go): no model, nothing leaves the machine, on by default.
//
// The voice is the optional layer, off until `ccpet voice on`. It asks Haiku -
// through the `claude` already installed and already logged in, so there is no
// key to configure - for two things:
//
//   - three lines about what the session is doing, every half hour of work,
//     which the statusline says one at a time where the idle line goes;
//   - which shape, but only when the rules could not tell: they name the two
//     or three lines of the tree the session fits, and the model picks one.
//
// It was not always this frugal. The first cut asked for one line every ten
// minutes and put all fourteen shapes to the model every half hour: measured,
// some ten calls and twelve thousand tokens per session an hour, most of it
// the fixed cost of a call - about 750 tokens before a word of ours. So the
// rules do what they can do, the model is asked what only it can answer, and
// one call brings back three lines instead of one.
//
// What it is careful about:
//
//   - It spends the person's own quota, a little, and that is theirs to
//     decide.
//   - The hook never waits for it. It claims the turn, starts a detached
//     `ccpet voice say` and returns; the answer lands in the session's file a
//     few seconds later, or it does not and the rules and the repertoire carry
//     on.
//   - What the model is told is what the footer already shows, plus the names
//     of the last tools and the files they touched. Not the conversation, and
//     not a Bash command line: those carry secrets.
//   - The pet underneath is untouched. XP, level, generation and the path the
//     habits chose are the same pet.json; the shape is worn over it, at the
//     depth the level has reached, and booked in the album.
package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kyros-software/claude-code-themes/internal/lockfile"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
)

const (
	// Every is the floor between two requests for lines.
	Every = 30 * time.Minute
	// Turn is how long each of the lines a request brings back is on screen.
	Turn = 10 * time.Minute
	// wanted is how many lines one request asks for: Every worth of Turns.
	wanted = 3
	// asked is how many it is asked for, to keep `wanted`. Measured asking
	// for three: one turn in three came back with none short enough, and the
	// second question that cost is thirteen hundred tokens against the
	// twenty of two spare lines.
	asked = 5
	// Timeout is how long the model gets before the request is dropped.
	Timeout = 45 * time.Second
	// MaxRunes is the longest line band 4 is known to hold - the same bound
	// the repertoire is tested against.
	MaxRunes = 57
	// ShapeEvery is how long a session keeps the shape it was given before it
	// is read again. A shape that changes every ten minutes is a pet nobody
	// recognises.
	ShapeEvery = 30 * time.Minute
	// trailKept is how many tool calls back a session remembers.
	trailKept = 12
	// voiceNeeds is how many steps a session takes before it is worth a
	// request: one that runs a single command and closes should cost nothing.
	// The same number the rules wait for, on purpose - asked any sooner, the
	// lines came back in the voice of a shape the rules replaced four steps
	// later, and were thrown away.
	voiceNeeds = ruleNeeds
	// tries is how many times the model is asked before giving up on a turn.
	tries = 2
	// pastKept is how many of its own lines it is told not to repeat.
	pastKept = 9
	// retry is the floor between two requests when the last one brought
	// nothing back, doubled for every failure in a row up to retryMost: a
	// `claude` that is logged out should not be started every five minutes
	// by every session for the rest of the day.
	retry     = 5 * time.Minute
	retryMost = 4
	// busyFor is how long a process that was started is taken to be still at
	// it, if it never says it has finished: the worst it can take - a doubt
	// and `tries` requests, each up to Timeout - and a little.
	busyFor = (tries+1)*Timeout + 15*time.Second

	// ChildEnv marks a process started on the voice's behalf. The hook does
	// nothing under it: the `claude` asked for the line is itself a Claude
	// Code, and one that fed the pet or asked for a line of its own would be
	// a loop.
	ChildEnv = "CCPET_VOICE_CHILD"
	// ctxEnv carries the Context from the hook to `ccpet voice say`.
	ctxEnv = "CCPET_VOICE_CTX"
)

// File is pet-voice.json, which holds the switch, and also the scratch file of
// one session, which holds everything else: the shape that session wears, its
// line, and the trail of what it has been doing.
type File struct {
	Enabled bool `json:"enabled"`
	// Lines are what the last request brought back, said one per Turn from At
	// on, in the voice of Form.
	Lines   []string `json:"lines,omitempty"`
	Form    string   `json:"form"`
	At      int64    `json:"at"`
	AskedAt int64    `json:"asked_at"`
	Past    []string `json:"past"`

	// Mark is the line of the tree this session wears, by its mark's id.
	Mark   string `json:"mark,omitempty"`
	MarkAt int64  `json:"mark_at,omitempty"`
	// Doubt is the lines the rules could not choose between, left for the
	// model to settle. Empty when they were sure, or it has answered.
	Doubt []string `json:"doubt,omitempty"`
	// SaidAs is the mark the session wore when Lines were written: a shape
	// that has changed since is lines in the wrong voice.
	SaidAs string `json:"said_as,omitempty"`
	// Fails counts requests in a row that brought nothing back.
	Fails int `json:"fails,omitempty"`
	// Busy is when the process now asking the model was started, and zero
	// when none is. One at a time: the doubt and the lines used to be claimed
	// on two consecutive tool calls, half a second apart, and the two
	// processes each asked both questions.
	Busy int64 `json:"busy,omitempty"`
	// Trail is the last tool calls, oldest first: "Edit voice.go".
	Trail []string `json:"trail,omitempty"`

	// What the rules read - see rules.go. Tally counts steps by kind, Files
	// the edits by file, Tools the calls by tool.
	Tally map[string]int `json:"tally,omitempty"`
	Files map[string]int `json:"files,omitempty"`
	Tools map[string]int `json:"tools,omitempty"`
	Steps int            `json:"steps,omitempty"`
	Repro bool           `json:"repro,omitempty"`

	// Fixed, in the switch's file, turns the per-session shapes off: every
	// session draws the shape the tree gave the pet, as before 3.3.
	Fixed bool `json:"fixed,omitempty"`
	// Worn, in the switch's file, counts how many times each line of the tree
	// has been put on by a session: what the rules weigh novelty by.
	Worn map[string]int `json:"worn,omitempty"`
}

// Context is what the hook knows that pet.json does not.
type Context struct {
	Session string `json:"session,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Target  string `json:"target,omitempty"`
	Label   string `json:"label,omitempty"`
	Active  int64  `json:"active,omitempty"`
	// Kind is what sort of step this was - see rules.go.
	Kind string `json:"kind,omitempty"`
}

func (c Context) step() string {
	if c.Tool == "" {
		return ""
	}
	if c.Target == "" {
		return c.Tool
	}
	return c.Tool + " " + c.Target
}

// SessionPath is one session's file, in the scratch directory beside the
// statusline's own, so SessionEnd and the daily sweep take it away. A caller
// with no session - `ccpet voice say` typed by hand - shares the switch's file.
func SessionPath(statePath, sessionID string) string {
	if path := session.PathFor(sessionID, "voice"); path != "" {
		return path
	}
	return PathBeside(statePath)
}

// PathBeside is the voice file that goes with a pet.json: next to it. Derived
// from the state path, not from the config directory, so that a test that
// hands the hook a pet in a temp dir cannot reach the real switch.
func PathBeside(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "pet-voice.json")
}

// Load never fails: a missing or broken file is a voice switched off.
func Load(path string) File {
	var f File
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, &f) != nil {
		return File{}
	}
	return f
}

func save(path string, f File) bool {
	raw, err := json.Marshal(f)
	if err != nil {
		return false
	}
	return session.WriteAtomic(path, raw)
}

// update is read-change-write under the file's lock.
func update(path string, change func(*File) bool) bool {
	release, ok := lockfile.Take(path)
	if !ok {
		return false
	}
	defer release()
	f := Load(path)
	if !change(&f) {
		return false
	}
	return save(path, f)
}

// amend is update for a file that has to be there already. The detached half
// uses it: a session that closed while the model was thinking has had its
// file taken away, and writing the answer would put it back for nobody.
func amend(path string, change func(*File) bool) bool {
	return update(path, func(f *File) bool {
		if _, err := os.Stat(path); err != nil {
			return false
		}
		return change(f)
	})
}

// Mind is what the statusline needs of a session: the line of the tree it
// wears, and its line. The line is "" with the voice off; the shape is not -
// the rules choose it then.
type Mind struct {
	Mark   string
	file   File
	now    time.Time
	voiced bool
}

// Read is one refresh's look at the session: the shape it wears, whoever chose
// it, and - with the voice on - its line.
func Read(statePath, sessionID string, now time.Time) Mind {
	sw := Load(PathBeside(statePath))
	if session.SafeID(sessionID) == "" {
		// No session, no dress: the file would be the switch's own.
		if !sw.Enabled {
			return Mind{}
		}
		return Mind{file: sw, now: now, voiced: true}
	}
	if sw.Fixed && !sw.Enabled {
		return Mind{}
	}
	f := Load(SessionPath(statePath, sessionID))
	mark := ""
	if _, ok := pet.Unlocks[f.Mark]; ok && !sw.Fixed {
		mark = f.Mark
	}
	return Mind{Mark: mark, file: f, now: now, voiced: sw.Enabled}
}

// Line is the line to show for this form right now, or "".
func (m Mind) Line(form string) string {
	f := m.file
	if !m.voiced || len(f.Lines) == 0 || f.Form != form {
		return ""
	}
	age := m.now.Sub(time.Unix(f.At, 0))
	if age < 0 {
		return ""
	}
	// One line a Turn, and silence once they have all had theirs: past that
	// the moment they were written about is gone, and the repertoire speaks.
	i := int(age / Turn)
	if i >= len(f.Lines) {
		return ""
	}
	// Cleaned again on the way out: the file is on disk, and what is in it
	// goes straight to a terminal.
	return Clean(f.Lines[i])
}

// Nudge is the hook's half: note what the session has just done, let the rules
// settle its shape, and - with the voice on - claim a turn that is due and
// start the process that will ask the model. It returns at once either way.
func Nudge(statePath string, c Context, now time.Time) {
	sw := Load(PathBeside(statePath))
	if sw.Fixed && !sw.Enabled {
		return
	}
	if c.Session == "" && !sw.Enabled {
		// No session to dress, and the file would be the switch's own.
		return
	}
	path := SessionPath(statePath, c.Session)
	// Under the lock: a session can finish two tool calls in the same second,
	// and with no session id several of them share this file.
	claimed, worn := false, ""
	update(path, func(f *File) bool {
		wrote := c.Kind != ""
		f.note(c)
		if step := c.step(); step != "" && (len(f.Trail) == 0 || f.Trail[len(f.Trail)-1] != step) {
			f.Trail = append(f.Trail, step)
			if len(f.Trail) > trailKept {
				f.Trail = f.Trail[len(f.Trail)-trailKept:]
			}
			wrote = true
		}
		// The rules first, whoever else is listening: they are free, and
		// what they are sure of nobody needs to be asked.
		doubted := false
		if c.Session != "" && !sw.Fixed {
			var doubt []string
			worn, doubt = f.rule(c, sw.Worn, func() []string { return others(path, now) }, now)
			if sw.Enabled && len(doubt) > 1 {
				f.Doubt, doubted = doubt, true
			}
		}
		if !sw.Enabled {
			return wrote || worn != ""
		}
		// Asked when the session has nothing left to say - no lines, lines
		// that have had their turns, lines in the voice of a shape it no
		// longer wears - or the moment the rules have a doubt for the model
		// to settle. Never on a session that has barely started.
		//
		// It was a clock of its own once, and the two drifted: a doubt moved
		// the clock, the lines ran out on theirs, and the pet went back to the
		// repertoire for twenty minutes with the voice on.
		wears := f.Mark
		if sw.Fixed || c.Session == "" {
			wears = ""
		}
		switch {
		case f.Steps < voiceNeeds:
		case busy(*f, now):
			// Somebody is asking already. It settles the doubt and, if the
			// lines are spent, asks for those too; what it leaves undone is
			// claimed by the first tool call after it has finished.
		case doubted:
			claimed = true // once per reading of the shape: see rule
		case spent(*f, wears, now) && due(*f, now):
			f.AskedAt = now.Unix()
			claimed = true
		}
		if claimed {
			f.Busy = now.Unix()
		}
		return wrote || claimed || worn != ""
	})
	if worn != "" {
		collect(statePath, worn, now)
	}
	if claimed {
		spawn(c)
	}
}

// collect books a shape a session has put on: every rung of the line down to
// where the pet stands goes in the album, marked as worn - shown like one the
// tree gave, and not weighed when the tree chooses the next life's branches.
func collect(statePath, mark string, now time.Time) {
	update(PathBeside(statePath), func(sw *File) bool {
		if sw.Worn == nil {
			sw.Worn = map[string]int{}
		}
		sw.Worn[mark]++
		return true
	})
	natural, _ := pet.CurrentForm(pet.Load(statePath))
	worn := pet.Lineage(pet.Guise(natural, mark))
	pet.Update(statePath, func(s *pet.State) bool {
		before := len(s.Album)
		for _, form := range worn {
			s.CollectWorn(form, now.Unix())
		}
		return len(s.Album) != before
	})
}

// busy says a process started for this session has not finished yet, nor been
// gone long enough to be given up for dead.
func busy(f File, now time.Time) bool {
	since := now.Sub(time.Unix(f.Busy, 0))
	return f.Busy != 0 && since >= 0 && since < busyFor
}

// due says enough time has gone by since the last request to make another:
// long enough for it to have answered, and longer each time it has not.
func due(f File, now time.Time) bool {
	fails := f.Fails
	if fails > retryMost {
		fails = retryMost
	}
	since := now.Sub(time.Unix(f.AskedAt, 0))
	return f.AskedAt == 0 || since < 0 || since >= retry<<fails
}

// spent says the session has no line left for the shape it wears.
//
// Exactly Every, with nothing taken off to ask early: the shape is read on the
// same half hour and a few seconds sooner, so asking at the stroke asks once,
// in the voice of whatever the rules have just made it. Asking early asked
// twice whenever the shape changed.
func spent(f File, wears string, now time.Time) bool {
	age := now.Sub(time.Unix(f.At, 0))
	return len(f.Lines) == 0 || f.SaidAs != wears || age < 0 || age >= Every
}

// stale says the session's lines have had their turns, or were written for
// another shape, or were never written.
func stale(f File, form string, now time.Time) bool {
	age := now.Sub(time.Unix(f.At, 0))
	return len(f.Lines) == 0 || f.Form != form || age < 0 || age >= Every
}

// spawn is a variable so the tests can see a line being asked for without a
// process being started.
var spawn = func(c Context) {
	self, err := os.Executable()
	if err != nil {
		return
	}
	raw, _ := json.Marshal(c)
	cmd := exec.Command(self, "voice", "say")
	cmd.Env = append(os.Environ(), ChildEnv+"=1", ctxEnv+"="+string(raw))
	detach(cmd)
	if cmd.Start() == nil {
		cmd.Process.Release()
	}
}

// ask is the model call, a variable for the same reason.
var ask = func(system, prompt string) (string, error) {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return "", err
	}
	// The character goes in a file and the moment on stdin, not on the
	// command line: both run to several lines, and where `claude` is a .cmd
	// - Windows, installed with npm - an argument with a line break in it
	// does not survive cmd.exe.
	file, err := os.CreateTemp("", "ccpet-voice-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString(system)
	if cerr := file.Close(); err != nil || cerr != nil {
		return "", fmt.Errorf("writing the system prompt: %v %v", err, cerr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	// No settings, so no hooks and no plugins; no MCP servers; no tools; no
	// session left on disk. What is left is one request with the login the
	// person already has. --bare would be shorter and cannot be used: it
	// only takes an API key.
	cmd := exec.CommandContext(ctx, bin, "-p",
		"--model", "haiku",
		// The least it will think: these are one word and three short lines,
		// and unasked a one-word answer was measured at up to 395 tokens out.
		"--effort", "low",
		"--setting-sources", "",
		"--strict-mcp-config",
		"--tools", "",
		"--disable-slash-commands",
		"--no-session-persistence",
		"--output-format", "text",
		"--system-prompt-file", file.Name())
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = append(os.Environ(), ChildEnv+"=1")
	// Away from the project: its CLAUDE.md is not this request's business.
	cmd.Dir = os.TempDir()
	// The deadline kills `claude`; this is for whatever it started that still
	// holds the pipe, which Output would otherwise wait on for ever.
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.Output()
	return string(out), err
}

// Say is the detached half: settle the doubt the rules left, if any, and ask
// for the session's lines if the ones it has are spent. force asks for lines
// whatever their age, for `ccpet voice say` typed by hand. It reports the
// lines the session has afterwards, new or not.
func Say(statePath string, c Context, now time.Time, force bool) []string {
	s := pet.Load(statePath)
	pet.DecayHunger(s, now)
	natural, level := pet.CurrentForm(s)
	path := SessionPath(statePath, c.Session)
	f := Load(path)
	// Typed by hand there is no session: the file is the switch's own, and it
	// may not exist yet.
	write := amend
	if c.Session == "" {
		write = update
	}
	// However it ends, the session is free to ask again.
	defer write(path, func(f *File) bool {
		if f.Busy == 0 {
			return false
		}
		f.Busy = 0
		return true
	})

	// Only a session has a shape of its own. Typed by hand there is no
	// session to dress, and the pet speaks as what the tree made it.
	fixed := Load(PathBeside(statePath)).Fixed
	mark := f.Mark
	if fixed || c.Session == "" {
		mark = ""
	} else if len(f.Doubt) > 1 {
		picked := chooseMark(c, f, f.Doubt, others(path, now))
		write(path, func(f *File) bool {
			f.Doubt = nil
			if picked != "" {
				f.Mark = picked
			}
			return true
		})
		if picked != "" && picked != mark {
			mark = picked
			collect(statePath, mark, now)
		}
	}
	form := pet.Guise(natural, mark)
	if !force && !stale(f, form, now) {
		return f.Lines
	}

	// A small model does not count characters; it is told a number of words,
	// and whatever still comes back long is dropped. One more try if nothing
	// at all was fit to say.
	system, prompt := System(form), Prompt(s, form, level, c, f, now)
	// What it may not say: what it has said, and the lines the form already
	// has by heart - the two it was shown, and the rest of them.
	taken := append(append([]string{}, f.Past...), pet.Voice(form)...)
	var lines []string
	for try := 0; try < tries && len(lines) == 0; try++ {
		out, err := ask(system, prompt)
		if err != nil {
			break
		}
		lines = choose(pick(out, taken), recent(f.Trail, c), facts(s, c, f, now))
		// Each try is a request of its own: it has not seen what it wrote the
		// first time, so it cannot be told "not those".
		prompt += fmt.Sprintf("\n\nMás cortas todavía: ninguna puede pasar de %d caracteres.", MaxRunes)
	}
	if len(lines) == 0 {
		write(path, func(f *File) bool { f.Fails++; return true })
		return nil
	}
	write(path, func(f *File) bool {
		f.Lines, f.Form, f.At, f.SaidAs, f.Fails = lines, form, now.Unix(), mark, 0
		f.Past = append(f.Past, lines...)
		if len(f.Past) > pastKept {
			f.Past = f.Past[len(f.Past)-pastKept:]
		}
		return true
	})
	return lines
}

// echoes says three in four of a line's words are another line's. Every word
// counts, the small ones too: these are lines of six or seven words, and
// "respira. el repo no se va a ir" is "respira. el bug no se va a ir" with
// one of them changed.
func echoes(line, other string) bool {
	words := func(s string) []string {
		var out []string
		for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) }) {
			if utf8.RuneCountInString(w) >= 2 {
				out = append(out, w)
			}
		}
		return out
	}
	mine, theirs := words(line), map[string]bool{}
	for _, w := range words(other) {
		theirs[w] = true
	}
	if len(mine) < 4 {
		return strings.ToLower(line) == strings.ToLower(other)
	}
	same := 0
	for _, w := range mine {
		if theirs[w] {
			same++
		}
	}
	return same*4 >= len(mine)*3
}

// aside is the model talking about the job instead of doing it - "hmm, no:
// eso sería un recuento, prohibido", "aquí va la versión buena" - which it
// does out loud now and then, in lines short enough to pass for the pet's.
var aside = regexp.MustCompile(`\b(frases?|palabras|lista|recuento|encargo|versión|prohibid[oa]|caracteres|instrucci[oó]n(es)?|entorno|carpeta|directorio|statusline|mascota|tmp)\b|\b(sin|ni|no es|no hay|no tiene)( un| el)? (repo|repositorio|git)\b|\bya (la|lo|las|los) (dije|he dicho)\b|[()\[\]]`)

// garbled is what is not a sentence at all, which a small model at its lowest
// effort hands over about once in a hundred lines: "standard view",
// "ese-tsc-no-cierra-solo-caminando-lobo", a path. Under three words, a word
// that is three words hyphenated together, or a slash.
func garbled(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 3 || strings.ContainsAny(line, "/\\") {
		return true
	}
	// Cut off before its end: "sí, ya tiene su..."
	if strings.HasSuffix(line, "...") || strings.HasSuffix(line, "…") {
		return true
	}
	for _, w := range fields {
		if strings.Count(strings.Trim(w, "-"), "-") >= 2 {
			return true
		}
	}
	return false
}

// speaker is the model signing the line: "sabueso: lo que huelo es...".
var speaker = regexp.MustCompile(`^[\p{L}]{3,14}:\s+`)

var bullet = regexp.MustCompile(`^(\d{1,2}[.)]|[-*•])\s+`)

// pick is the lines of an answer that can be said: clean, lower case like the
// repertoire, not said before and not twice in the same answer. All of them:
// which three are said is choose's business.
func pick(out string, past []string) []string {
	var lines []string
	for _, raw := range strings.Split(out, "\n") {
		// "1. ", "- ", "* ": told not to number them, and it will anyway.
		// Only a list marker: "9 commits y sigo" keeps its nine.
		raw = bullet.ReplaceAllString(strings.TrimSpace(raw), "")
		raw = speaker.ReplaceAllString(raw, "")
		line := strings.ToLower(Clean(raw))
		// "aquí van las frases:" is a heading, whatever it was told.
		if line == "" || strings.HasSuffix(line, ":") || aside.MatchString(line) || garbled(line) ||
			said(past, line) || said(lines, line) {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func said(past []string, line string) bool {
	for _, p := range past {
		if echoes(line, p) {
			return true
		}
	}
	return false
}

// lines is the fourteen lines of the tree as the model is offered them: the
// mark's id, and the kind of work it is the shape of. In the tree's own order.
var lines = []struct{ mark, work string }{
	{"surgeon", "cambios pequeños y precisos, un arreglo quirúrgico, commits limpios"},
	{"weaver", "cambios anchos que tocan muchos ficheros a la vez, refactors grandes"},
	{"monk", "sesión tranquila y ligera, leer y preguntar con poco ruido"},
	{"gardener", "documentación, README, comentarios, cuidar lo que ya existe"},
	{"bloodhound", "depurar: seguir el rastro de un fallo y reproducirlo antes de arreglarlo"},
	{"exterminator", "tests: escribirlos, pasarlos, dejar la suite en verde"},
	{"cartographer", "planificar: repartir un trabajo largo en tareas y recorrerlas"},
	{"oracle", "explorar y diseñar antes de tocar código, investigar, decidir"},
	{"bolt", "algo rápido: entrar, un cambio corto y salir"},
	{"sniper", "una sola tarea con una sola herramienta, directo al blanco"},
	{"ox", "sesión muy larga, horas de trabajo seguido sobre lo mismo"},
	{"mole", "cavar hondo en un mismo sitio: un módulo, un fichero, una y otra vez"},
	{"gremlin", "tocar el sistema: instalar, configurar, scripts, automatizar a lo bruto"},
	{"kraken", "demasiadas cosas a la vez, el contexto desbordado"},
}

// chooseMark puts the rules' doubt to the model: which of these few lines of
// the tree fits the session. "" if the answer is not one of them.
func chooseMark(c Context, f File, among, taken []string) string {
	offered := map[string]bool{}
	for _, mark := range among {
		offered[mark] = true
	}
	var b strings.Builder
	b.WriteString("Formas posibles:\n")
	for _, l := range lines {
		if offered[l.mark] {
			fmt.Fprintf(&b, "- %s (%s): %s\n", l.mark, pet.Name(l.mark), l.work)
		}
	}
	b.WriteString("\nEsta sesión:\n")
	if c.Repo != "" {
		b.WriteString("- repo: " + c.Repo + "\n")
	}
	if len(f.Trail) > 0 {
		b.WriteString("- últimas herramientas, de la más antigua a la más reciente: " + strings.Join(f.Trail, ", ") + "\n")
	} else if step := c.step(); step != "" {
		b.WriteString("- lo último que ha hecho: " + step + "\n")
	}
	if c.Label != "" {
		b.WriteString("- el contexto está: " + pet.Name(c.Label) + "\n")
	}
	if c.Active >= 60 {
		fmt.Fprintf(&b, "- lleva %s de trabajo\n", span(c.Active))
	}
	if len(taken) > 0 {
		b.WriteString("\nOtras sesiones abiertas llevan: " + strings.Join(taken, ", ") +
			". Si dudas entre dos, elige la que no lleve ninguna.\n")
	}
	b.WriteString("\nResponde solo con el identificador.")

	out, err := ask("Eliges qué forma toma la mascota de la statusline de un programador, según el trabajo "+
		"que está haciendo esta sesión. Respondes con UNA palabra: el identificador de la lista, tal cual, "+
		"sin explicar nada.", b.String())
	if err != nil {
		return ""
	}
	if picked := parseMark(out); offered[picked] {
		return picked
	}
	return ""
}

// parseMark finds the one line the answer names, by id or by its Spanish name.
// An answer that names none, or two, is no answer.
func parseMark(out string) string {
	words := strings.FieldsFunc(strings.ToLower(firstLine(out)), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	found := ""
	for _, word := range words {
		for _, l := range lines {
			if word == l.mark || word == pet.Name(l.mark) {
				if found != "" && found != l.mark {
					return ""
				}
				found = l.mark
			}
		}
	}
	return found
}

// others is the lines the other live sessions wear, so that they can be
// spread: four windows open should not be four of the same creature because
// each was read alone. The rules weigh by it, and the model is told.
func others(mine string, now time.Time) []string {
	paths, _ := filepath.Glob(filepath.Join(session.TmpDir(), session.Prefix+"voice-*"))
	seen := map[string]bool{}
	var out []string
	for _, path := range paths {
		if path == mine || strings.HasSuffix(path, ".lock") {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || now.Sub(info.ModTime()) > ShapeEvery {
			continue // a session that has not used a tool in a while
		}
		mark := Load(path).Mark
		if _, ok := pet.Unlocks[mark]; ok && !seen[mark] {
			seen[mark] = true
			out = append(out, mark)
		}
	}
	sort.Strings(out)
	return out
}

// temper is who each form is when it talks: what it cares about and how it
// says so. Forty-one of them, because forty-one creatures that all tease in
// the same voice are one creature in forty-one costumes: a mark and its title
// are the same line of the tree at two ages and still not the same person -
// the cirujano is a surgeon, the bisturí is the blade - and the trades above
// them are somebody else again.
var temper = map[string]string{
	"spark": "recién nacida y descarada; todo le parece enorme y nuevo, pregunta sin parar y no tiene vergüenza",

	"pattern": "maestra de escuela con regla; adora el orden, las rutinas y hacer las cosas una detrás de otra",
	"probe":   "científica curiosa; no se cree nada sin experimento y contesta a todo con otra pregunta",
	"ember":   "fogosa e impulsiva; primero actúa y luego piensa, y todo le parece lento y frío",

	"refactor":  "corrector de estilo implacable; odia lo duplicado y los malos nombres, y tacha sin piedad",
	"tidy":      "maniático de la limpieza; ve polvo y trastos por todas partes y los recoge con retintín",
	"bughunter": "trampero de bichos; habla de huellas, cepos y trofeos, paciente y algo fanfarrón",
	"architect": "arquitecto con los planos bajo el brazo; todo es estructura, cimientos y qué carga con qué",
	"sprinter":  "atleta de cien metros con el cronómetro en la mano; vive de tacos de salida y líneas de meta",
	"marathon":  "entrenadora de fondo; dosifica, hidrata y anima con dureza cariñosa",
	"feral":     "animal sin domesticar; no reconoce reglas ni permisos y enseña los dientes por gusto",

	"surgeon":      "cirujano meticuloso y frío, humor de quirófano; adora el corte limpio y le ofende lo que sobra",
	"weaver":       "tejedor paciente y sentencioso; ve hilos y nudos en todo y refunfuña si le enredan la madeja",
	"monk":         "monje sereno e irónico; aconseja respirar y suelta proverbios que se acaba de inventar",
	"gardener":     "jardinero tierno y regañón; habla de podar, regar y malas hierbas, y defiende lo viejo que da fruto",
	"bloodhound":   "detective de novela negra, obsesivo; lo olfatea todo y sospecha de todos, empezando por ti",
	"exterminator": "veterano chulesco de la guerra contra las plagas; cuenta bajas y desconfía del verde fácil",
	"cartographer": "cartógrafo pedante; todo son rutas, desvíos y un «ya te lo dije» siempre a punto",
	"oracle":       "adivino teatral; profetiza obviedades con muchísima solemnidad y humo",
	"bolt":         "puro nervio eléctrico; frases cortísimas, chispazos, se aburre en diez segundos",
	"sniper":       "tirador parco y certero; desprecia el ruido y cada palabra de más",
	"ox":           "buey de labranza, sufrido y quejica; cuenta las horas con resignación y sorna",
	"mole":         "topo miope y testarudo; lleva horas en el mismo túnel y le parece el mejor sitio del mundo",
	"gremlin":      "duende saboteador; el caos le encanta y te anima a tocar justo lo que no deberías",
	"kraken":       "pulpo desbordado y dramático; no le quedan tentáculos y pide auxilio con guasa",

	"scalpel":   "el bisturí mismo: frío, afilado y altivo; solo corta, y desprecia la mano que tiembla",
	"loom":      "máquina vieja y constante; habla de pasadas, tramas y urdimbres, y exige que le traigan hilo",
	"abbot":     "abad paternal y con autoridad; te llama hijo mío y cita la regla del claustro con sorna",
	"forest":    "bosque antiguo que habla despacio y en plural; todo es raíces, estaciones y paciencia",
	"wolf":      "lobo viejo de pocas palabras; habla de manada, acecho y paciencia de depredador",
	"wasp":      "avispa irritable y territorial; zumba, amenaza con picar y no perdona un descuido",
	"atlas":     "titán cansado que carga con el plan entero; solemne, sufrido y un poco mártir",
	"sphinx":    "esfinge que guarda la puerta; solo habla en acertijos y disfruta de que no los resuelvas",
	"storm":     "tormenta con prisa y delirios de grandeza; llega, lo empapa todo y se va",
	"falcon":    "halcón altivo que lo ve todo desde arriba; un solo picado, y desprecio por quien aletea",
	"mammoth":   "mamut prehistórico; lento y enorme, ha visto pasar eras y lo compara todo con el hielo",
	"worm":      "gusano humilde y constante; vive dentro de la manzana y no entiende a quien quiere salir",
	"devil":     "diablo de despacho; tienta con contratos, letra pequeña y atajos que se pagan luego",
	"leviathan": "monstruo abisal, inmenso y soñoliento; todo le cabe, hasta que deja de caber",

	"phoenix": "ave que murió de hambre y ha vuelto; rencorosa con gracia, habla de cenizas y de renacer",
	"chimera": "dos cabezas que nunca están de acuerdo; se interrumpen y se llevan la contraria",
}

// System is the character, and how it sounds.
//
// Mostly examples. The first cut of this was a paragraph of rules - no
// figures, no lists, do not describe, do not mention the directory - and what
// came back was the rules: lines that described, lines that recited, and now
// and then the model explaining out loud which rule it was obeying. A small
// model copies a voice far better than it follows a brief, so it is handed
// the form's own five asides and asked for five more like them.
func System(form string) string {
	who := temper[form]
	if who == "" {
		who = temper[pet.Root]
	}
	lines := pet.Asides[form]
	if len(lines) == 0 {
		lines = pet.Asides[pet.Root]
	}
	var b strings.Builder
	b.WriteString("Eres «" + pet.Name(form) + "», la mascota pixelada que vive en la statusline de un programador " +
		"y le mira trabajar. Eres: " + who + ".\n\n" +
		"Le hablas a él, de tú, por encima del hombro. Así suenas - estas ya se las has dicho, " +
		"y no valen ni retocadas:\n")
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	b.WriteString("\nTe dirán qué está haciendo. Escribe CINCO frases que no hayas dicho nunca, con esa misma voz " +
		"y sobre lo que hace AHORA, una por línea y " +
		"nada más. En minúsculas, con tus imágenes y no las de otro. MUY cortas, como las de arriba: ocho " +
		"palabras como mucho y nunca más de 45 caracteres; la que se pasa se tira. " +
		"Cada una le dice algo a él sobre lo que hace: una pulla, un consejo, una queja, una pregunta. " +
		"Sin cifras y sin contar lo que ya sabe.")
	return b.String()
}

var weekdays = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}

// Prompt is the moment: two or three things, in words. Everything that was
// ever put on this list got said, every turn - the figures first, then "muchos
// commits" in every session of the day - so the list is short, what is on it
// is what he is doing right now, and the background takes one line that
// changes from turn to turn.
func Prompt(s *pet.State, form string, level int, c Context, f File, now time.Time) string {
	var b strings.Builder
	for _, fact := range facts(s, c, f, now) {
		b.WriteString(fact + "\n")
	}
	if len(f.Past) > 0 {
		b.WriteString("\nYa le has dicho, no lo repitas:\n")
		for _, line := range f.Past {
			b.WriteString(line + "\n")
		}
	}
	// The one rule it is told twice. Without it the voice was at its best and
	// four lines in five ran to sixty characters and more, which is a line
	// nobody reads: the band drops it.
	b.WriteString("\nCinco frases, de ocho palabras como mucho cada una. Una sola cosa por frase, y cada frase " +
		"sobre una cosa distinta: no enumeres ni te repitas.")
	return b.String()
}

// facts is what the model is told about the moment, a sentence each.
func facts(s *pet.State, c Context, f File, now time.Time) []string {
	var out []string
	where := ""
	if c.Repo != "" {
		where = " en " + c.Repo
	}
	if touched := recent(f.Trail, c); len(touched) > 0 {
		out = append(out, "Ahora mismo"+where+" anda con: "+strings.Join(touched, ", ")+".")
	} else if where != "" {
		out = append(out, "Ahora mismo trabaja"+where+".")
	}
	// What he can do something about is always said.
	switch c.Label {
	case "tired":
		out = append(out, "El contexto de la sesión va cargado.")
	case "drowning":
		out = append(out, "El contexto de la sesión está a punto de reventar.")
	}
	if s.Hunger >= pet.HungerWarn {
		out = append(out, "Tú tienes hambre: hace mucho que no te da de comer.")
	}
	var background []string
	if when := notable(now); when != "" {
		background = append(background, "Es "+when+".")
	}
	background = append(background, meals(s, now)...)
	switch {
	case c.Active >= 4*3600:
		background = append(background, "Lleva muchas horas seguidas en esta sesión.")
	case c.Active >= 90*60:
		background = append(background, "Lleva un par de horas en esta sesión.")
	case c.Active > 0 && c.Active < 15*60:
		background = append(background, "Acaba de sentarse.")
	}
	if s.Streak >= 3 {
		background = append(background, "Lleva varios días seguidos viniendo.")
	}
	if n := len(background); n > 0 {
		out = append(out, background[int(now.Unix()/int64(Every/time.Second))%n])
	}
	return out
}

// recent is what the session has just had its hands on, without the tool
// names: "auth.py", "pytest". The last three, each once. Three and not more:
// given four it said all four, in a row, and called it a remark.
func recent(trail []string, c Context) []string {
	if len(trail) == 0 && c.step() != "" {
		trail = []string{c.step()}
	}
	var out []string
	seen := map[string]bool{}
	for i := len(trail) - 1; i >= 0 && len(out) < 3; i-- {
		_, what, ok := strings.Cut(trail[i], " ")
		if !ok || seen[what] {
			continue // a tool with nothing named: "Grep" says little
		}
		seen[what] = true
		out = append([]string{what}, out...)
	}
	return out
}

// choose is the best `wanted` of the lines that could be said: the ones that
// are a remark and not an inventory. Asked not to, the model still hands back
// "apt, compose, docker... ¿y si lo haces a ciegas?" and "dos horas de sesión
// y el contexto a punto de reventar" - the list it was given, and the fact it
// was given, said back. Those are dropped; of the rest, the ones that name
// one thing go before the ones that name two.
func choose(lines, things, told []string) []string {
	type ranked struct {
		line   string
		weight int
	}
	var keep []ranked
	for _, line := range lines {
		named := naming(line, things)
		switch {
		case named >= 3, named == 2 && listed(line, things):
			continue // an inventory
		case parrots(line, told):
			continue // the fact, said back
		case invented(line, told):
			continue // a count or a date nobody gave it: "ocho horas de surco"
		}
		keep = append(keep, ranked{line, named})
	}
	sort.SliceStable(keep, func(i, j int) bool { return keep[i].weight < keep[j].weight })

	// One line a subject. Told that he has been making commits all day, four
	// of its five lines are about commits, and three of those in a row is one
	// remark made three times. A line about nothing on the list - pure
	// character - has no subject and never clashes.
	subjects := subjectsOf(things, told)
	var out, spare []string
	used := map[string]bool{}
	for _, k := range keep {
		clash := false
		mine := about(k.line, subjects)
		for _, subject := range mine {
			clash = clash || used[subject]
		}
		if clash {
			spare = append(spare, k.line)
			continue
		}
		for _, subject := range mine {
			used[subject] = true
		}
		out = append(out, k.line)
	}
	// Better a second line on the same subject than a turn with nothing to say.
	out = append(out, spare...)
	if len(out) > wanted {
		out = out[:wanted]
	}
	return out
}

// counted is a number, or a when: the thing the model was not given and
// supplies anyway - "ocho compactaciones", "a las cinco de la mañana", "la
// migración de ayer", "plan.md lleva días igual". Any of them is fine when
// the facts said so, and made up when they did not; see invented.
//
// "dos" is let through. It is as often a figure of speech - dos cabezas, dos
// cosas a la vez - as a count, and dropping it took good lines with it.
var counted = regexp.MustCompile(`\d|\b(tres|cuatro|cinco|seis|siete|ocho|nueve|diez|once|doce|quince|veinte|treinta|cuarenta|cincuenta|cien|mil)\b`)

var dated = regexp.MustCompile(`\b(ayer|anteayer|anoche|días|semanas?|meses|años)\b`)

// invented says a line states a count or a when that nothing it was told
// carries.
func invented(line string, told []string) bool {
	heard := strings.ToLower(strings.Join(told, " "))
	for _, re := range []*regexp.Regexp{counted, dated} {
		for _, m := range re.FindAllString(line, -1) {
			if !strings.Contains(heard, m) {
				return true
			}
		}
	}
	return false
}

// filler is the words of a fact that are not what it is about.
var filler = map[string]bool{
	"ahora": true, "mismo": true, "anda": true, "trabaja": true, "lleva": true, "hace": true,
	"está": true, "tiene": true, "mucho": true, "muchas": true, "muchos": true, "varias": true,
	"varios": true, "alguna": true, "algún": true, "seguidas": true, "seguidos": true, "punto": true,
}

// stem is a word cut short enough for "commits" to be "commit" and
// "compactado" to be "compactar". Crude, and enough for seven-word lines.
func stem(w string) string {
	if r := []rune(w); len(r) > 5 {
		return string(r[:5])
	}
	return w
}

// subjectsOf is what a turn's lines can be about: the things in hand, and
// the words of the facts that carry them.
func subjectsOf(things, told []string) map[string]bool {
	out := map[string]bool{}
	for _, thing := range things {
		for _, s := range stems(thing) {
			out[stem(s)] = true
		}
	}
	for _, fact := range told {
		for _, w := range words(fact) {
			if utf8.RuneCountInString(w) >= 5 && !filler[w] {
				out[stem(w)] = true
			}
		}
	}
	return out
}

// about is which of the subjects a line touches.
func about(line string, subjects map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, w := range words(line) {
		if s := stem(w); subjects[s] && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// stems is the ways a thing can be named in a line: "auth.py" and "auth".
func stems(thing string) []string {
	thing = unaccent.Replace(strings.ToLower(thing))
	if base, _, ok := strings.Cut(thing, "."); ok && base != "" {
		return []string{thing, base}
	}
	return []string{thing}
}

// unaccent is for telling what a line names, not for anything shown: the
// file is guia.md and the pet, which can spell, says "la guía".
var unaccent = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u")

func words(s string) []string {
	return strings.FieldsFunc(unaccent.Replace(strings.ToLower(s)), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-')
	})
}

// naming is how many of the things the session is handling a line names.
func naming(line string, things []string) int {
	in := map[string]bool{}
	// A program whose name is a Spanish word is not named by the word: "tú
	// tecleas y yo cargo" is a buey complaining, not a remark about cargo.
	homonym := map[string]bool{"cargo": true, "sed": true, "ve": true, "es": true, "top": true}
	for _, w := range words(line) {
		in[w] = true
	}
	n := 0
	for _, thing := range things {
		if homonym[strings.ToLower(thing)] {
			continue
		}
		for _, stem := range stems(thing) {
			if in[strings.Trim(stem, ".")] || strings.Contains(unaccent.Replace(strings.ToLower(line)), stem) && strings.Contains(stem, ".") {
				n++
				break
			}
		}
	}
	return n
}

// listed says two things are named side by side - "auth.py y pytest", "cargo,
// parser" - which is a list of two, as against "pytest verde y auth roto",
// which is a sentence about both.
func listed(line string, things []string) bool {
	ws := words(strings.NewReplacer(".py", "", ".rs", "", ".go", "", ".ts", "", ".js", "", ".yml", "", ".md", "", ".css", "", ".sql", "").Replace(strings.ToLower(line)))
	is := map[string]bool{}
	for _, thing := range things {
		for _, stem := range stems(thing) {
			is[stem] = true
		}
	}
	// "el readme, la guía y el changelog" is a list with its articles on.
	var bare []string
	for _, w := range ws {
		switch w {
		case "el", "la", "los", "las", "un", "una", "tu", "tus", "ese", "esa":
		default:
			bare = append(bare, w)
		}
	}
	ws = bare
	for i, w := range ws {
		if !is[w] {
			continue
		}
		if i+1 < len(ws) && is[ws[i+1]] {
			return true
		}
		if i+2 < len(ws) && (ws[i+1] == "y" || ws[i+1] == "e" || ws[i+1] == "con") && is[ws[i+2]] {
			return true
		}
	}
	return false
}

// parrots says nearly every word of a line that means something is in what
// the model was told: the fact handed back, with nothing added.
func parrots(line string, told []string) bool {
	heard := map[string]bool{}
	for _, fact := range told {
		for _, w := range words(fact) {
			heard[w] = true
		}
	}
	// By the first four letters: "paras" for "para", "commit" for "commits".
	short := func(w string) string {
		if r := []rune(w); len(r) > 4 {
			return string(r[:4])
		}
		return w
	}
	for w := range heard {
		heard[short(w)] = true
	}
	mine, same := 0, 0
	for _, w := range words(line) {
		if utf8.RuneCountInString(w) < 4 {
			continue
		}
		mine++
		if heard[short(w)] {
			same++
		}
	}
	return mine >= 3 && same*5 >= mine*4 // four in five
}

// notable is the day and the hour when they are worth a remark, and "" the
// rest of the week: a Tuesday at eleven is not news.
func notable(now time.Time) string {
	day, hour := now.Weekday(), now.Hour()
	switch {
	case hour < 6:
		return weekdays[day] + " de madrugada"
	case hour >= 22:
		return weekdays[day] + " por la noche, tarde"
	case day == time.Saturday || day == time.Sunday:
		return weekdays[day] + ", fin de semana"
	case day == time.Monday && hour < 12:
		return "lunes por la mañana"
	case day == time.Friday && hour >= 15:
		return "viernes por la tarde"
	}
	return ""
}

// meals is today's log in words, one piece per kind of meal: "ha hecho
// muchos commits hoy".
func meals(s *pet.State, now time.Time) []string {
	if s.LogDay != pet.Today(now) {
		return nil
	}
	count := map[string]int{}
	for _, e := range s.Log {
		if e.XP > 0 {
			count[e.Event]++
		}
	}
	var out []string
	for _, k := range []struct{ event, one, many string }{
		{"commit", "Hoy ha hecho algún commit.", "Hoy no para de hacer commits."},
		{"tests", "Hoy ha dejado alguna suite en verde.", "Hoy lleva varias suites en verde."},
		{"compact", "Hoy ha compactado el contexto.", "Hoy ha compactado el contexto muchas veces."},
		{"task", "Hoy ha cerrado alguna tarea del plan.", "Hoy ha cerrado muchas tareas del plan."},
	} {
		switch n := count[k.event]; {
		case n >= 4:
			out = append(out, k.many)
		case n > 0:
			out = append(out, k.one)
		}
	}
	return out
}

func span(seconds int64) string {
	h, m := seconds/3600, seconds%3600/60
	if h == 0 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %d min", h, m)
}

// Clean turns what the model sent into something that can go on a terminal, or
// into "" if it cannot: one line, printable, no wrapping quotes, short enough.
//
// The length is a refusal and not a cut. A line chopped at 57 characters is a
// sentence without its end, and the repertoire is right there to say instead.
func Clean(raw string) string {
	line := firstLine(raw)
	// Refused whole, not scrubbed: an escape sequence without its ESC is
	// still "[2J" in the middle of the footer.
	for _, r := range line {
		if !unicode.IsPrint(r) {
			return ""
		}
	}
	line = strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "\"'`«»“”‘’"))
	if line == "" || utf8.RuneCountInString(line) > MaxRunes {
		return ""
	}
	return line
}

func firstLine(raw string) string {
	for _, l := range strings.Split(raw, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// ContextFrom reads the hook's payload and the session's facts.
func ContextFrom(payload map[string]any, facts session.Facts) Context {
	c := Context{Label: facts.Label, Active: facts.Active}
	c.Session, _ = payload["session_id"].(string)
	c.Session = session.SafeID(c.Session)
	if cwd, _ := payload["cwd"].(string); cwd != "" {
		c.Repo = name(filepath.Base(cwd))
	}
	c.Tool, _ = payload["tool_name"].(string)
	input, _ := payload["tool_input"].(map[string]any)
	if file, _ := input["file_path"].(string); file != "" {
		c.Target = name(filepath.Base(file))
	} else if command, _ := input["command"].(string); command != "" {
		c.Target = program(command)
	}
	return c
}

// name is a file or directory name fit to be shown to the model and kept in
// the trail, or "". A name is whatever somebody called a file in a repo that
// was cloned: it is not trusted to be short, or to be only a name.
func name(s string) string {
	if utf8.RuneCountInString(s) > 40 {
		return ""
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_.+@", r)) {
			return ""
		}
	}
	return s
}

// program is the name of what a command line runs, and nothing after it: the
// arguments are where the tokens and the passwords are.
func program(command string) string {
	// `cd somewhere && go test` is a go, not a cd.
	for strings.HasPrefix(strings.TrimSpace(command), "cd ") {
		_, rest, chained := strings.Cut(command, "&&")
		if !chained {
			break
		}
		command = rest
	}
	for _, word := range strings.Fields(command) {
		if strings.Contains(word, "=") {
			continue // FOO=bar in front of the command
		}
		word = filepath.Base(word)
		if len(word) > 24 {
			return ""
		}
		for _, r := range word {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
				return ""
			}
		}
		return word
	}
	return ""
}

const usage = "uso: ccpet voice {on|off|status|say} | ccpet voice shape {on|off}\n"

// Run is `ccpet voice ...`.
func Run(args []string, stdout io.Writer, statePath string, now time.Time) int {
	path := PathBeside(statePath)
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "on", "off":
		on := action == "on"
		if !update(path, func(f *File) bool { f.Enabled = on; return true }) {
			fmt.Fprintln(stdout, "no he podido escribir "+path)
			return 1
		}
		if on {
			if _, err := exec.LookPath("claude"); err != nil {
				fmt.Fprintln(stdout, "voz encendida, pero no encuentro `claude` en el PATH: no dirá nada nuevo")
				return 0
			}
			fmt.Fprintln(stdout, "voz encendida · haiku desempata la forma cuando las reglas dudan y escribe tres frases cada media hora de trabajo")
		} else {
			fmt.Fprintln(stdout, "voz apagada · la forma de cada sesión la eligen las reglas, y habla el repertorio")
		}
		return 0
	case "shape":
		// The per-session shapes, which need no model and are on by default.
		if len(args) < 2 || (args[1] != "on" && args[1] != "off") {
			fmt.Fprint(stdout, usage)
			return 2
		}
		fixed := args[1] == "off"
		if !update(path, func(f *File) bool { f.Fixed = fixed; return true }) {
			fmt.Fprintln(stdout, "no he podido escribir "+path)
			return 1
		}
		if fixed {
			fmt.Fprintln(stdout, "forma fija · todas las sesiones llevan la forma del árbol")
		} else {
			fmt.Fprintln(stdout, "forma por sesión · cada sesión lleva la que pide su trabajo")
		}
		return 0
	case "status":
		f := Load(path)
		if f.Fixed {
			fmt.Fprintln(stdout, "forma fija (la del árbol)")
		} else {
			fmt.Fprintln(stdout, "forma por sesión")
		}
		state := "apagada"
		if f.Enabled {
			state = "encendida"
		}
		fmt.Fprintln(stdout, "voz "+state)
		if !f.Enabled {
			fmt.Fprintln(stdout, "enciéndela con `ccpet voice on` (o /pet-voice on): usa haiku con tu sesión de claude")
		}
		// Each live session, with the shape it wears and what it last said.
		paths, _ := filepath.Glob(filepath.Join(session.TmpDir(), session.Prefix+"voice-*"))
		sort.Strings(paths)
		for _, p := range paths {
			if strings.HasSuffix(p, ".lock") {
				continue
			}
			// A session that died without a SessionEnd leaves its file for
			// the daily sweep; it is not a session any more.
			if info, err := os.Stat(p); err != nil || now.Sub(info.ModTime()) > 2*ShapeEvery {
				continue
			}
			sf := Load(p)
			if sf.Mark == "" && (len(sf.Lines) == 0 || !f.Enabled) {
				continue
			}
			shape := "-"
			if _, ok := pet.Unlocks[sf.Mark]; ok {
				shape = pet.Name(sf.Mark)
			}
			said := "aún no ha dicho nada"
			if !f.Enabled {
				said = "por reglas"
			} else if len(sf.Lines) > 0 {
				// Cleaned like the statusline cleans them: the file is on
				// disk and this goes to a terminal.
				var shown []string
				for _, line := range sf.Lines {
					if line = Clean(line); line != "" {
						shown = append(shown, line)
					}
				}
				said = time.Unix(sf.At, 0).Format("15:04") + " · " + strings.Join(shown, " / ")
			}
			fmt.Fprintf(stdout, "sesión · %-13s · %s\n", shape, said)
		}
		return 0
	case "say":
		// The hook's detached half, and the way to try it by hand. Works with
		// the voice off, on purpose: it is how one finds out what it sounds
		// like before switching it on.
		var c Context
		json.Unmarshal([]byte(os.Getenv(ctxEnv)), &c)
		// From the hook it asks only if the session's lines are spent; typed
		// by hand it always asks.
		lines := Say(statePath, c, now, os.Getenv(ChildEnv) == "")
		if len(lines) == 0 {
			fmt.Fprintln(stdout, "(sin frases: el modelo no ha respondido o ninguna cabía)")
			return 1
		}
		for _, line := range lines {
			fmt.Fprintln(stdout, line)
		}
		return 0
	}
	fmt.Fprint(stdout, usage)
	return 2
}
