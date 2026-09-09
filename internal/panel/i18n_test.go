package panel

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// TestMain pins the language for this package: these tests assert on wording,
// and the theme now speaks two. Spanish, because that is the default and the
// assertions were written against it - a developer with CCPET_LANG=en in their
// shell would otherwise fail every one of them.
func TestMain(m *testing.M) {
	os.Setenv(i18n.Env, string(i18n.ES))
	os.Exit(m.Run())
}

func aPetWorthPrinting() *pet.State {
	s := pet.New()
	s.XP = 450
	s.Hunger = 3
	s.Streak = 4
	s.BestStreak = 9
	s.AteAt = time.Date(2026, 9, 9, 11, 0, 0, 0, time.UTC).Unix()
	s.Counters = map[string]int{
		"methodical": 40, "diffs": 12, "diff_streak": 6,
		"bypass_turns": 8, "repro_before_fix": 33,
	}
	return s
}

// The whole panel in English, end to end: every section, and not one Spanish
// row label left behind. The words are checked by hand rather than against the
// catalogue, because a test that reads the same table the code does would pass
// on a label nobody ever translated.
func TestThePanelPrintsInEnglish(t *testing.T) {
	i18n.Use(i18n.EN)
	defer i18n.Use("")

	now := time.Date(2026, 9, 9, 13, 30, 0, 0, time.UTC)
	body := strings.Join(render(t, aPetWorthPrinting(), now), "\n")

	for _, want := range []string{
		"level", "hunger", "streak", "best 9", "ate ",
		"habits", "done", "on its way", "leads nowhere from here",
		"sessions", "turns on bypass", "clean diffs",
		"food", "green suite", "ready", "context at 100%",
		"if you blow the context",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the English panel never says %q", want)
		}
	}
	for _, unwanted := range []string{
		"nivel", "hambre", "racha", "mejor", "comió", "hábitos", "sesiones",
		"comida", "listo", "cumplido", "en camino", "diffs limpios",
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("the English panel still says %q", unwanted)
		}
	}
}

// The bars line up in both languages: the row labels are four different words
// of four different lengths, and the column they end in used to be two
// hardcoded spaces.
func TestTheBarsStartInTheSameColumnInBothLanguages(t *testing.T) {
	defer i18n.Use("")
	now := time.Date(2026, 9, 9, 13, 30, 0, 0, time.UTC)
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		at, rows := -1, 0
		for _, line := range render(t, aPetWorthPrinting(), now) {
			bar := strings.IndexAny(line, "█░")
			if bar < 0 || strings.HasPrefix(strings.TrimSpace(line), "·") {
				continue
			}
			// The XP column beside the sprite is a bar too, and it is indented
			// differently on purpose; the four rows all start at column 2.
			if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "     ") {
				continue
			}
			rows++
			if at == -1 {
				at = bar
				continue
			}
			if bar != at {
				t.Errorf("%s: a bar starts at column %d, the first one at %d\n%q",
					lang, bar, at, line)
			}
		}
		if rows < 3 {
			t.Errorf("%s: only %d bar rows were compared; this test is not looking at the panel", lang, rows)
		}
	}
}
