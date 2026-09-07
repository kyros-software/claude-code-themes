package pet

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
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
	XP         int    `json:"xp"`
	Hunger     int    `json:"hunger"`
	LastFed    int64  `json:"last_fed"`
	Streak     int    `json:"streak"`
	BestStreak int    `json:"best_streak"`
	LevelSeen  int    `json:"level_seen"`
	HungerPeak int    `json:"hunger_peak"`
	FedAt      int64  `json:"fed_at"`
	AteAt      int64  `json:"ate_at"`
	LastDay    string `json:"last_day"`
	RepoDay    string `json:"repo_day"`
	LogDay     string `json:"log_day"`
	Secret     Secret `json:"secret"`

	// FormSeen is the highest rung the shape has ever stood on. The walk is
	// recomputed from the counters every refresh and two habits can fall back
	// to zero, so without this a broken streak took the shape down the tree
	// with it. See pet.Tier and pet.RememberForm.
	//
	// Absent from an older file, which is exactly right: an empty name is tier
	// 0 and holds nothing back, so the first refresh records whatever the pet
	// already is and the floor starts from there.
	FormSeen string `json:"form_seen"`

	Counters map[string]int    `json:"counters"`
	Meals    map[string]int64  `json:"meals"`
	Log      []LogEntry        `json:"log"`
	DayMarks map[string]string `json:"day_marks"`

	// What the pet has said lately, so it does not repeat itself, and when.
	// See speech.go.
	Said   []string `json:"said"`
	SaidAt int64    `json:"said_at"`
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
	if form, _ := CurrentForm(s); form != "" {
		RememberForm(s, form)
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

// Yesterday is the day before it.
func Yesterday(at time.Time) string { return at.Add(-24 * time.Hour).Format("2006-01-02") }

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
