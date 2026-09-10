package invaders

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

func aField(t *testing.T, cols, rows int) Field {
	t.Helper()
	f, ok := FieldFor(cols, rows)
	if !ok {
		t.Fatalf("%dx%d was refused", cols, rows)
	}
	return f
}

func aGame(t *testing.T, form string, level int) Game {
	t.Helper()
	return NewGame(aField(t, 80, 24), form, level, Save{Wave: 1, Seed: 0xA11CE})
}

// drive runs n ticks with the same key, which is most of what a test needs.
func drive(g Game, in Key, n int) Game {
	for i := 0; i < n; i++ {
		g = Tick(g, in)
	}
	return g
}

// pilot is a player: it lines the creature up under whatever is lowest and
// shoots. Deliberately simple - a person dodges bombs and picks off the flanks
// to give the block room, and this does neither - so the waves it reaches are a
// floor and not a forecast.
func pilot(g Game) Key {
	if g.Ready == 0 && g.Kit.Special != AbilityVolley {
		return Ability
	}
	target, ok := lowestColumn(g)
	if !ok {
		return Fire
	}
	muzzle := g.Muzzle()
	switch {
	case target < muzzle-0.5:
		return Left
	case target > muzzle+0.5:
		return Right
	}
	// Lined up: stop drifting past it, then shoot.
	if g.Drift != 0 {
		return Stop
	}
	return Fire
}

func lowestColumn(g Game) (float64, bool) {
	if g.Boss.Alive {
		return g.Boss.X + BossCols/2, true
	}
	best, low, found := 0.0, -1, false
	for _, m := range g.Squad.Members {
		x, y := g.Squad.At(m)
		if !found || y > low {
			best, low, found = float64(x), y, true
		}
	}
	return best, found
}

// autopilot plays a run to its end and reports how far it got.
func autopilot(t *testing.T, form string, level int, seed uint64) (wave, ticks int) {
	t.Helper()
	g := NewGame(aField(t, 80, 24), form, level, Save{Wave: 1, Seed: seed})
	for ; ticks < 2_000_000 && g.Phase != Over; ticks++ {
		g = Tick(g, pilot(g))
		if g.Wave.N > wave {
			wave = g.Wave.N
		}
	}
	return wave, ticks
}

// levelFor is the level a form is naturally worn at.
func levelFor(form string) int {
	switch pet.Tier(form) {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 6:
		return 6
	default:
		return 5
	}
}

// Every one of the forty-one has to be able to play. A form that cannot clear
// wave one is not a hard form, it is a form nobody can use - and since losing
// costs the creature a level, an unplayable form is a punishment for having
// evolved into it.
func TestEveryFormCanPlayItsOwnFirstWaves(t *testing.T) {
	if len(pet.Sprites) != 41 {
		t.Fatalf("the atlas is 41 forms, there are %d", len(pet.Sprites))
	}
	for form := range pet.Sprites {
		level := levelFor(form)
		wave, _ := autopilot(t, form, level, 0xBEEF)
		if wave < 2 {
			t.Errorf("a %s at level %d never cleared wave 1", form, level)
		}
	}
}

// The pet's level has to be worth something, or feeding it is decoration.
func TestAGrownCreatureGetsFurtherThanALarva(t *testing.T) {
	larva, _ := autopilot(t, "spark", 1, 0xBEEF)
	grown, _ := autopilot(t, "wasp", 6, 0xBEEF)
	if grown <= larva {
		t.Errorf("a larva reached wave %d and a level-6 title reached %d", larva, grown)
	}
}

// The run has to end. There is no last wave, so if a pilot can survive for ever
// the ladder does not climb and the game has no ending at all.
//
// This is also the headless proof that the whole tick runs: no tty, one seed,
// a full run's worth of frames.
func TestARunPlaysItselfUntilTheLadderOutgrowsIt(t *testing.T) {
	g := NewGame(aField(t, 80, 24), "wasp", 6, Save{Wave: 1, Seed: 0xBEEF})
	best, ticks := 0, 0
	for ; ticks < 2_000_000 && g.Phase != Over; ticks++ {
		g = Tick(g, pilot(g))
		if g.Wave.N > best {
			best = g.Wave.N
		}
		if g.HP < 0 || g.HP > g.Kit.MaxHP {
			t.Fatalf("tick %d: %d of %d hp", ticks, g.HP, g.Kit.MaxHP)
		}
		if g.Ship < 0 || g.Ship > g.Field.ShipColMax() {
			t.Fatalf("tick %d: the creature is at column %d", ticks, g.Ship)
		}
		for _, m := range g.Squad.Members {
			x, y := g.Squad.At(m)
			if y < 0 || x+TroopCols > g.Field.Cols+cellCols {
				t.Fatalf("tick %d: a member at %d,%d", ticks, x, y)
			}
		}
	}
	if g.Phase != Over {
		t.Fatalf("still alive on wave %d after %d ticks: the ladder does not climb", g.Wave.N, ticks)
	}
	t.Logf("the autopilot got to wave %d in %d minutes", best, ticks/TicksPerSecond/60)
}
