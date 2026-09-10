// Package invaders is ccpet invade: a shmup whose ship is the pet you already
// have. The pet's form decides the weapon, its mark refines it and its level
// scales it.
//
// It reads pet.json and writes it exactly once, when a run is lost, to take the
// level that cost. Nothing else in here can touch it. See pet.Setback.
package invaders

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// MaxWave is as far as the ladder is ever asked about. There is no last wave -
// a run ends when the enemies outgrow your kit, not at a number - so this is not
// a goal, it is arithmetic safety: the wave count comes off disk, and a
// hand-edited or corrupt file saying 1e9 would otherwise walk straight into the
// multiplications the curve is made of.
const MaxWave = 10000

// MaxUps is as many upgrades of one kind as a save may claim. Sanity, like
// MaxWave: they are replayed one at a time when a run resumes, and a corrupt
// file saying a billion would be a billion trips through boost before the first
// frame.
const MaxUps = 500

// Save is a run between sessions, and the records that outlive it.
//
// Its own file and not a corner of pet.json: that file has strict
// legacy-compatibility rules, a reader on every statusline refresh and a lock
// around every write, and a game has no business near any of them.
type Save struct {
	Wave      int    `json:"wave"`
	HP        int    `json:"hp"`
	Score     int    `json:"score"`
	Kills     int    `json:"kills"`
	BestWave  int    `json:"best_wave"`
	BestScore int    `json:"best_score"`
	Runs      int    `json:"runs"`
	Revived   bool   `json:"revived"`
	Seed      uint64 `json:"seed"`

	// The upgrades taken in the run, counted by kind. They are here because a
	// run is quit and resumed all the time - the arena pauses it every turn -
	// and coming back with the pet's bare kit after building a gun for twenty
	// waves would read as the game having forgotten.
	Power int `json:"up_power"`
	Speed int `json:"up_speed"`
	Mag   int `json:"up_mag"`
}

// SavePath is ~/.claude/invaders.json, through config.Dir like everything else.
func SavePath() string { return filepath.Join(config.Dir(), "invaders.json") }

// LoadSave never fails. A missing file, unparseable JSON, a directory where the
// file goes, or a wave from the far future is a fresh run - the same rule
// pet.Load follows, and for the same reason: a game that will not start because
// of a bad byte on disk is worse than a game that starts over.
//
// The names are LoadSave and StoreSave rather than Load and Save so that nothing
// in run.go can be misread as loading or saving the pet.
func LoadSave(path string) Save {
	// Every way out of here goes through sane, including the ones that give up
	// early: a fresh run is wave ONE, and an unreadable file that handed back a
	// zeroed struct was a run starting on wave zero, which the curve has no
	// recipe for.
	raw, err := os.ReadFile(path)
	if err != nil {
		return Save{}.sane()
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil || doc == nil {
		return Save{}.sane()
	}
	s := Save{
		Wave:      asInt(doc["wave"]),
		HP:        asInt(doc["hp"]),
		Score:     asInt(doc["score"]),
		Kills:     asInt(doc["kills"]),
		BestWave:  asInt(doc["best_wave"]),
		BestScore: asInt(doc["best_score"]),
		Runs:      asInt(doc["runs"]),
		Revived:   doc["revived"] == true,
		Seed:      uint64(asInt(doc["seed"])),
		Power:     asInt(doc["up_power"]),
		Speed:     asInt(doc["up_speed"]),
		Mag:       asInt(doc["up_mag"]),
	}
	return s.sane()
}

// sane pins every number into the range the game can actually play in. A wave
// of 0 is a run that never starts and a wave of a billion is an overflow.
func (s Save) sane() Save {
	if s.Wave < 1 {
		s.Wave = 1
	}
	if s.Wave > MaxWave {
		s.Wave = MaxWave
	}
	for _, n := range []*int{&s.HP, &s.Score, &s.Kills, &s.BestWave, &s.BestScore, &s.Runs} {
		if *n < 0 {
			*n = 0
		}
	}
	// The upgrades are replayed one at a time on the way in, so a hand-edited
	// file saying a million of them would be a million passes through boost.
	for _, n := range []*int{&s.Power, &s.Speed, &s.Mag} {
		*n = clamp(*n, 0, MaxUps)
	}
	if s.BestWave > MaxWave {
		s.BestWave = MaxWave
	}
	if s.BestWave < s.Wave {
		s.BestWave = s.Wave
	}
	return s
}

// Fresh is wave one at full life, keeping the records and counting the run. The
// upgrades are not kept: they were built by the run that just ended.
func (s Save) Fresh(maxHP int, seed uint64) Save {
	return Save{
		Wave: 1, HP: maxHP,
		BestWave: s.BestWave, BestScore: s.BestScore,
		Runs: s.Runs + 1,
		Seed: seed,
	}.sane()
}

// StoreSave writes the run, whole and renamed into place, keeping any key it
// does not know about: this file is small now and will not stay that way.
func StoreSave(s Save, path string) error {
	doc := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, &doc) != nil || doc == nil {
			doc = map[string]any{}
		}
	}
	doc["wave"] = s.Wave
	doc["hp"] = s.HP
	doc["score"] = s.Score
	doc["kills"] = s.Kills
	doc["best_wave"] = s.BestWave
	doc["best_score"] = s.BestScore
	doc["runs"] = s.Runs
	doc["revived"] = s.Revived
	doc["seed"] = s.Seed
	doc["up_power"] = s.Power
	doc["up_speed"] = s.Speed
	doc["up_mag"] = s.Mag

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ccpet-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// asInt reads a number that has been through JSON and may not be one any more.
// Numeric strings count, because a hand-edited file is the common case for this
// kind of file; a bool never does, the way pet.Load has it.
func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		if n != n || n > float64(MaxWave)*1e6 || n < -float64(MaxWave)*1e6 {
			return 0
		}
		return int(n)
	case string:
		out, neg, any := 0, false, false
		for i, r := range n {
			if i == 0 && (r == '-' || r == '+') {
				neg = r == '-'
				continue
			}
			if r < '0' || r > '9' {
				return 0
			}
			out, any = out*10+int(r-'0'), true
			if out > MaxWave*1000000 {
				return 0
			}
		}
		if !any {
			return 0
		}
		if neg {
			return -out
		}
		return out
	}
	return 0
}
