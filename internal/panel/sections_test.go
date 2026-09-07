package panel

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// panelFor draws /pet for a given state and hands back the plain text.
func panelFor(t *testing.T, tweak func(*pet.State)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pet.json")
	s := pet.New()
	tweak(s)
	if !pet.Save(s, path) {
		t.Fatal("could not write the state")
	}
	var out bytes.Buffer
	if code := showPanel(&out, path, time.Now()); code != 0 {
		t.Fatalf("showPanel returned %d", code)
	}
	return theme.Strip(out.String())
}

// A level 5 fork is a RACE, and a race needs both runners on screen. The panel
// showed only the leader, which cannot be read as a race at all: "sabueso
// 33/10" says the fork is settled and says nothing about what it beat.
func TestTheForkShowsBothMarksAndWhichIsAhead(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 515
		s.Counters = map[string]int{
			"inquisitive": 21, "tests": 21, // spark -> probe -> bughunter
			"repro_before_fix": 33, // bloodhound asks 10: paid for
			"test_streak":      1,  // exterminator asks 15
		}
	})
	for _, want := range []string{"la marca del nivel 5", "sabueso", "exterminador", "33/10", "1/15"} {
		if !strings.Contains(got, want) {
			t.Errorf("the fork section is missing %q:\n%s", want, got)
		}
	}
	// The one already paid for is ticked, and it comes first.
	if i, j := strings.Index(got, "sabueso"), strings.Index(got, "exterminador"); i > j {
		t.Error("the mark that is ahead is not listed first")
	}
	if !strings.Contains(got, "✓ sabueso") {
		t.Error("the mark whose habit is already met is not ticked")
	}
}

// A trade with no fork left - a mark, a title - has no section to print, and
// must not print an empty one.
func TestNoForkSectionWithoutAFork(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 0 // spark: its children are temperaments, not marks
	})
	if strings.Contains(got, "la marca del nivel 5") {
		t.Errorf("a larva was given a fork section:\n%s", got)
	}
}

// Twenty-odd counters decide the tree and the panel showed one. Every counter
// that has moved is now on screen, under the question it answers.
func TestEveryCounterThatMovedIsShown(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 515
		s.Counters = map[string]int{
			"inquisitive": 21, "tests": 21,
			"repro_before_fix": 33,
			"widest_commit":    7,
			"bypass_turns":     206,
			"sessions_4h":      4,
			"longest_plan":     0, // at zero: a habit that has not started
		}
	})
	for _, want := range []string{
		"hábitos", "reproducir antes de arreglar", "33",
		"commit más ancho", "7",
		"sesiones", "turnos en bypass", "206", "sesiones de 4h o más",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "plan más largo cerrado") {
		t.Error("a counter still at zero was printed")
	}
}

// What can be eaten now, and what has to wait. A refused meal used to say so
// only at the moment you tried it.
func TestTheLarderSaysWhatIsReadyAndWhatIsWaiting(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 100
		s.Meals = map[string]int64{"feed": time.Now().Unix()} // just fed
	})
	if !strings.Contains(got, "comida") {
		t.Fatalf("no larder:\n%s", got)
	}
	if !strings.Contains(got, "tests en verde") || !strings.Contains(got, "listo") {
		t.Errorf("a ready meal is not marked ready:\n%s", got)
	}
	// /feed has a four-hour cooldown and was just eaten.
	line := ""
	for _, l := range strings.Split(got, "\n") {
		if strings.Contains(l, "/feed") {
			line = l
		}
	}
	if !strings.Contains(line, "en ") {
		t.Errorf("the cooldown is not shown: %q", line)
	}
}

// The overflow is not a meal. It is what happens TO the pet, so it is not
// listed as something ready to eat.
func TestTheOverflowIsNotOnTheMenu(t *testing.T) {
	got := panelFor(t, func(s *pet.State) { s.XP = 100 })
	for _, l := range strings.Split(got, "\n") {
		if strings.Contains(l, "contexto al 100%") && strings.Contains(l, "listo") {
			t.Errorf("the penalty is offered as a meal: %q", l)
		}
	}
	if !strings.Contains(got, "si revientas el contexto") {
		t.Errorf("the penalty is not explained at all:\n%s", got)
	}
}

// The XP row says where you ARE out of where the next level IS. It used to
// print the stretch - 118 of 1600 - which is a third number next to the total
// under the sprite and relates to neither.
func TestTheXPRowIsTheTotalAndTheTarget(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 518
		s.Counters = map[string]int{"inquisitive": 21, "tests": 21}
	})
	if !strings.Contains(got, "518/2000 xp") {
		t.Errorf("want the total over the next threshold:\n%s", got)
	}
	if strings.Contains(got, "118/1600") {
		t.Error("the stretch is being printed as if it were the position")
	}
}

// The bug was never that a habit appears twice - the fork section is a race
// and the habits table is an inventory, and the same number can answer both
// questions. It was that the two DISAGREED: the summary row printed "10/10",
// capped at the threshold, beside the section's "33/10".
func TestTheHabitNeverContradictsItself(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 515
		s.Counters = map[string]int{
			"inquisitive": 21, "tests": 21, "repro_before_fix": 33, "test_streak": 1,
		}
	})
	if strings.Contains(got, "10/10") {
		t.Errorf("a habit is printed capped at its own threshold:\n%s", got)
	}
	if !strings.Contains(got, "33/10") {
		t.Errorf("the real count is missing:\n%s", got)
	}
	// The old summary row is gone: the fork section carries it now.
	if strings.Contains(got, "marca  ") {
		t.Errorf("the capped summary row is back:\n%s", got)
	}
}

// The colour is the point of the table, and it means one thing: whether this
// counter is taking the pet anywhere it can still get to.
//
//	green   the threshold is met
//	amber   on its way to a shape still ahead
//	white   counted, and leading nowhere from here
//
// Most of the table is white, and that is the reading: bypass_turns opens
// `gremlin`, off the ember branch, which a probe walked past long ago.
func TestTheColourSaysWhetherACounterLeadsAnywhere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pet.json")
	s := pet.New()
	s.XP = 515
	s.Counters = map[string]int{
		"inquisitive": 21, "tests": 21,
		"repro_before_fix": 33,  // bloodhound, ahead, threshold met
		"test_streak":      1,   // exterminator, ahead, on its way
		"bypass_turns":     206, // gremlin, on a branch behind us
	}
	if !pet.Save(s, path) {
		t.Fatal("could not write the state")
	}
	var out bytes.Buffer
	showPanel(&out, path, time.Now())
	painted := out.String()

	for _, tc := range []struct {
		label string
		tint  theme.Colour
		why   string
	}{
		{"reproducir antes de arreglar", theme.Ident, "threshold met"},
		{"días seguidos en verde", theme.Number, "on its way"},
		{"turnos en bypass", theme.Emph, "leads nowhere from here"},
	} {
		line := ""
		for _, l := range strings.Split(painted, "\n") {
			if strings.Contains(theme.Strip(l), tc.label) &&
				!strings.Contains(theme.Strip(l), "marca del nivel") {
				line = l
			}
		}
		if line == "" {
			t.Errorf("%q is not in the table", tc.label)
			continue
		}
		if !strings.Contains(line, theme.Fg(tc.tint)) {
			t.Errorf("%q (%s) is not painted as expected", tc.label, tc.why)
		}
	}
}

// And the key is on screen, because three tints nobody can decode are three
// tints of decoration.
func TestTheColourKeyIsPrinted(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = 515
		s.Counters = map[string]int{"inquisitive": 21, "tests": 21, "repro_before_fix": 33}
	})
	for _, want := range []string{"cumplido", "en camino", "no lleva a nada"} {
		if !strings.Contains(got, want) {
			t.Errorf("the key is missing %q:\n%s", want, got)
		}
	}
}

// Every row of the larder lines up, the penalty included - it has the longest
// label, and leaving it out of the width is what pushed its line out of true.
func TestTheLarderColumnsLineUp(t *testing.T) {
	got := panelFor(t, func(s *pet.State) { s.XP = 100 })
	var widths []int
	for _, l := range strings.Split(got, "\n") {
		if i := strings.Index(l, " xp   "); i >= 0 && strings.HasPrefix(l, "    ") {
			widths = append(widths, i)
		}
	}
	if len(widths) < 5 {
		t.Fatalf("found %d larder rows, want at least 5", len(widths))
	}
	for _, w := range widths[1:] {
		if w != widths[0] {
			t.Errorf("the larder columns do not line up: %v", widths)
			break
		}
	}
}
