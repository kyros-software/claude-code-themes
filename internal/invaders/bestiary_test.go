package invaders

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Every row of every sprite has to be exactly as wide as its craft says, and
// there have to be as many rows as it claims.
//
// This is the test that earns its keep on a paste. The painter blits a row at a
// time from the top-left corner, so a row one cell short leaves the hull open on
// the right and a row one cell long spills into whatever is beside it - and
// neither shows up as an error, only as art that looks slightly wrong in a frame
// that is gone in fifty milliseconds.
func TestEverySpriteIsTheWidthItClaims(t *testing.T) {
	for _, c := range append(append([]Craft{}, Fleet...), Rock) {
		if len(c.Rows) != c.H {
			t.Errorf("%s says %d rows and draws %d", c.Name, c.H, len(c.Rows))
		}
		for i, row := range c.Rows {
			if got := utf8.RuneCountInString(row); got != c.W {
				t.Errorf("%s row %d is %d cells and should be %d: %q", c.Name, i, got, c.W, row)
			}
			if theme.Width(row) != c.W {
				t.Errorf("%s row %d measures %d columns in a terminal: %q",
					c.Name, i, theme.Width(row), row)
			}
		}
	}
	for _, b := range Bosses {
		for _, row := range []string{b.Top, b.Upper, b.Lower, b.Legs[0], b.Legs[1]} {
			if got := theme.Width(row); got != BossCols {
				t.Errorf("%s has a row %d cells wide: %q", b.Name, got, row)
			}
		}
		if theme.Width(b.Left+b.Eyes+b.Right) != BossCols {
			t.Errorf("%s middle row is %d cells", b.Name, theme.Width(b.Left+b.Eyes+b.Right))
		}
	}
}

// Ten ships, and no two of them the same thing. Two that fly and shoot alike are
// one ship drawn twice, which is nine kinds of enemy and a bigger table.
func TestNoTwoOfTheFleetAreTheSameShip(t *testing.T) {
	if len(Fleet) != 10 {
		t.Fatalf("the fleet is ten ships, there are %d", len(Fleet))
	}
	type flight struct {
		W, H, HP, Cadence, Points int
		Fall, Drift               float64
	}
	seen := map[flight]string{}
	names := map[string]bool{}
	art := map[string]string{}
	for _, c := range Fleet {
		if names[c.Name] {
			t.Errorf("two ships called %q", c.Name)
		}
		names[c.Name] = true

		shape := strings.Join(c.Rows, "\n")
		if other, dup := art[shape]; dup {
			t.Errorf("%s is drawn exactly like %s", c.Name, other)
		}
		art[shape] = c.Name

		f := flight{c.W, c.H, c.HP, c.Cadence, c.Points, c.Fall, c.Drift}
		if other, dup := seen[f]; dup {
			t.Errorf("%s flies exactly like %s", c.Name, other)
		}
		seen[f] = c.Name
	}
}

// Every ship needs a gun, a fall and something to be worth, or it is scenery
// that cannot be cleared - and a wave does not end until the last of them is
// gone.
func TestEveryShipCanFlyAndBeKilled(t *testing.T) {
	for _, c := range append(append([]Craft{}, Fleet...), Rock) {
		if c.HP < 1 {
			t.Errorf("%s arrives already dead", c.Name)
		}
		if c.Fall <= 0 {
			t.Errorf("%s never comes down", c.Name)
		}
		if c.W < 1 || c.H < 2 {
			// Two rows is the floor, and not for looks: a shot travels 1.1 rows
			// a tick, so a one-row target can be stepped clean over - which is
			// exactly the bug that made the old block's top row unkillable.
			t.Errorf("%s is %dx%d, and anything one row tall can be jumped over by a shot",
				c.Name, c.W, c.H)
		}
		if c.Name != Rock.Name {
			if c.Cadence < 1 {
				t.Errorf("%s has no gun", c.Name)
			}
			if c.Points < 1 {
				t.Errorf("%s is worth nothing", c.Name)
			}
			if c.Stage < 1 || c.Stage > stagesDeep {
				t.Errorf("%s arrives in stage %d, and there are %d", c.Name, c.Stage, stagesDeep)
			}
		}
	}
}

// The fleet comes out over the run rather than all at once: wave one is two
// kinds of ship and by the last stage it is all ten. A stage that unlocks
// nothing new is a stage the player cannot tell from the one before.
func TestTheFleetIsUnlockedStageByStage(t *testing.T) {
	first := 0
	for _, c := range Fleet {
		if c.Stage == 1 {
			first++
		}
	}
	if first < 2 {
		t.Errorf("stage one has %d ships in it, which is not a fleet", first)
	}
	deepest := 0
	for _, c := range Fleet {
		if c.Stage > deepest {
			deepest = c.Stage
		}
	}
	if deepest != stagesDeep {
		t.Errorf("the deepest ship arrives in stage %d and the ladder goes to %d",
			deepest, stagesDeep)
	}
}

// The bigger a ship is, the more it is worth: a player who has just spent six
// shots on a coraza has to be paid better than one who clipped a zángano.
func TestABiggerShipIsWorthMore(t *testing.T) {
	for _, a := range Fleet {
		for _, b := range Fleet {
			if a.HP > b.HP && a.Points < b.Points {
				t.Errorf("%s takes more killing than %s and pays less: %d against %d",
					a.Name, b.Name, a.Points, b.Points)
			}
		}
	}
}

func TestNoSpriteIsBlank(t *testing.T) {
	for _, c := range append(append([]Craft{}, Fleet...), Rock) {
		if strings.TrimSpace(strings.Join(c.Rows, "")) == "" {
			t.Errorf("%s is drawn with nothing at all", c.Name)
		}
		if c.Name == "" || c.Desc == "" {
			t.Errorf("a ship with no name or no description: %+v", c)
		}
	}
}

// The two leg frames of a boss have to differ, or it slides instead of walking.
func TestEveryBossWalks(t *testing.T) {
	for _, b := range Bosses {
		if b.Legs[0] == b.Legs[1] {
			t.Errorf("%s has the same legs in both frames", b.Name)
		}
	}
}

func TestEveryPaletteIsFilledIn(t *testing.T) {
	for i, p := range append(append([]Palette{}, Stages[:]...), Ranks[:]...) {
		if p.Label == "" {
			t.Errorf("palette %d has no label", i)
		}
		if p.Points < 1 {
			t.Errorf("%s pays nothing", p.Label)
		}
		var zero theme.Colour
		for j, c := range p.Ramp {
			if c == zero {
				t.Errorf("%s has no colour at ramp step %d", p.Label, j)
			}
		}
		for j, c := range p.Tones {
			if c == zero {
				t.Errorf("%s has no colour at tone %d", p.Label, j)
			}
		}
		if p.Eye == zero {
			t.Errorf("%s has no eye colour", p.Label)
		}
	}
}

// The eyes are the one thing that breaks the flat colour of a sprite, and they
// are what makes a screen of them readable. Every boss has them, and so does
// every ship in the fleet bar the smallest.
func TestEveryBossHasEyes(t *testing.T) {
	for _, b := range Bosses {
		if strings.TrimSpace(b.Eyes) == "" {
			t.Errorf("%s has no eyes", b.Name)
		}
	}
	for _, c := range Fleet {
		if !strings.Contains(strings.Join(c.Rows, ""), "o") {
			t.Errorf("%s has no eyes to paint in the light tone", c.Name)
		}
	}
}

// Thirty-five bosses in five ranks, and every rank has some: bossFor picks by
// rank, so an empty one would hand back the first boss on the list for ever.
func TestEveryRankHasBossesInIt(t *testing.T) {
	if len(Bosses) != 35 {
		t.Fatalf("the canvas has thirty-five, there are %d", len(Bosses))
	}
	count := map[int]int{}
	for _, b := range Bosses {
		count[b.Rank]++
	}
	for rank := 1; rank <= len(Ranks); rank++ {
		if count[rank] == 0 {
			t.Errorf("rank %d has no bosses", rank)
		}
	}
}
