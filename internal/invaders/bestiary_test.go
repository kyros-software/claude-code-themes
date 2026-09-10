package invaders

import (
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The sprites are pixel grids packed into half blocks and pasted in, which is a
// paste waiting to lose a character. Every row has to be exactly as wide as the
// grid says it is, in CELLS - one glyph too many and the formation shears, one
// too few and it gaps.
func TestEverySpriteIsTheWidthItClaims(t *testing.T) {
	if len(Troops) != 3 {
		t.Errorf("%d troop sprites, want the arcade's three", len(Troops))
	}
	if len(Bosses) != 35 {
		t.Errorf("%d rank sprites, want the canvas's 35", len(Bosses))
	}

	for _, s := range Troops {
		if w := theme.Width(s.Glyph); w != TroopCols {
			t.Errorf("%s is %d cells wide, want %d: %q", s.Name, w, TroopCols, s.Glyph)
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

// Three species, and no two of them the same glyph - which is the whole reason
// the arcade drew three rather than one.
func TestTheThreeAreThreeDifferentThings(t *testing.T) {
	seen := map[string]string{}
	for _, s := range Troops {
		key := s.Glyph
		if other, dup := seen[key]; dup {
			t.Errorf("%s is drawn exactly like %s", s.Name, other)
		}
		seen[key] = s.Name
		if s.Name == "" || s.Points <= 0 {
			t.Errorf("%q is worth %d points", s.Name, s.Points)
		}
	}
	// The arcade's own values, top row worth the most.
	for i := 1; i < len(Troops); i++ {
		if Troops[i].Points >= Troops[i-1].Points {
			t.Errorf("%s is worth %d and %s is worth %d: the top row has to pay best",
				Troops[i].Name, Troops[i].Points, Troops[i-1].Name, Troops[i-1].Points)
		}
	}
}

// A boss has two leg frames, and they have to differ, or the walk is a still.
// The troops have no walk: at one cell there is nothing to animate, and what
// moves is the block.
func TestEverySpriteWalks(t *testing.T) {
	for _, s := range Bosses {
		if s.Legs[0] == s.Legs[1] {
			t.Errorf("boss %s has the same legs in both frames", s.Name)
		}
	}
}

// A glyph you cannot see is an enemy nobody can aim at.
func TestNoSpriteIsBlank(t *testing.T) {
	for _, s := range Troops {
		if strings.TrimSpace(s.Glyph) == "" {
			t.Errorf("%s is drawn with %q", s.Name, s.Glyph)
		}
	}
}

// Every stage and rank has to be worth something and be called something, since
// what a stage changes is the colour and the price.
func TestEveryPaletteIsFilledIn(t *testing.T) {
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
	for i := 1; i < len(Stages); i++ {
		if Stages[i].Points <= Stages[i-1].Points {
			t.Errorf("stage %d is worth no more than stage %d", i+1, i)
		}
	}
	for i := 1; i < len(Ranks); i++ {
		if Ranks[i].Points <= Ranks[i-1].Points {
			t.Errorf("rank %d is worth no more than rank %d", i+1, i)
		}
	}
}

// Every boss has eyes: they are the only thing that breaks its flat colour.
func TestEveryBossHasEyes(t *testing.T) {
	for _, s := range Bosses {
		if strings.TrimSpace(s.Eyes) == "" {
			t.Errorf("boss %s has no eyes", s.Name)
		}
	}
}
