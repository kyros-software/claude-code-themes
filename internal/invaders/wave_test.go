package invaders

import "testing"

// The creature runs along the floor and the block comes down at it. This is the
// geometry every other test in here assumes.
func TestTheCreatureStandsOnTheFloorAndTheBlockStartsAtTheTop(t *testing.T) {
	f := aField(t, 80, 24)
	if f.Rows != 24-HUDRows-HelpRows {
		t.Fatalf("the field is %d rows of a 24 row terminal", f.Rows)
	}
	if f.ShipRow()+ShipRows != f.Rows {
		t.Errorf("the creature sits at row %d of %d and is %d tall", f.ShipRow(), f.Rows, ShipRows)
	}
	if f.ShipColMax()+ShipCols != f.Cols {
		t.Errorf("the creature may reach column %d of %d", f.ShipColMax(), f.Cols)
	}

	fm := NewFormation(WaveFor(1, f), f)
	if fm.Y != 0 {
		t.Errorf("the block starts at row %v", fm.Y)
	}
	if fm.Bottom() >= f.ShipRow() {
		t.Errorf("the block starts already on top of the creature: %d against %d",
			fm.Bottom(), f.ShipRow())
	}
	left, right := fm.Edges()
	if left < 0 || right > f.Cols {
		t.Errorf("the block starts spanning %d..%d of %d columns", left, right, f.Cols)
	}
}

// Below the floor it refuses rather than drawing a mess.
func TestAFieldSmallerThanTheFloorIsRefusedAndNotDrawn(t *testing.T) {
	for _, c := range [][2]int{{59, 24}, {80, 17}, {0, 0}, {-1, -1}} {
		if f, ok := FieldFor(c[0], c[1]); ok {
			t.Errorf("%dx%d was accepted as %+v", c[0], c[1], f)
		}
	}
	for _, c := range [][2]int{{MinCols, MinRows}, {80, 24}, {200, 60}} {
		if _, ok := FieldFor(c[0], c[1]); !ok {
			t.Errorf("%dx%d was refused", c[0], c[1])
		}
	}
}

// The block is the same shape of problem in a narrow window as in a wide one,
// and it always fits: eleven columns is the arcade's number and the most this
// will draw, four the fewest that still reads as a formation.
func TestTheBlockFitsEveryTerminalItAccepts(t *testing.T) {
	for cols := MinCols; cols <= 200; cols++ {
		for _, rows := range []int{MinRows, 24, 40, 60} {
			f := aField(t, cols, rows)
			if got := f.FormationCols(); got < 4 || got > 11 {
				t.Fatalf("%dx%d: %d columns", cols, rows, got)
			}
			if got := f.FormationRows(); got < 2 || got > 5 {
				t.Fatalf("%dx%d: %d rows", cols, rows, got)
			}
			if f.BlockCols() > f.Cols {
				t.Fatalf("%dx%d: the block is %d cells wide", cols, rows, f.BlockCols())
			}
			fm := NewFormation(WaveFor(1, f), f)
			if fm.Bottom() >= f.ShipRow() {
				t.Fatalf("%dx%d: the block starts on the creature", cols, rows)
			}
		}
	}
}

// There is no last wave, so there is no n the recipe may refuse - including the
// clamp at the top, which a corrupt save can reach.
func TestEveryWaveGeneratesNoMatterHowFarYouGet(t *testing.T) {
	f := aField(t, 80, 24)
	for _, n := range []int{-5, 0, 1, 2, 99, 100, 1000, MaxWave, MaxWave + 1} {
		w := WaveFor(n, f)
		if w.HP < 1 || w.Step < 1 || w.Drop < 1 {
			t.Errorf("wave %d came out as %+v", n, w)
		}
		if len(w.Species) != f.FormationRows() {
			t.Errorf("wave %d lines up %d rows for a field that holds %d",
				n, len(w.Species), f.FormationRows())
		}
		for _, s := range w.Species {
			if s < 0 || s >= len(Troops) {
				t.Errorf("wave %d fields species %d of %d", n, s, len(Troops))
			}
		}
	}
}

// A boss every fifth wave, for ever, and one of the canvas's thirty-five.
func TestABossStandsOnEveryFifthWaveForever(t *testing.T) {
	f := aField(t, 80, 24)
	for n := 1; n <= 2000; n++ {
		w := WaveFor(n, f)
		if got, want := w.Boss, n%5 == 0; got != want {
			t.Fatalf("wave %d: boss = %v, want %v", n, got, want)
		}
		if w.Boss && (w.BossOf < 0 || w.BossOf >= len(Bosses)) {
			t.Fatalf("wave %d fields boss %d of %d", n, w.BossOf, len(Bosses))
		}
	}
}

// The ranks climb with the waves, so the first bosses you meet are the canvas's
// larvae and the ones deep in a run are the four it calls jefes.
func TestTheBossRanksClimbWithTheWaves(t *testing.T) {
	f := aField(t, 80, 24)
	prev := 0
	for n := 5; n <= 5*len(Bosses); n += 5 {
		rank := Bosses[WaveFor(n, f).BossOf].Rank
		if rank < prev {
			t.Errorf("wave %d fields a rank %d boss after a rank %d", n, rank, prev)
		}
		prev = rank
	}
	if got := Bosses[WaveFor(5, f).BossOf].Rank; got != 1 {
		t.Errorf("the first boss is rank %d, want a larva", got)
	}
	// And the top rank arrives while a run is still going, rather than after
	// forty-five waves of larvae.
	if got := Bosses[WaveFor(30, f).BossOf].Rank; got != len(Ranks) {
		t.Errorf("wave 30 fields a rank %d boss, want the canvas's jefes by then", got)
	}
}

// The ladder only ever goes up.
func TestTheLadderNeverGetsEasier(t *testing.T) {
	f := aField(t, 80, 24)
	prev := WaveFor(1, f)
	for n := 2; n <= 2000; n++ {
		w := WaveFor(n, f)
		if w.HP < prev.HP || w.Step > prev.Step || w.Drop > prev.Drop {
			t.Fatalf("wave %d is easier than %d:\n%+v\n%+v", n, n-1, w, prev)
		}
		prev = w
	}
	if WaveFor(MaxWave, f).HP <= WaveFor(200, f).HP {
		t.Error("the swarm's HP has a ceiling, which means a run has no ending")
	}
}

// A wave is a thing you can learn: wave twelve is the same twelve every time you
// reach it, whatever seed the run is on.
func TestAWaveIsTheSameLineUpEveryTimeYouReachIt(t *testing.T) {
	f := aField(t, 80, 24)
	for n := 1; n <= 60; n++ {
		a, b := WaveFor(n, f), WaveFor(n, f)
		for i := range a.Species {
			if a.Species[i] != b.Species[i] {
				t.Fatalf("wave %d is not the same twice", n)
			}
		}
	}
}

// The three stack the way the arcade stacks them, and what a later wave changes
// is the colour and the price rather than the shape. That is the trade made by
// using the arcade's bestiary instead of forty of our own.
func TestTheThreeStackTheWayTheArcadeStacksThem(t *testing.T) {
	f := aField(t, 80, 40)
	w := WaveFor(41, f)
	if len(w.Species) < 3 {
		t.Fatalf("only %d rows to check", len(w.Species))
	}
	for i, want := range []string{"squid", "crab", "octopus"} {
		if got := Troops[w.Species[i]].Name; got != want {
			t.Errorf("row %d is a %s, want a %s", i, got, want)
		}
	}
	for _, s := range w.Species[3:] {
		if Troops[s].Name != "octopus" {
			t.Errorf("the rows under the crab are %s, want octopuses", Troops[s].Name)
		}
	}

	// The line-up does not change with the wave; the stage does.
	early, late := WaveFor(1, f), WaveFor(41, f)
	for i := range early.Species {
		if early.Species[i] != late.Species[i] {
			t.Error("the shapes change with the wave, which the arcade's three do not")
		}
	}
	if late.Stage <= early.Stage {
		t.Errorf("wave 41 is stage %d and wave 1 is stage %d", late.Stage, early.Stage)
	}
}

// A squid is worth three octopuses, at every stage, which is the arcade's own
// arithmetic and the reason to shoot the top row first.
func TestTheTopRowPaysBest(t *testing.T) {
	f := aField(t, 80, 40)
	g := NewGame(f, "spark", 1, Save{Wave: 21, Seed: 1}) // not a multiple of five: a boss wave has no block

	byName := map[string]int{}
	for _, m := range g.Squad.Members {
		before := g
		after := before.killMember(m)
		byName[Troops[m.Species].Name] = after.Score - before.Score
	}
	if byName["squid"] <= byName["crab"] || byName["crab"] <= byName["octopus"] {
		t.Errorf("a squid pays %d, a crab %d, an octopus %d",
			byName["squid"], byName["crab"], byName["octopus"])
	}
	// And a deeper stage pays more for the same species.
	shallow := NewGame(f, "spark", 1, Save{Wave: 1, Seed: 1})
	deep := NewGame(f, "spark", 1, Save{Wave: 41, Seed: 1})
	one := func(g Game) int {
		for _, m := range g.Squad.Members {
			if Troops[m.Species].Name == "octopus" {
				return g.killMember(m).Score - g.Score
			}
		}
		return 0
	}
	if one(deep) <= one(shallow) {
		t.Errorf("an octopus pays %d on wave 1 and %d on wave 41", one(shallow), one(deep))
	}
}

// The block walks sideways and steps down at the walls. That is the whole of its
// movement and the reason the game has a clock.
func TestTheBlockWalksSidewaysAndStepsDownAtTheWalls(t *testing.T) {
	g := aGame(t, "spark", 1)
	startY := g.Squad.Y
	moved, dropped := false, false

	for i := 0; i < 6000 && !dropped; i++ {
		before := g.Squad.X
		g = Tick(g, None)
		if g.Squad.X != before {
			moved = true
		}
		if g.Squad.Y > startY {
			dropped = true
		}
	}
	if !moved {
		t.Error("the block never moved sideways")
	}
	if !dropped {
		t.Error("the block never stepped down")
	}
}

// The last few always come down fast. It is the arcade's most famous accident
// and it is kept on purpose: an empty screen must not be a slow one.
func TestAnEmptyingBlockComesDownFaster(t *testing.T) {
	w := WaveFor(1, aField(t, 80, 24))
	full := StepEvery(w, 33, 33)
	few := StepEvery(w, 3, 33)
	if few >= full {
		t.Errorf("a full block steps every %d ticks and the last three every %d", full, few)
	}
	if few < 2 {
		t.Errorf("the last of them step every %d ticks, which is faster than a frame", few)
	}
}

// Clearing a flank gives the rest of the block the whole screen, which is what
// makes shooting the edges first a tactic rather than a habit.
func TestClearingAFlankLetsTheRestUseTheScreen(t *testing.T) {
	f := aField(t, 80, 24)
	fm := NewFormation(WaveFor(1, f), f)
	_, wide := fm.Edges()

	cols := f.FormationCols()
	next := make([]Member, 0, len(fm.Members))
	for _, m := range fm.Members {
		if m.Col != cols-1 {
			next = append(next, m)
		}
	}
	fm.Members = next
	if _, narrow := fm.Edges(); narrow >= wide {
		t.Errorf("dropping the right flank left the right edge at %d, was %d", narrow, wide)
	}
}
