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

// pilot is a player: it lines the ship up under whatever is lowest, shoots,
// reloads when it is empty and spends a kit when it is hurt. Deliberately simple
// - a person dodges bombs, picks off the ships that are about to land and saves
// the ability for a crowd, and this does none of that - so the waves it reaches
// are a floor and not a forecast.
func pilot(g Game) Key {
	if g.Phase == Choosing {
		// Always the magazine, because the pilot's weakness is standing still
		// reloading and that is the upgrade that shortens it.
		return Three
	}
	if g.Ammo == 0 && g.Loading == 0 {
		return Rearm
	}
	if g.Kits > 0 && g.HP*2 <= g.Kit.MaxHP {
		return Heal
	}
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
	// Lined up, and nothing to stop: not pressing anything IS letting go, so the
	// ship parks itself.
	return Fire
}

// lowestColumn is whatever is closest to the floor, which is what a pilot with
// one gun should be shooting at.
func lowestColumn(g Game) (float64, bool) {
	if g.Boss.Alive {
		return g.Boss.X + BossCols/2, true
	}
	best, low, found := 0.0, -1.0, false
	for _, a := range g.Aliens {
		if !found || a.Y > low {
			best, low, found = a.X+float64(a.Craft().W)/2, a.Y, true
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
		for _, a := range g.Aliens {
			if a.Y < 0 || a.X < 0 || a.X+float64(a.Craft().W) > float64(g.Field.Cols) {
				t.Fatalf("tick %d: a %s at %g,%g of %d columns",
					ticks, a.Craft().Name, a.X, a.Y, g.Field.Cols)
			}
		}
		if g.Ammo < 0 || g.Ammo > g.Kit.Cap {
			t.Fatalf("tick %d: %d rounds of %d", ticks, g.Ammo, g.Kit.Cap)
		}
		if g.Kits < 0 || g.Kits > kitsMax {
			t.Fatalf("tick %d: %d kits in the hold", ticks, g.Kits)
		}
	}
	if g.Phase != Over {
		t.Fatalf("still alive on wave %d after %d ticks: the ladder does not climb", g.Wave.N, ticks)
	}
	t.Logf("the autopilot got to wave %d in %d minutes", best, ticks/TicksPerSecond/60)
}
