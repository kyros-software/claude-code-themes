package pet

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// The life file - ~/.claude/pet.json.
//
// Anyone can edit this file (or a full disk can corrupt it), so NOTHING is
// trusted until it has been through Load. A broken pet.json is a newborn pet,
// not a crash: if a string slips through where a number belongs, the whole
// statusline disappears.

// LogMax caps the day's log.
const LogMax = 40

// Secret is the level-5 form that is not on the tree, or empty. It marshals as
// null rather than "" so a file written here is byte-for-byte the file the
// Python wrote, and a user can move between the two mid-session.
type Secret string

// MarshalJSON writes null for the empty secret.
func (s Secret) MarshalJSON() ([]byte, error) {
	if s == "" {
		return []byte("null"), nil
	}
	return json.Marshal(string(s))
}

// UnmarshalJSON accepts null, "" and a name.
func (s *Secret) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		*s = ""
		return nil
	}
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return err
	}
	*s = Secret(name)
	return nil
}

// LogEntry is one meal.
type LogEntry struct {
	Event string `json:"q"`
	XP    int    `json:"xp"`
	At    int64  `json:"t"`
	Note  string `json:"n"`
}

// State is the whole life file. Field order matches the file written by the
// Python this replaces, so an upgrade in either direction is a no-op.
type State struct {
	XP         int   `json:"xp"`
	Hunger     int   `json:"hunger"`
	LastFed    int64 `json:"last_fed"`
	Streak     int   `json:"streak"`
	BestStreak int   `json:"best_streak"`
	LevelSeen  int   `json:"level_seen"`
	// StreakSeen is the streak last announced. The streak is a standing state,
	// not an event: read as one, a pet with three days or more said the same
	// "N días de racha" every five minutes, all day. It speaks when the number
	// moves, which is once a day. Absent from an older file: 0, so the first
	// refresh after an upgrade says it once and then holds its tongue.
	StreakSeen int    `json:"streak_seen"`
	HungerPeak int    `json:"hunger_peak"`
	FedAt      int64  `json:"fed_at"`
	AteAt      int64  `json:"ate_at"`
	LastDay    string `json:"last_day"`
	RepoDay    string `json:"repo_day"`
	LogDay     string `json:"log_day"`
	Secret     Secret `json:"secret"`

	// FormSeen is the highest rung the shape has stood on. In v2 it was the
	// floor that kept a broken streak from taking the shape down the tree; in
	// v3 the path (Branch) does that, and this is kept for older builds that
	// read the file, and for the v2 -> v3 migration, which rebuilds the path
	// from it.
	FormSeen string `json:"form_seen"`

	// Branch is the pet's path in this generation: the child taken at each
	// fork, keyed by the parent, marks and titles included -
	// {"spark": "pattern", "pattern": "tidy", "tidy": "monk", "monk": "abbot"}.
	//
	// v3: each link is written once, by Advance, when the pet crosses into
	// that fork, and never revised. The shape IS this path.
	Branch map[string]string `json:"branch"`

	// RepoRuns is, per repo, the last day a session was closed in it and how
	// many days running that makes. same_repo_days is the best live run among
	// them. It used to be ONE run, kept in RepoDay: closing a session in a
	// second repo the same day reset it to 1, so anybody who works in two
	// repos a day could never reach `mole`. Pruned on every close to the runs
	// that are still alive (today or yesterday), so it does not grow with
	// every repo ever opened.
	RepoRuns map[string]RepoRun `json:"repo_runs"`

	// v3 - see docs/design/v3-generaciones.md.
	//
	// Generation counts the pet's lives, from 1. A file without it is a v2
	// pet, and Load migrates it (see migrateV2).
	Generation int `json:"generation"`
	// StageBase is every counter as it stood when the current stage began, and
	// StageMax the per-stage maximum of the RecordMax counters. Together they
	// are State.Stage: what each habit did since the last evolution, which is
	// what decides the next fork.
	StageBase map[string]int `json:"stage_base"`
	StageMax  map[string]int `json:"stage_max"`
	// Album is every form the pet has worn, life after life.
	Album []AlbumEntry `json:"album"`
	// TitleDays counts the days of use spent wearing a title. At TitleStay the
	// generation closes and an egg is laid. See Hatch.
	TitleDays int `json:"title_days"`

	Counters map[string]int    `json:"counters"`
	Meals    map[string]int64  `json:"meals"`
	Log      []LogEntry        `json:"log"`
	DayMarks map[string]string `json:"day_marks"`

	// What the pet has said lately, so it does not repeat itself, and when.
	// See speech.go.
	Said   []string `json:"said"`
	SaidAt int64    `json:"said_at"`
}

// AlbumEntry is one form worn: which, in which life, and when it arrived.
type AlbumEntry struct {
	Form string `json:"form"`
	Gen  int    `json:"gen"`
	At   int64  `json:"at"`
}

// RepoRun is one repo's run of consecutive days with a session closed in it.
type RepoRun struct {
	Day string `json:"day"`
	Run int    `json:"run"`
}

// Clone is a State that shares nothing with the one it came from.
//
// `copy := *s` looks like a copy and is not: five of these fields are maps and
// slices, so the copy keeps pointing at the original's containers and a write
// through either one is a write through both. The panel builds a hypothetical
// pet to ask what the next level would turn it into, and today nothing on that
// path writes to a counter - so the shallow copy was correct by luck, and one
// added Bump away from silently editing the real pet while answering a
// question about an imaginary one.
//
// Cloning costs a few dozen map entries once per `/pet`, which is not a price
// worth reasoning about.
func (s *State) Clone() *State {
	out := *s
	out.Counters = make(map[string]int, len(s.Counters))
	for k, v := range s.Counters {
		out.Counters[k] = v
	}
	out.Meals = make(map[string]int64, len(s.Meals))
	for k, v := range s.Meals {
		out.Meals[k] = v
	}
	out.Branch = make(map[string]string, len(s.Branch))
	for k, v := range s.Branch {
		out.Branch[k] = v
	}
	out.StageBase = make(map[string]int, len(s.StageBase))
	for k, v := range s.StageBase {
		out.StageBase[k] = v
	}
	out.StageMax = make(map[string]int, len(s.StageMax))
	for k, v := range s.StageMax {
		out.StageMax[k] = v
	}
	out.Album = append([]AlbumEntry(nil), s.Album...)
	out.RepoRuns = make(map[string]RepoRun, len(s.RepoRuns))
	for k, v := range s.RepoRuns {
		out.RepoRuns[k] = v
	}
	out.DayMarks = make(map[string]string, len(s.DayMarks))
	for k, v := range s.DayMarks {
		out.DayMarks[k] = v
	}
	// LogEntry and string are values, so copying the backing array is enough.
	out.Log = append([]LogEntry(nil), s.Log...)
	out.Said = append([]string(nil), s.Said...)
	return &out
}

// New is a newborn pet.
func New() *State {
	return &State{Counters: map[string]int{}, Log: []LogEntry{},
		DayMarks: map[string]string{}, Said: []string{},
		Meals: map[string]int64{}}
}

// --- v1 (Spanish) -> v2 (English). Read once, written back translated. ------

var legacyFields = map[string]string{
	"hambre": "hunger", "comio": "last_fed", "racha": "streak",
	"mejor_racha": "best_streak", "nivel_visto": "level_seen",
	"hambre_tope": "hunger_peak", "feed_hoy": "fed_today",
	"ultimo_dia": "last_day", "repo_dia": "repo_day", "hoy_dia": "log_day",
	"hoy": "log", "marcas_dia": "day_marks", "contadores": "counters",
	"secreta": "secret",
}

var legacyCounters = map[string]string{
	"metodico": "methodical", "inquisitivo": "inquisitive", "impulsivo": "impulsive",
	"ctx_bajo": "ctx_low", "planes": "plans", "ctx_limite": "ctx_maxed",
	"sesiones_cortas": "short_sessions", "sesiones_largas": "long_sessions",
	"racha_diffs": "diff_streak", "commit_ancho": "widest_commit",
	"sesiones_bajo_40": "sessions_under_40", "dias_docs": "docs_days",
	"repro_antes_fix": "repro_before_fix", "racha_tests": "test_streak",
	"plan_entero": "longest_plan", "planes_antes_codigo": "plans_before_code",
	"sesiones_15min": "sessions_15min", "tarea_una_herramienta": "single_tool_tasks",
	"sesiones_4h": "sessions_4h", "dias_mismo_repo": "same_repo_days",
	"turnos_bypass": "bypass_turns", "sesiones_ctx100": "ctx100_sessions",
}

var legacyForms = map[string]string{
	"chispa": "spark", "pauta": "pattern", "sonda": "probe", "brasa": "ember",
	"pulcro": "tidy", "cazabugs": "bughunter", "arquitecto": "architect",
	"velocista": "sprinter", "maraton": "marathon", "salvaje": "feral",
	"cirujano": "surgeon", "tejedor": "weaver", "monje": "monk",
	"jardinero": "gardener", "sabueso": "bloodhound",
	"exterminador": "exterminator", "cartografo": "cartographer",
	"oraculo": "oracle", "relampago": "bolt", "francotirador": "sniper",
	"buey": "ox", "topo": "mole", "fenix": "phoenix", "quimera": "chimera",
}

var legacyEvents = map[string]string{"tarea": "task", "reventon": "overflow"}

// asInt is the one gate every number passes. It matches the Python it replaces:
// a numeric string counts, a bool never does, and NaN/Inf are not numbers.
func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0
		}
		return int(n)
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0
		}
		return int(f)
	}
	return 0
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Path is where the life file lives.
func Path() string { return filepath.Join(config.Dir(), "pet.json") }

// Load never fails and never hands back a field with the wrong type.
func Load(path string) *State {
	s := New()
	raw, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil || doc == nil {
		return s
	}

	// Rename v1 keys. Unknown keys are left alone - a counter this version does
	// not know about is still somebody's progress.
	flat := make(map[string]any, len(doc))
	for k, v := range doc {
		if newKey, ok := legacyFields[k]; ok {
			k = newKey
		}
		flat[k] = v
	}

	s.XP = asInt(flat["xp"])
	s.Hunger = asInt(flat["hunger"])
	s.LastFed = int64(asInt(flat["last_fed"]))
	s.Streak = asInt(flat["streak"])
	s.BestStreak = asInt(flat["best_streak"])
	s.LevelSeen = asInt(flat["level_seen"])
	s.StreakSeen = asInt(flat["streak_seen"])
	s.HungerPeak = asInt(flat["hunger_peak"])
	// fed_today (a daily counter) is gone: /feed is a cooldown now, and an old
	// key just falls through here unread.
	s.FedAt = int64(asInt(flat["fed_at"]))
	// LastFed is the HUNGER CLOCK - DecayHunger walks it forward hour by hour -
	// so it cannot answer "when did it last eat": a pet abandoned for two days
	// read as "comió hace 0m". AteAt only moves when something is actually
	// eaten. An older file has no ate_at and falls back to last_fed.
	s.AteAt = int64(asInt(flat["ate_at"]))
	if s.AteAt == 0 {
		s.AteAt = int64(asInt(flat["last_fed"]))
	}
	s.SaidAt = int64(asInt(flat["said_at"]))
	s.LastDay = asString(flat["last_day"])
	s.RepoDay = asString(flat["repo_day"])
	s.Generation = asInt(flat["generation"])
	s.TitleDays = asInt(flat["title_days"])
	for key, dst := range map[string]*map[string]int{"stage_base": &s.StageBase, "stage_max": &s.StageMax} {
		if m, ok := flat[key].(map[string]any); ok {
			*dst = make(map[string]int, len(m))
			for k, v := range m {
				(*dst)[k] = asInt(v)
			}
		}
	}
	if album, ok := flat["album"].([]any); ok {
		for _, raw := range album {
			e, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			form := asString(e["form"])
			// Only forms that exist: a renamed or hand-written name would draw
			// as the root, and the album would be lying about it.
			if _, known := Sprites[form]; !known {
				continue
			}
			s.Album = append(s.Album, AlbumEntry{Form: form, Gen: asInt(e["gen"]), At: int64(asInt(e["at"]))})
		}
	}
	s.LogDay = asString(flat["log_day"])

	if secret := asString(flat["secret"]); secret != "" {
		if translated, ok := legacyForms[secret]; ok {
			secret = translated
		}
		if _, ok := Sprites[secret]; ok {
			s.Secret = Secret(secret)
		}
	}

	// The rung the shape stands on, through the same gate the secret goes
	// through: a v1 name is translated, and a name that is not a form at all is
	// dropped rather than kept. Tier would read an unknown name as 0 and let it
	// past harmlessly, but a floor is exactly the field where "harmless
	// nonsense" should not be stored in the first place.
	if form := asString(flat["form_seen"]); form != "" {
		if translated, ok := legacyForms[form]; ok {
			form = translated
		}
		if _, ok := Sprites[form]; ok {
			s.FormSeen = form
		}
	}

	// The path, through a stricter gate than the rung: an entry only survives
	// if the key is a form the tree really has AND the value is one of its own
	// children. A pair that is not a real edge would cut the path short.
	if runs, ok := flat["repo_runs"].(map[string]any); ok {
		for repo, v := range runs {
			r, ok := v.(map[string]any)
			if !ok || repo == "" {
				continue
			}
			day, run := asString(r["day"]), asInt(r["run"])
			if day == "" || run < 1 {
				continue
			}
			if s.RepoRuns == nil {
				s.RepoRuns = map[string]RepoRun{}
			}
			s.RepoRuns[repo] = RepoRun{Day: day, Run: run}
		}
	}

	if held, ok := flat["branch"].(map[string]any); ok {
		for parent, raw := range held {
			child := asString(raw)
			if translated, ok := legacyForms[child]; ok {
				child = translated
			}
			if translated, ok := legacyForms[parent]; ok {
				parent = translated
			}
			for _, kid := range Tree[parent] {
				if kid != child {
					continue
				}
				if s.Branch == nil {
					s.Branch = map[string]string{}
				}
				s.Branch[parent] = child
			}
		}
	}

	if counters, ok := flat["counters"].(map[string]any); ok {
		for k, v := range counters {
			if newKey, ok := legacyCounters[k]; ok {
				k = newKey
			}
			s.Counters[k] = asInt(v)
		}
	}

	if marks, ok := flat["day_marks"].(map[string]any); ok {
		for k, v := range marks {
			if newKey, ok := legacyCounters[k]; ok {
				k = newKey
			}
			if day, ok := v.(string); ok {
				s.DayMarks[k] = day
			}
		}
	}

	// One clock per food. FedAt was the only one there was, which was fine
	// while /feed was the only food on a cooldown; a second one would have
	// shared the timestamp and the two would have gagged each other.
	if meals, ok := flat["meals"].(map[string]any); ok {
		for name, v := range meals {
			s.Meals[name] = int64(asInt(v))
		}
	}
	if s.Meals["feed"] == 0 && s.FedAt != 0 {
		s.Meals["feed"] = s.FedAt
	}

	if said, ok := flat["said"].([]any); ok {
		for _, item := range said {
			if line, ok := item.(string); ok {
				s.Said = append(s.Said, line)
			}
		}
		if len(s.Said) > SaidMemory {
			s.Said = s.Said[len(s.Said)-SaidMemory:]
		}
	}

	if log, ok := flat["log"].([]any); ok {
		for _, item := range log {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			event := asString(entry["q"])
			if translated, ok := legacyEvents[event]; ok {
				event = translated
			}
			s.Log = append(s.Log, LogEntry{
				Event: event,
				XP:    asInt(entry["xp"]),
				At:    int64(asInt(entry["t"])),
				Note:  asString(entry["n"]),
			})
		}
		if len(s.Log) > LogMax {
			s.Log = s.Log[len(s.Log)-LogMax:]
		}
	}
	return s
}

// Save writes a state to disk atomically - temp file in the same directory,
// then rename - and WITHOUT taking the lock.
//
// It is not the way to change the pet. Load, mutate, Save is the shape that
// LOSES UPDATES: several sessions share this file, and because Save writes the
// whole State, the loser of a race does not merely lose its own increment, it
// puts back everything it read. Measured: 100 meals fed concurrently landed as
// 72 XP instead of 800. pet.Update is the replacement, and lock_test.go fails
// if production code calls this directly.
//
// The rename keeps its own promise regardless: nobody ever reads half a json.
// That was always true, and it was never the same thing as not losing writes.
func Save(s *State, path string) bool {
	// The rung, before anything else, because it is not decoration: it is a
	// field that has to agree with the state being written, exactly like the
	// nil containers below.
	//
	// Doing it here rather than at each call site is the whole point. Six
	// different paths persist this file - the statusline, a bubble, an
	// overflow, `ccpet count`, `ccpet <food>`, the session close - and only two
	// of them had any reason to think about shapes. With the statusline
	// switched off a pet could stand at rung 6 for weeks with form_seen never
	// written at all, and then one blown context dropped it to rung 3 AND
	// recorded the fall as the truth, so the floor came into existence pointing
	// at the wrong rung. The bug is not that a path forgot; it is that
	// forgetting was possible.
	//
	// RememberForm only ever raises, so a save that happens to catch the pet
	// mid-fall - book() writes right after an overflow has broken a streak -
	// cannot write the fall down.
	//
	// v3: the forks the level has reached are decided and written here for the
	// same reason - six paths persist this file and a fork must not depend on
	// which one ran. Advance also books each new form in the album.
	now := time.Now().Unix()
	if s.Generation < 1 {
		s.Generation = 1
	}
	Advance(s, now)
	if form, _ := CurrentForm(s); form != "" {
		RememberForm(s, form)
		// A secret is worn, not walked to, so the album learns it here.
		if secretForm(form) {
			s.Collect(form, now)
		}
	}
	if s.Counters == nil {
		s.Counters = map[string]int{}
	}
	if s.Log == nil {
		s.Log = []LogEntry{}
	}
	if s.DayMarks == nil {
		s.DayMarks = map[string]string{}
	}
	if s.Said == nil {
		s.Said = []string{}
	}
	if s.Meals == nil {
		s.Meals = map[string]int64{}
	}
	if s.Branch == nil {
		s.Branch = map[string]string{}
	}
	if s.RepoRuns == nil {
		s.RepoRuns = map[string]RepoRun{}
	}
	if s.StageBase == nil {
		s.StageBase = map[string]int{}
	}
	if s.StageMax == nil {
		s.StageMax = map[string]int{}
	}
	if s.Album == nil {
		s.Album = []AlbumEntry{}
	}
	dir := filepath.Dir(path)
	if os.MkdirAll(dir, 0o755) != nil {
		return false
	}
	tmp, err := os.CreateTemp(dir, ".pet-*.json")
	if err != nil {
		return false
	}
	name := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(s); err != nil {
		tmp.Close()
		os.Remove(name)
		return false
	}
	if tmp.Close() != nil || os.Rename(name, path) != nil {
		os.Remove(name)
		return false
	}
	return true
}

// Today is the local calendar day of a timestamp.
func Today(at time.Time) string { return at.Format("2006-01-02") }

// Yesterday is the calendar day before it.
//
// AddDate and not Add(-24h): a day is not always 24 hours. On the night the
// clocks go back the day has 25, and at 23:30 subtracting 24 hours lands on
// the same date - "yesterday" was today, and every day-run that asks (the
// streak, docs_days, same_repo_days) reset to 1. On the night they go forward
// it has 23, and at 00:30 it skipped a day the other way.
func Yesterday(at time.Time) string { return at.AddDate(0, 0, -1).Format("2006-01-02") }

// Bump adds. For counters that count occurrences.
func (s *State) Bump(counter string, n int) {
	if s.Counters == nil {
		s.Counters = map[string]int{}
	}
	s.Counters[counter] += n
}

// RecordMax keeps the maximum. For counters that measure a record, not a total:
// "a refactor touching 10+ files" is the widest one, not the sum of them.
func (s *State) RecordMax(counter string, value int) {
	if s.Counters == nil {
		s.Counters = map[string]int{}
	}
	if value > s.Counters[counter] {
		s.Counters[counter] = value
	}
	if s.StageMax == nil {
		s.StageMax = map[string]int{}
	}
	if value > s.StageMax[counter] {
		s.StageMax[counter] = value
	}
}

// maxCounters keep the largest value ever seen, so their stage value is the
// largest seen this stage. See RecordMax.
var maxCounters = map[string]bool{"widest_commit": true, "longest_plan": true}

// Stage is what a counter did since the current stage began.
//
// For a counter that only grows it is the difference from StageBase. A streak
// that broke since (test_streak on a red, diff_streak on a blown context) is
// below its base, and then everything it holds was built this stage.
//
// Runs of days (same_repo_days, docs_days) follow the same rule, and it says
// the right thing: a run that carried on from before counts only the days it
// added this stage, and one that broke and restarted counts what it has. A run
// counted whole would let a hundred-day streak from last month decide this
// month's fork.
func (s *State) Stage(counter string) int {
	if maxCounters[counter] {
		return s.StageMax[counter]
	}
	now, base := s.Counters[counter], s.StageBase[counter]
	if now < base {
		return now
	}
	return now - base
}

// NewStage starts a stage: every counter's present value becomes the base.
func (s *State) NewStage() {
	s.StageBase = make(map[string]int, len(s.Counters))
	for k, v := range s.Counters {
		s.StageBase[k] = v
	}
	s.StageMax = map[string]int{}
}

// Collect books a form in the album for this generation, once.
func (s *State) Collect(form string, at int64) {
	gen := s.Generation
	if gen < 1 {
		gen = 1
	}
	for _, e := range s.Album {
		if e.Form == form && e.Gen == gen {
			return
		}
	}
	s.Album = append(s.Album, AlbumEntry{Form: form, Gen: gen, At: at})
}

// MarkDay counts CONSECUTIVE DAYS, not occurrences: only the day change counts,
// so repeating it twenty times today adds nothing.
func (s *State) MarkDay(name string, at time.Time) bool {
	counter, day := name+"_days", Today(at)
	if s.DayMarks == nil {
		s.DayMarks = map[string]string{}
	}
	if s.DayMarks[counter] == day {
		return false
	}
	if s.DayMarks[counter] == Yesterday(at) {
		s.Counters[counter]++
	} else {
		if s.Counters == nil {
			s.Counters = map[string]int{}
		}
		s.Counters[counter] = 1
	}
	s.DayMarks[counter] = day
	return true
}

// MarkRepoDay books a session closed in repo today, and sets same_repo_days to
// the best run still alive across every repo.
//
// A file from before RepoRuns carries its one run in RepoDay ("repo|day") and
// in the counter; that run is taken over rather than started from scratch.
// RepoDay itself is still written, so an older build reading this file keeps
// the meaning it had.
func (s *State) MarkRepoDay(repo string, now time.Time) {
	if repo == "" {
		return
	}
	today, yesterday := Today(now), Yesterday(now)
	if s.RepoRuns == nil {
		s.RepoRuns = map[string]RepoRun{}
	}
	if s.Counters == nil {
		s.Counters = map[string]int{}
	}
	if len(s.RepoRuns) == 0 {
		if i := strings.LastIndex(s.RepoDay, "|"); i > 0 {
			run := s.Counters["same_repo_days"]
			if run < 1 {
				run = 1
			}
			s.RepoRuns[s.RepoDay[:i]] = RepoRun{Day: s.RepoDay[i+1:], Run: run}
		}
	}

	run := s.RepoRuns[repo]
	switch run.Day {
	case today:
	case yesterday:
		run = RepoRun{Day: today, Run: run.Run + 1}
	default:
		run = RepoRun{Day: today, Run: 1}
	}
	s.RepoRuns[repo] = run

	best := 0
	for name, r := range s.RepoRuns {
		if r.Day != today && r.Day != yesterday {
			delete(s.RepoRuns, name)
			continue
		}
		if r.Run > best {
			best = r.Run
		}
	}
	s.Counters["same_repo_days"] = best
	s.RepoDay = repo + "|" + today
}
