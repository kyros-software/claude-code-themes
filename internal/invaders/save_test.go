package invaders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// TestMain pins the language for this package: these tests assert on wording and
// the theme speaks two. Spanish, because it is the default and a developer with
// CCPET_LANG=en in their shell would otherwise fail every one of them.
func TestMain(m *testing.M) {
	os.Setenv(i18n.Env, string(i18n.ES))
	os.Exit(m.Run())
}

func savePathIn(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "invaders.json")
}

// The whole point of the file. Everything a run needs to come back has to
// survive the trip, including the seed - without it a resumed run plays a
// different sequence from the one it was in, and the determinism the tick is
// built on stops meaning anything across sessions.
func TestTheRunRoundTripsThroughTheFile(t *testing.T) {
	path := savePathIn(t)
	want := Save{
		Wave: 12, HP: 7, Score: 8400, Kills: 611,
		BestWave: 20, BestScore: 15200, Runs: 4,
		Revived: true, Seed: 0xC0FFEE,
	}
	if err := StoreSave(want, path); err != nil {
		t.Fatal(err)
	}
	if got := LoadSave(path); got != want {
		t.Errorf("came back as %+v, want %+v", got, want)
	}
}

// A bad byte on disk must not be a game that will not start. Same rule pet.Load
// follows, and for the same reason: starting over is a worse outcome than
// crashing only if you have never seen a tool crash on a file it wrote itself.
func TestACorruptSaveIsANewRunAndNotACrash(t *testing.T) {
	for _, c := range []struct {
		name, body string
	}{
		{"nothing there", ""},
		{"truncated json", `{"wave": 12, "hp":`},
		{"an array where an object goes", `[1, 2, 3]`},
		{"a bare null", `null`},
		{"a string where a number goes", `{"wave": "twelve", "hp": "seven"}`},
		{"numbers as strings", `{"wave": "12", "hp": "7"}`},
		{"a nan", `{"wave": 1e400}`},
		{"negative everything", `{"wave": -5, "hp": -2, "runs": -1}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := savePathIn(t)
			if c.body != "" {
				if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got := LoadSave(path)
			if got.Wave < 1 || got.Wave > MaxWave {
				t.Errorf("wave came back as %d", got.Wave)
			}
			if got.HP < 0 || got.Runs < 0 {
				t.Errorf("came back as %+v", got)
			}
		})
	}

	t.Run("a directory where the file goes", func(t *testing.T) {
		dir := t.TempDir()
		if got := LoadSave(dir); got.Wave != 1 {
			t.Errorf("reading a directory gave %+v", got)
		}
	})
}

// Numeric strings count, because a hand-edited save is the common case for a
// file like this and losing somebody's records to a pair of quotes is rude.
func TestAHandEditedNumberStillCounts(t *testing.T) {
	path := savePathIn(t)
	if err := os.WriteFile(path, []byte(`{"wave":"12","best_score":"900"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadSave(path)
	if got.Wave != 12 || got.BestScore != 900 {
		t.Errorf("came back as %+v, want wave 12 and 900 points", got)
	}
}

// There is no last wave, so nothing else stops a wave number from growing. A
// save saying a billion would walk into the multiplications the curve is made
// of; this is where that is pinned, once, on the way in.
func TestAWaveFromTheFutureIsPinnedAndNotMultipliedOut(t *testing.T) {
	for _, body := range []string{
		`{"wave": 1000000000}`,
		`{"wave": 1e18}`,
		`{"wave": "999999999999999999999"}`,
		`{"wave": -3}`,
	} {
		path := savePathIn(t)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		got := LoadSave(path)
		if got.Wave < 1 || got.Wave > MaxWave {
			t.Errorf("%s came back as wave %d", body, got.Wave)
		}
	}
}

// The file will grow. A version of the game that does not know a key yet must
// not be the one that deletes it.
func TestKeysWeDoNotKnowSurviveASave(t *testing.T) {
	path := savePathIn(t)
	if err := os.WriteFile(path, []byte(`{"wave":3,"per_form_records":{"wasp":40}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := StoreSave(Save{Wave: 4, HP: 5}, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "per_form_records") {
		t.Errorf("the unknown key is gone:\n%s", raw)
	}
}

// Written whole and renamed into place, and leaving nothing behind if it fails
// halfway. A half-written save read by the next run is a corrupt save, which is
// survivable, but litter in ~/.claude is forever.
func TestTheWriteIsAtomicAndLeavesNoLitter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invaders.json")
	for i := 0; i < 5; i++ {
		if err := StoreSave(Save{Wave: i + 1, HP: 3}, path); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "invaders.json" {
			t.Errorf("it left %q behind", e.Name())
		}
	}
}

// Two files, two systems. The game's records must not be able to arrive in the
// pet's file by a rename, a typo or a refactor.
func TestTheSaveIsNotPetJson(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	if SavePath() == pet.Path() {
		t.Fatalf("the game saves into the pet's own file: %s", SavePath())
	}
	if filepath.Dir(SavePath()) != filepath.Dir(pet.Path()) {
		t.Errorf("the save is not in the config dir: %s", SavePath())
	}
	if filepath.Base(SavePath()) != "invaders.json" {
		t.Errorf("the save is called %q", filepath.Base(SavePath()))
	}
}

// A fresh run keeps what a run is not allowed to reset - the records and the
// number of times you have tried - and counts itself.
func TestAFreshRunKeepsTheRecordsAndCountsItself(t *testing.T) {
	old := Save{Wave: 22, HP: 0, Score: 900, Kills: 40, BestWave: 22, BestScore: 15200, Runs: 3, Revived: true}
	got := old.Fresh(12, 99)

	if got.Wave != 1 || got.HP != 12 {
		t.Errorf("it starts at wave %d with %d hp", got.Wave, got.HP)
	}
	if got.BestWave != 22 || got.BestScore != 15200 {
		t.Errorf("it forgot the records: %+v", got)
	}
	if got.Runs != 4 {
		t.Errorf("runs = %d, want 4", got.Runs)
	}
	if got.Score != 0 || got.Kills != 0 || got.Revived {
		t.Errorf("it carried something over: %+v", got)
	}
	if got.Seed != 99 {
		t.Errorf("seed = %d, want the one it was handed", got.Seed)
	}
}
