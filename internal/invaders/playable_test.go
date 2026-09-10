package invaders

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// levelFor is the level a form is naturally worn at: a larva is level 1 and a
// title is level 6, so asking whether a spark can hold wave 10 is asking the
// wrong question.
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

// autopilot plays a run to its end and reports the wave it got to and how long
// it lasted.
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

// Every one of the forty-one has to be able to play. A form that dies on wave
// one is not a hard form, it is a form nobody can use - and since losing costs
// the creature a level, an unplayable form is a punishment for having evolved
// into it.
//
// The autopilot is deliberately simple: it dodges what is about to touch it,
// leans on its ability, and otherwise steers at whatever will arrive soonest
// and can still be reached. A person plays better than that, so these numbers
// are a floor and not a forecast.
//
// This is the test that found the real balance bug. At a base speed of 0.18 a
// cell per tick one of the swarm took twenty-two seconds to cross the field -
// longer than the whole of wave one - so they piled up thirty-three deep and
// every single-lane family died before wave 3, whatever its level.
func TestEveryFormCanPlayItsOwnFirstWaves(t *testing.T) {
	if len(pet.Sprites) != 41 {
		t.Fatalf("the atlas is 41 forms, there are %d", len(pet.Sprites))
	}
	for form := range pet.Sprites {
		level := levelFor(form)
		wave, _ := autopilot(t, form, level, 0xBEEF)
		if wave < 3 {
			t.Errorf("a %s at level %d died on wave %d", form, level, wave)
		}
	}
}

// The pet's level has to be worth something, or feeding it is decoration. A
// creature at the top of the tree must get measurably further than a larva.
func TestAGrownCreatureGetsFurtherThanALarva(t *testing.T) {
	larva, _ := autopilot(t, "spark", 1, 0xBEEF)
	grown, _ := autopilot(t, "wasp", 6, 0xBEEF)
	if grown <= larva*2 {
		t.Errorf("a larva reached wave %d and a level-6 title reached %d: the level buys nothing",
			larva, grown)
	}
}

// reachable is the lane of the enemy that will arrive soonest AND that the ship
// can still get to in time. Chasing the nearest one by x alone means chasing
// things already past the point of being shot.
func reachable(g Game) (int, bool) {
	best, found := 0, false
	bestCost := 0.0
	for _, e := range g.Enemies {
		ticksToArrive := e.X / e.Speed
		rows := e.Row - g.Face()
		if rows < 0 {
			rows = -rows
		}
		ticksToTurn := float64(rows * moveWait)
		if ticksToTurn > ticksToArrive {
			continue
		}
		if !found || ticksToArrive < bestCost {
			best, bestCost, found = e.Row, ticksToArrive, true
		}
	}
	return best, found
}

// pilot is a player that dodges. Chasing the nearest enemy walks the ship into
// it, which is a way to lose that says nothing about the game.
func pilot(g Game) Key {
	// Anything about to touch the ship: get out of its lane first.
	for _, e := range g.Enemies {
		if e.Boss() || e.X > float64(ShipCols)+6 {
			continue
		}
		if e.Row < g.Ship+HitTop || e.Row > g.Ship+HitBottom {
			continue
		}
		if e.Row-g.Ship <= HitTop+1 && g.Ship < g.Field.ShipRowMax() {
			return Down
		}
		if g.Ship > 0 {
			return Up
		}
		return Down
	}
	if g.Ready == 0 {
		return Fire
	}
	if row, ok := reachable(g); ok {
		if row < g.Face() {
			return Up
		}
		if row > g.Face() {
			return Down
		}
	}
	return None
}
