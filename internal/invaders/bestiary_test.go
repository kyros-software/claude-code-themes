package invaders

import (
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The sprites come off a design canvas as pipe-separated strings, which is a
// paste waiting to lose a character. Every row has to be exactly as wide as the
// grid says it is, in CELLS - one glyph too many and the formation shears, one
// too few and it gaps.
func TestEverySpriteIsTheWidthItClaims(t *testing.T) {
	if len(Troops) != 40 {
		t.Errorf("%d troop sprites, want the canvas's 40", len(Troops))
	}
	if len(Bosses) != 35 {
		t.Errorf("%d rank sprites, want the canvas's 35", len(Bosses))
	}

	for _, s := range Troops {
		rows := map[string]string{
			"top":   s.Top,
			"legs":  s.Legs[0],
			"legs2": s.Legs[1],
		}
		for name, row := range rows {
			if w := theme.Width(row); w != TroopCols {
				t.Errorf("troop %s: %s is %d cells, want %d: %q", s.Name, name, w, TroopCols, row)
			}
		}
		if w := theme.Width(s.Left) + theme.Width(s.Eyes) + theme.Width(s.Right); w != TroopCols {
			t.Errorf("troop %s: the face is %d cells, want %d", s.Name, w, TroopCols)
		}
	}

	for _, s := range Bosses {
		for name, row := range map[string]string{
			"top": s.Top, "upper": s.Upper, "lower": s.Lower,
			"legs": s.Legs[0], "legs2": s.Legs[1],
		} {
			if w := theme.Width(row); w != BossCols {
				t.Errorf("boss %s: %s is %d cells, want %d: %q", s.Name, name, w, BossCols, row)
			}
		}
		if w := theme.Width(s.Left) + theme.Width(s.Eyes) + theme.Width(s.Right); w != BossCols {
			t.Errorf("boss %s: the face is %d cells, want %d", s.Name, w, BossCols)
		}
	}
}

// Every sprite belongs to a stage or a rank that exists, or it is drawn in a
// colour read off the end of an array.
func TestEverySpriteBelongsToAPaletteThatExists(t *testing.T) {
	for _, s := range Troops {
		if s.Stage < 1 || s.Stage > len(Stages) {
			t.Errorf("troop %s is in stage %d of %d", s.Name, s.Stage, len(Stages))
		}
		if s.Name == "" {
			t.Error("a troop sprite has no name")
		}
	}
	for _, s := range Bosses {
		if s.Rank < 1 || s.Rank > len(Ranks) {
			t.Errorf("boss %s is in rank %d of %d", s.Name, s.Rank, len(Ranks))
		}
	}
	for i, p := range Stages {
		if p.Points <= 0 || p.Label == "" {
			t.Errorf("stage %d is worth %d points and is called %q", i+1, p.Points, p.Label)
		}
	}
	for i, p := range Ranks {
		if p.Points <= 0 || p.Label == "" {
			t.Errorf("rank %d is worth %d points and is called %q", i+1, p.Points, p.Label)
		}
	}
}

// The stages are worth more as they go deeper, or there is no reason to reach
// them.
func TestADeeperStageIsWorthMore(t *testing.T) {
	for i := 1; i < len(Stages); i++ {
		if Stages[i].Points <= Stages[i-1].Points {
			t.Errorf("stage %d is worth %d and stage %d is worth %d",
				i+1, Stages[i].Points, i, Stages[i-1].Points)
		}
	}
	for i := 1; i < len(Ranks); i++ {
		if Ranks[i].Points <= Ranks[i-1].Points {
			t.Errorf("rank %d is worth %d and rank %d is worth %d",
				i+1, Ranks[i].Points, i, Ranks[i-1].Points)
		}
	}
}

// The eyes are the only thing that breaks the flat colour of a sprite, and they
// are three cells in the middle of the face. Every one of the seventy-five has
// them, because a bug with no eyes reads as a block.
func TestEverySpriteHasEyes(t *testing.T) {
	for _, s := range Troops {
		if strings.TrimSpace(s.Eyes) == "" {
			t.Errorf("troop %s has no eyes", s.Name)
		}
	}
	for _, s := range Bosses {
		if strings.TrimSpace(s.Eyes) == "" {
			t.Errorf("boss %s has no eyes", s.Name)
		}
	}
}

// Two leg frames, and they have to differ, or the walk is a still.
func TestEverySpriteWalks(t *testing.T) {
	for _, s := range Troops {
		if s.Legs[0] == s.Legs[1] {
			t.Errorf("troop %s has the same legs in both frames", s.Name)
		}
	}
	for _, s := range Bosses {
		if s.Legs[0] == s.Legs[1] {
			t.Errorf("boss %s has the same legs in both frames", s.Name)
		}
	}
}
