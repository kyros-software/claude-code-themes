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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
