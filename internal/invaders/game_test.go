package invaders

import (
	"strings"
	"testing"
)

// Everything else in this package leans on this. Two runs from one seed and one
// key sequence have to end up in the same state.
func TestTheTickIsDeterministicGivenASeed(t *testing.T) {
	keys := []Key{None, Left, Fire, None, Right, Fire, Ability, Pause, Pause, None}
	play := func() Game {
		g := aGame(t, "bughunter", 4)
		for i := 0; i < 6000; i++ {
			g = Tick(g, keys[i%len(keys)])
		}
		return g
	}
	a, b := play(), play()

	if a.Rand != b.Rand || a.Frame != b.Frame || a.HP != b.HP || a.Score != b.Score {
		t.Fatalf("two runs of one seed diverged:\n%+v\n%+v", a, b)
	}
	if len(a.Squad.Members) != len(b.Squad.Members) || a.Squad.X != b.Squad.X {
		t.Fatalf("the blocks diverged: %d at %v against %d at %v",
			len(a.Squad.Members), a.Squad.X, len(b.Squad.Members), b.Squad.X)
	}
}

// Tick takes a value and returns one. If it wrote through the slices it was
// handed, holding a state and ticking it twice would give two different answers.
func TestTickingTheSameStateTwiceGivesTheSameAnswer(t *testing.T) {
	g := drive(aGame(t, "marathon", 3), Fire, 400)
	a, b := Tick(g, Fire), Tick(g, Fire)

	if len(a.Squad.Members) != len(b.Squad.Members) || len(a.Shots) != len(b.Shots) {
		t.Fatalf("the same state ticked twice gave %d/%d and %d/%d",
			len(a.Squad.Members), len(a.Shots), len(b.Squad.Members), len(b.Shots))
	}
	if a.HP != b.HP || a.Score != b.Score || a.Rand != b.Rand {
		t.Error("two ticks of one state disagree on hp, score or the seed")
	}
}

// The gun is yours. The first draft fired by itself and aimed by itself, which
// left the player one verb and nothing to be good at.
func TestTheGunOnlyFiresWhenYouPressIt(t *testing.T) {
	g := aGame(t, "spark", 3)
	if quiet := drive(g, None, 400); len(quiet.Shots) != 0 {
		t.Errorf("it fired %d shots on its own", len(quiet.Shots))
	}
	if firing := drive(g, Fire, 40); len(firing.Shots) == 0 {
		t.Error("it did not fire when told to")
	}
}

// And it does not aim. A shot leaves the middle of the creature and goes
// straight up: lining the creature up IS the game.
func TestAShotLeavesTheCreatureAndGoesStraightUp(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 20
	g = Tick(g, Fire)
	if len(g.Shots) != 1 {
		t.Fatalf("it fired %d shots", len(g.Shots))
	}
	s := g.Shots[0]
	if s.X < float64(g.Ship) || s.X > float64(g.Ship+ShipCols) {
		t.Errorf("the shot left from column %v and the creature is at %d", s.X, g.Ship)
	}
	was := s.X
	for i := 0; i < 5; i++ {
		g = Tick(g, None)
	}
	if len(g.Shots) > 0 && g.Shots[0].X != was {
		t.Errorf("the shot drifted from %v to %v without homing", was, g.Shots[0].X)
	}
	if len(g.Shots) > 0 && g.Shots[0].Y >= float64(g.Field.ShipRow()) {
		t.Error("the shot is not travelling up")
	}
}

// The rule that makes it a game rather than a hose. Miss, and you wait for the
// shot to reach the top before you may try again.
//
// Measured: without it a level-six title cleared forty-five waves in three
// minutes, four seconds a wave, because nothing limited how much lead was in the
// air. With it the same creature takes ten seconds a wave.
func TestOnlySoManyOfYourShotsMayBeInTheAirAtOnce(t *testing.T) {
	for _, form := range []string{"spark", "weaver", "wasp", "sprinter"} {
		g := NewGame(aField(t, 80, 24), form, 6, Save{Wave: 1, Seed: 3})
		cap := g.Kit.InFlight()
		if cap < 2 || cap > 6 {
			t.Errorf("%s may have %d shots in the air", form, cap)
		}
		for i := 0; i < 600; i++ {
			g = Tick(g, Fire)
			if len(g.Shots) > cap {
				t.Fatalf("%s: %d shots in the air, cap is %d", form, len(g.Shots), cap)
			}
		}
	}
}

// Moving is left and right along the floor, and it stops at the walls.
func TestTheCreatureRunsAlongTheFloorAndStopsAtTheWalls(t *testing.T) {
	g := aGame(t, "spark", 1)
	if left := drive(g, Left, 500); left.Ship != 0 {
		t.Errorf("running left ended at column %d", left.Ship)
	}
	right := drive(g, Right, 500)
	if right.Ship != g.Field.ShipColMax() {
		t.Errorf("running right ended at column %d of %d", right.Ship, g.Field.ShipColMax())
	}
	if right.Field.ShipRow() != g.Field.ShipRow() {
		t.Error("the creature left the floor")
	}
}

// A bomb costs life. It is the slow way to lose, and the one you can dodge.
func TestABombThatReachesYouCostsLife(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 10
	g.Bombs = []Bomb{{X: float64(g.Ship + 4), Y: float64(g.Field.ShipRow()) - 0.2, Hurt: 1}}
	was := g.HP

	g = Tick(g, None)
	if g.HP != was-1 {
		t.Errorf("a bomb cost %d life, want 1", was-g.HP)
	}
	for _, b := range g.Bombs {
		if b.Y >= float64(g.Field.ShipRow()) && b.X >= float64(g.Ship) && b.X < float64(g.Ship+ShipCols) {
			t.Error("the bomb went through and kept going")
		}
	}
}

// A bomb that misses does not. Standing still is a choice, not a sentence.
func TestABombThatMissesCostsNothing(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 10
	g.Bombs = []Bomb{{X: 60, Y: float64(g.Field.ShipRow()) - 0.2, Hurt: 1}}
	was := g.HP
	if g = Tick(g, None); g.HP != was {
		t.Errorf("a bomb three metres away cost %d life", was-g.HP)
	}
}

// They land on you. This is the other way to lose and the one the arcade is
// famous for: bombs whittle you down, but the block arriving is over whatever
// life you had left. It is also what stops a slow gun from waiting a wave out.
func TestWhenTheBlockLandsTheRunIsOver(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Squad.Y = float64(g.Field.ShipRow())
	g = Tick(g, None)

	if g.Phase != Over {
		t.Errorf("the block landed and the run is in %v", g.Phase)
	}
	if g.Banner != BannerLanded {
		t.Errorf("the banner is %q, want the one that says they landed", g.Banner)
	}
	if g.HP != 0 {
		t.Errorf("it left %d life", g.HP)
	}
}

// Clearing the block clears the wave, and a clean one pays for a sloppy one.
func TestClearingTheBlockClearsTheWave(t *testing.T) {
	g := aGame(t, "monk", 3)
	if g.Kit.Regen < 1 {
		t.Fatal("a monk is the regen mark and has none")
	}
	g.HP = 2
	g.Squad.Members = nil

	g = Tick(g, None)
	if g.Phase != Cleared {
		t.Fatalf("the wave did not close: %v", g.Phase)
	}
	if g.HP != 2+g.Kit.Regen {
		t.Errorf("a cleared wave paid %d, want %d", g.HP-2, g.Kit.Regen)
	}
}

// A cleared boss heals you to full: that is what makes every fifth wave a
// rhythm rather than a countdown.
func TestABossClearedHealsToFull(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "sprinter", 4, Save{Wave: 5, Seed: 11})
	if !g.Boss.Alive {
		t.Fatal("wave 5 has no boss")
	}
	g.HP = 1
	g.Boss.Alive = false

	g = Tick(g, None)
	if g.HP != g.Kit.MaxHP {
		t.Errorf("a cleared boss left %d of %d life", g.HP, g.Kit.MaxHP)
	}
	if g.Banner != BannerBossOff {
		t.Errorf("the banner is %q", g.Banner)
	}
}

// A boss is one big sprite off the canvas, it shoots, and it comes down.
func TestTheBossShootsAndDescends(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "spark", 3, Save{Wave: 5, Seed: 7})
	startY, startX := g.Boss.Y, g.Boss.X

	bombs, moved := false, false
	for i := 0; i < 3000 && g.Phase == Playing; i++ {
		g = Tick(g, None)
		if len(g.Bombs) > 0 {
			bombs = true
		}
		if g.Boss.X != startX {
			moved = true
		}
	}
	if !bombs {
		t.Error("the boss never fired")
	}
	if !moved {
		t.Error("the boss never moved")
	}
	if g.Boss.Alive && g.Boss.Y <= startY {
		t.Error("the boss never came down")
	}
}

// The phoenix gets one second life. One.
func TestPhoenixRevivesOnceAndOnlyOnce(t *testing.T) {
	g := aGame(t, "phoenix", 5)
	if !g.Kit.Revive {
		t.Fatal("the phoenix does not carry the revival")
	}
	if g = g.wound(99); g.Phase == Over {
		t.Fatal("it died the first time")
	}
	if !g.Revived || g.HP < 1 {
		t.Error("it did not come back")
	}
	if g = g.wound(99); g.Phase != Over {
		t.Error("it came back twice")
	}

	other := aGame(t, "wasp", 6)
	if other = other.wound(9999); other.Phase != Over {
		t.Error("a form with no revival came back anyway")
	}
}

// Splash goes sideways, along the row. A formation is three rows deep and eleven
// wide, so a vertical splash is either nothing at all or the whole column.
func TestSplashTakesTheNeighboursAlongTheRow(t *testing.T) {
	g := aGame(t, "exterminator", 5)
	if g.Kit.Splash < 1 {
		t.Fatal("the exterminator is the splash mark and has none")
	}
	before := len(g.Squad.Members)

	// A shot placed exactly on the second member of the top row.
	m := g.Squad.Members[1]
	x, y := g.Squad.At(m)
	g.Shots = []Shot{{X: float64(x), Y: float64(y), Damage: 99, Splash: g.Kit.Splash}}
	g = g.resolveHits()

	killed := before - len(g.Squad.Members)
	if killed < 2 {
		t.Errorf("a splash shot killed %d, want it to take neighbours too", killed)
	}
	rows := map[int]bool{}
	for _, left := range g.Squad.Members {
		rows[left.Row] = true
	}
	if len(rows) < 2 {
		t.Error("the splash cleared whole rows, which is a screen-clear and not a splash")
	}
}

// Nothing moves while it is paused, and the same key lets it go again.
func TestAPausedGameDoesNotMoveAnything(t *testing.T) {
	g := drive(aGame(t, "ember", 3), Fire, 300)
	g = Tick(g, Pause)
	if g.Phase != Paused {
		t.Fatalf("it is in %v", g.Phase)
	}
	frozen := g

	g = drive(g, Fire, 200)
	if g.Frame != frozen.Frame || g.HP != frozen.HP || g.Squad.X != frozen.Squad.X {
		t.Error("something moved while it was paused")
	}
	if len(g.Shots) != len(frozen.Shots) {
		t.Error("it kept firing while paused")
	}
	if g = Tick(g, Pause); g.Phase != Playing {
		t.Errorf("it would not resume: %v", g.Phase)
	}
}

// Resuming is at the top of a wave with the life you had.
func TestTheRunResumesAtTheTopOfTheWaveWithTheHpItHad(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "oracle", 5, Save{Wave: 12, HP: 4, Score: 800, Kills: 40, Seed: 5})

	if g.Wave.N != 12 || g.HP != 4 {
		t.Errorf("it resumed on wave %d with %d life", g.Wave.N, g.HP)
	}
	if g.Score != 800 || g.Kills != 40 {
		t.Errorf("it forgot the score: %d/%d", g.Score, g.Kills)
	}
	if len(g.Squad.Members) != g.Started || g.Started == 0 {
		t.Error("it resumed mid-wave rather than at the top of one")
	}
}

// At zero the run ends and the records keep the wave you got to.
func TestAtZeroHpTheRunEndsAndTheRecordsKeepTheBestWave(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Wave = WaveFor(23, g.Field)
	g = g.wound(9999)

	if g.Phase != Over || g.HP != 0 {
		t.Fatalf("it is in %v with %d life", g.Phase, g.HP)
	}
	got := g.ToSave(Save{Wave: 23, BestWave: 10, BestScore: 100, Runs: 2})
	if got.BestWave != 23 {
		t.Errorf("best wave = %d, want the 23 it reached", got.BestWave)
	}
	if got.Wave != 1 || got.HP != g.Kit.MaxHP {
		t.Errorf("the next run starts at wave %d with %d life", got.Wave, got.HP)
	}
	if got.Runs != 3 {
		t.Errorf("runs = %d, want 3", got.Runs)
	}
}

// What a kill is worth is the arcade's value for that species, priced up by how
// deep the stage is.
func TestAKillIsWorthItsSpeciesTimesItsStage(t *testing.T) {
	g := aGame(t, "spark", 1)
	m := g.Squad.Members[0]
	want := Troops[m.Species].Points * g.Wave.Stage

	after := g.killMember(m)
	if got := after.Score - g.Score; got != want {
		t.Errorf("a %s is worth %d, scored %d", Troops[m.Species].Name, want, got)
	}
	if after.Kills != g.Kills+1 {
		t.Error("the kill was not counted")
	}
}

// The tick is where the game's rules live and it may touch nothing outside
// itself. Above all it may not reach pet.json: a run does cost the creature a
// level, but that happens once, in run.go, when the run is over.
func TestTheTickNeverTouchesThePetOrTheTerminal(t *testing.T) {
	src := readSource(t, "game.go")
	// Comments are stripped first: this is a guard on what the code does, and a
	// sentence that happens to end in "at a time." is not an import of time.
	code := withoutComments(src)
	for _, forbidden := range []string{
		"pet.Update(", "pet.Save(", "pet.Setback(",
		"os.", "syscall.", "\\033", "fmt.Print", "time.",
	} {
		if strings.Contains(code, forbidden) {
			t.Errorf("game.go reaches for %q", forbidden)
		}
	}
	for _, allowed := range []string{"pet.Vital", "pet.StateFor"} {
		if !strings.Contains(src, allowed) {
			t.Errorf("game.go no longer uses %q; check this list is still right", allowed)
		}
	}
}

// You have to be able to shoot without stopping. This is the test that says so,
// because it is the difference between a shmup and a turn-based game.
//
// A terminal has no key-up event and cannot say that two keys are held at once:
// hold left and the operating system streams left, press fire and it starts
// repeating fire instead, and the left never comes back until you let go and
// press it again. Momentum for a few tenths of a second was tried first and was
// not enough - hold the fire key and you still coast to a halt - so the creature
// latches.
func TestFiringNeverStopsYouMoving(t *testing.T) {
	g := aGame(t, "spark", 3)
	g.Ship = 40
	g = Tick(g, Left)

	// Nothing but the fire key from here on, for far longer than any coast.
	was := g.Ship
	for i := 0; i < 60; i++ {
		g = Tick(g, Fire)
	}
	if g.Ship >= was-10 {
		t.Errorf("it crawled from %d to %d while firing", was, g.Ship)
	}
	if len(g.Shots) == 0 {
		t.Error("and it did not fire either")
	}
}

// Latched means latched: one arrow and it keeps going, with no further keys.
func TestOneArrowKeepsItGoing(t *testing.T) {
	g := aGame(t, "spark", 3)
	g.Ship = 40
	g = drive(Tick(g, Left), None, 20)
	if g.Ship > 30 {
		t.Errorf("it stopped at column %d after one arrow", g.Ship)
	}
	if g.Drift != -1 {
		t.Errorf("it is drifting %d, want left", g.Drift)
	}
}

// And there has to be a brake, or the only way to stop is to hit a wall.
func TestTheDownArrowStopsIt(t *testing.T) {
	g := aGame(t, "spark", 3)
	g.Ship = 40
	g = drive(Tick(g, Left), None, 6)
	g = Tick(g, Stop)
	stopped := g.Ship

	g = drive(g, None, 30)
	if g.Ship != stopped {
		t.Errorf("it moved from %d to %d after the brake", stopped, g.Ship)
	}
	if g.Drift != 0 {
		t.Errorf("it is still drifting %d", g.Drift)
	}
}

// The other arrow reverses it at once, which is how you line one up.
func TestTheOtherArrowReversesItAtOnce(t *testing.T) {
	g := aGame(t, "spark", 3)
	g.Ship = 40
	g = drive(Tick(g, Left), None, 6)
	left := g.Ship

	g = drive(Tick(g, Right), None, 6)
	if g.Ship <= left {
		t.Errorf("it went from %d to %d after the other arrow", left, g.Ship)
	}
	if g.Drift != 1 {
		t.Errorf("it is drifting %d, want right", g.Drift)
	}
}

// A wall is a stop. Leaving it latched against one would mean the next key you
// press is a key you did not know you had to press.
func TestTheWallStopsIt(t *testing.T) {
	g := aGame(t, "spark", 3)
	g.Ship = 1
	g = drive(Tick(g, Left), None, 30)
	if g.Ship != 0 {
		t.Errorf("it ended at column %d", g.Ship)
	}
	if g.Drift != 0 {
		t.Error("it is still trying to walk into the wall")
	}
}

// withoutComments is the source with its comments taken out, for the guards that
// assert on what a file reaches for rather than on what it says about itself.
func withoutComments(src string) string {
	var out strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// The top row has to be shootable where it starts, and every other row too.
//
// A shot travels more than a row a tick, so testing only the row it landed on
// steps straight over about one row in eleven - and the row it stepped over most
// visibly was the top one, at field row zero, where the next step takes the shot
// off the field and it was thrown away untested. From the outside that is a row
// you cannot kill until the block drops a step, which is how it was reported.
func TestAShotHitsEveryRowItCrosses(t *testing.T) {
	for _, rows := range []int{MinRows, 20, 24, 30, 40} {
		f := aField(t, 80, rows)
		for row := 0; row < f.ShipRow(); row++ {
			g := NewGame(f, "spark", 1, Save{Wave: 1, Seed: 1})
			// One member, alone, on the row under test.
			g.Squad.X, g.Squad.Y = 20, 0
			g.Squad.Members = []Member{{Species: 0, Col: 0, Row: 0, HP: 1}}
			g.Squad.Wait = 1 << 30 // hold the block still
			g.Started = 1
			at, _ := g.Squad.At(g.Squad.Members[0])
			g.Squad.Y = float64(row)

			g.Ship = at - ShipCols/2
			if g.Ship < 0 {
				g.Ship = 0
			}
			g.Shots = []Shot{{X: float64(at), Y: float64(f.ShipRow()), Damage: 5}}

			killed := false
			for i := 0; i < 200 && !killed; i++ {
				g = g.moveShots()
				g = g.resolveHits()
				killed = len(g.Squad.Members) == 0
				if len(g.Shots) == 0 && !killed {
					break
				}
			}
			if !killed {
				t.Errorf("%d rows: a shot went straight through the member on row %d", rows, row)
			}
		}
	}
}

// And a shot that leaves the top of the field is gone rather than lingering at a
// negative row where nothing can be drawn.
func TestAShotThatLeavesTheFieldIsGone(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Squad.Members = nil
	g.Shots = []Shot{{X: 10, Y: 0.5, Damage: 1}}

	g = g.moveShots()
	g = g.resolveHits()
	if len(g.Shots) != 0 {
		t.Errorf("a shot off the top is still in the air at y=%v", g.Shots[0].Y)
	}
}
