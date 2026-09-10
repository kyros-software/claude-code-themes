package invaders

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// oneAlien puts a single ship on the field at a known place, with the wave's
// spawner turned off, which is what most of these tests want: the fleet arrives
// on a clock and a test that waits for it is a test that measures the clock.
func oneAlien(g Game, of int, x, y float64) Game {
	c := Fleet[of]
	// One short of the quota, and the next release pushed out of reach: the wave
	// must not be able to finish, or killing the ship clears the wave in the
	// same tick and the clear hands back the regen the test was measuring.
	g.Released = g.Wave.Count - 1
	g.Next = 1 << 30
	g.Aliens = []Alien{{Of: of, X: x, Y: y, HP: c.HP, MaxHP: c.HP, Fire: c.Cadence}}
	return g
}

// untilLanded ticks until the field is empty, which is the tick the last ship
// reached the floor on - and stops there, before the wave is counted as cleared.
func untilLanded(t *testing.T, g Game) Game {
	t.Helper()
	for i := 0; i < 200; i++ {
		g = Tick(g, None)
		if len(g.Aliens) == 0 {
			return g
		}
	}
	t.Fatalf("nothing landed in two hundred ticks: %+v", g.Aliens)
	return g
}

// The same seed and the same keys have to give the same run, or none of the
// balance measurements below mean anything.
func TestTheTickIsDeterministicGivenASeed(t *testing.T) {
	keys := []Key{Left, Fire, None, Right, Fire, Ability, None, Rearm, Fire, Heal}
	run := func() Game {
		g := aGame(t, "bughunter", 4)
		for i := 0; i < 3000; i++ {
			g = Tick(g, keys[i%len(keys)])
		}
		return g
	}
	a, b := run(), run()
	if a.Rand != b.Rand || a.Score != b.Score || a.HP != b.HP || len(a.Aliens) != len(b.Aliens) {
		t.Errorf("two runs from one seed diverged:\n%+v\n%+v", a.Rand, b.Rand)
	}
	if !reflect.DeepEqual(a.Aliens, b.Aliens) {
		t.Error("the fleet is somewhere else the second time")
	}
}

// Tick takes a value and returns one: ticking the same state twice has to give
// the same answer, which it does not if anything writes through a slice it was
// handed.
func TestTickingTheSameStateTwiceGivesTheSameAnswer(t *testing.T) {
	g := drive(aGame(t, "wasp", 5), Fire, 200)
	a, b := Tick(g, Fire), Tick(g, Fire)
	if !reflect.DeepEqual(a, b) {
		t.Error("the same state ticked twice came out differently: something is shared")
	}
}

// The gun is yours. The first draft fired by itself, which left the player one
// verb and nothing to be good at.
func TestTheGunOnlyFiresWhenYouPressIt(t *testing.T) {
	g := drive(aGame(t, "spark", 1), None, 200)
	if len(g.Shots) != 0 {
		t.Errorf("nobody pressed anything and there are %d shots in the air", len(g.Shots))
	}
	if g.Ammo != g.Kit.Cap {
		t.Errorf("the magazine went from %d to %d without a shot fired", g.Kit.Cap, g.Ammo)
	}
	g = Tick(g, Fire)
	if len(g.Shots) == 0 {
		t.Error("pressing fire fired nothing")
	}
}

func TestAShotLeavesTheShipAndGoesStraightUp(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 30
	g = Tick(g, Fire)
	if len(g.Shots) != 1 {
		t.Fatalf("one press fired %d shots", len(g.Shots))
	}
	s := g.Shots[0]
	if s.X < float64(g.Ship) || s.X > float64(g.Ship+ShipCols) {
		t.Errorf("the shot left from column %g and the ship is at %d", s.X, g.Ship)
	}
	was := s.Y
	g = Tick(g, None)
	if len(g.Shots) != 1 || g.Shots[0].Y >= was {
		t.Errorf("the shot did not travel up: %g then %v", was, g.Shots)
	}
	if g.Shots[0].X != s.X {
		t.Errorf("a straight shot drifted from %g to %g", s.X, g.Shots[0].X)
	}
}

// The magazine is the discipline: you get so many presses and then you stand
// there reloading. Firing on empty starts the reload rather than doing nothing,
// so nobody loses a run to not having read the help row.
func TestTheMagazineEmptiesAndFillsItselfBack(t *testing.T) {
	g := aGame(t, "spark", 1)
	cap := g.Kit.Cap

	fired := 0
	for i := 0; i < 400 && g.Ammo > 0; i++ {
		before := g.Ammo
		g = Tick(g, Fire)
		if g.Ammo < before {
			fired++
		}
	}
	if fired != cap {
		t.Errorf("the magazine held %d rounds and fired %d", cap, fired)
	}

	// Empty: the next press starts a reload, and no shot comes out of it.
	shots := len(g.Shots)
	g = Tick(g, Fire)
	if g.Loading == 0 {
		t.Error("firing on empty did not start a reload")
	}
	if len(g.Shots) > shots {
		t.Error("an empty gun fired anyway")
	}

	g = drive(g, None, g.Kit.Reload+1)
	if g.Ammo != cap {
		t.Errorf("after the reload there are %d of %d rounds", g.Ammo, cap)
	}
}

func TestReloadingEarlyCostsTheSameWaitAndNothingFiresDuringIt(t *testing.T) {
	g := aGame(t, "marathon", 4)
	g = Tick(g, Fire)
	g = Tick(g, Rearm)
	if g.Loading == 0 {
		t.Fatal("r did not start a reload")
	}
	shots := len(g.Shots)
	g = drive(g, Fire, g.Kit.Reload-1)
	if len(g.Shots) > shots+1 {
		t.Errorf("the gun fired %d more shots while it was reloading", len(g.Shots)-shots)
	}
	g = drive(g, None, 2)
	if g.Ammo != g.Kit.Cap {
		t.Errorf("a full reload left %d of %d rounds", g.Ammo, g.Kit.Cap)
	}
}

// A full magazine cannot be reloaded: pressing r out of habit must not cost you
// the wait for nothing.
func TestReloadingAFullMagazineIsNotAWait(t *testing.T) {
	g := aGame(t, "spark", 1)
	g = Tick(g, Rearm)
	if g.Loading != 0 {
		t.Error("r on a full magazine started a reload anyway")
	}
}

func TestTheShipRunsAlongTheFloorAndStopsAtTheWalls(t *testing.T) {
	g := aGame(t, "spark", 1)
	g = drive(g, Left, 400)
	if g.Ship != 0 {
		t.Errorf("all the way left is column %d", g.Ship)
	}
	g = drive(g, Right, 400)
	if g.Ship != g.Field.ShipColMax() {
		t.Errorf("all the way right is column %d of %d", g.Ship, g.Field.ShipColMax())
	}
}

// A tap is a tap: one press, one cell, and then it stays where you put it.
//
// This is the whole of what changed from the version before. That one latched -
// one press and the ship went until you said otherwise - which is smooth and is
// not what an arrow key means.
func TestATapMovesExactlyOneCellAndStops(t *testing.T) {
	g := aGame(t, "spark", 1)
	col := g.Ship
	g = Tick(g, Left)
	if g.Ship != col-1 {
		t.Errorf("one press moved it %d columns, want one", col-g.Ship)
	}
	g = drive(g, None, 60)
	if g.Ship != col-1 {
		t.Errorf("it drifted on to column %d after the key was let go", g.Ship)
	}
}

// A key held down is a STREAM of presses - a terminal has no other way to say it
// - and that is what glides. Letting go stops it within a frame or two.
func TestHoldingAnArrowGlidesAndLettingGoStops(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	g.Ship = 40

	held := g
	for i := 0; i < 20; i++ { // the autorepeat, arriving every tick
		held = Tick(held, Left)
	}
	if moved := 40 - held.Ship; moved < 18 {
		t.Errorf("twenty ticks of a held arrow moved %d columns", moved)
	}

	parked := held.Ship
	held = drive(held, None, 40)
	if skid := parked - held.Ship; skid > streamMax {
		t.Errorf("it skidded %d columns after the key was let go", skid)
	}
	if held.Side.Moving() {
		t.Error("it is still travelling with nothing pressed")
	}
}

// And two taps far apart are two taps, not a hold: the glide only starts when a
// press arrives while the last one is still recent.
func TestTwoTapsFarApartAreNotAHold(t *testing.T) {
	g := aGame(t, "spark", 1)
	col := g.Ship
	g = Tick(g, Left)
	g = drive(g, None, repeatWindow+4)
	g = Tick(g, Left)
	g = drive(g, None, 40)
	if g.Ship != col-2 {
		t.Errorf("two taps moved %d columns, want two", col-g.Ship)
	}
}

// The other arrow turns it round at once, without having to stop first.
func TestTheOtherArrowTurnsItRoundAtOnce(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	g.Ship = 40
	for i := 0; i < 10; i++ {
		g = Tick(g, Left)
	}
	low := g.Ship
	for i := 0; i < 10; i++ {
		g = Tick(g, Right)
	}
	if g.Ship <= low {
		t.Errorf("it turned round and reached column %d, having been at %d", g.Ship, low)
	}
}

// Firing must never stop you moving. This is the bug the user reported twice, and
// the reason the glide is on a leash rather than needing a key every single tick:
// the stream and the shots interleave, and the ship keeps going between them.
func TestFiringNeverStopsYouMoving(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	g.Ship = 40
	shots, col := 0, g.Ship
	for i := 0; i < 120; i++ {
		before := len(g.Shots)
		in := Left
		if i%2 == 1 {
			in = Fire
		}
		g = Tick(g, in)
		if len(g.Shots) > before {
			shots++
		}
	}
	if moved := col - g.Ship; moved < 20 {
		t.Errorf("a hundred and twenty ticks of moving and firing moved %d columns", moved)
	}
	if shots == 0 {
		t.Error("it moved and never fired")
	}
}

func TestABombThatReachesYouCostsLife(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 20
	hp := g.HP
	g.Bombs = []Bomb{{X: 22, Y: float64(g.Field.ShipRow()) - 1, Hurt: bombDrop}}
	g = drive(g, None, 8)
	if g.HP != hp-bombDrop {
		t.Errorf("a bomb on the hull took %d hp, want %d", hp-g.HP, bombDrop)
	}
}

func TestABombThatMissesCostsNothing(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 20
	hp := g.HP
	g.Bombs = []Bomb{{X: 2, Y: float64(g.Field.ShipRow()) - 1, Hurt: bombDrop}}
	g = drive(g, None, 20)
	if g.HP != hp {
		t.Errorf("a bomb at the far end of the row took %d hp", hp-g.HP)
	}
}

// A ship that gets to the floor costs life and is gone. It is not the end of the
// run: a fleet is individuals, and letting one through is a mistake rather than
// a defeat.
func TestAShipThatReachesTheFloorCostsLifeAndLeaves(t *testing.T) {
	g := aGame(t, "marathon", 5)
	g.Ship = 0
	hp := g.HP
	g = oneAlien(g, 0, 40, float64(g.Field.ShipRow())-1)

	// Only as far as the landing. One tick past it the wave is empty, and an
	// empty wave is a cleared wave, which hands back the regen - so a test that
	// drives on for a while measures nothing.
	g = untilLanded(t, g)
	if g.HP != hp-landDrop {
		t.Errorf("one through cost %d hp, want %d", hp-g.HP, landDrop)
	}
	if g.Phase == Over {
		t.Error("one ship through ended the run")
	}
}

// One that comes down on top of you costs more than one that slips past at the
// far end of the row.
func TestOneThatLandsOnYouCostsMoreThanOneThatSlipsPast(t *testing.T) {
	on := func(x float64) int {
		g := aGame(t, "marathon", 5)
		g.Ship = 20
		hp := g.HP
		g = oneAlien(g, 0, x, float64(g.Field.ShipRow())-1)
		g = untilLanded(t, g)
		return hp - g.HP
	}
	if ram, past := on(21), on(60); ram <= past {
		t.Errorf("a ram cost %d and one past the far end cost %d", ram, past)
	}
}

// A boss reaching the floor is over, whatever life was left: that is the fight
// lost, and it is the other way to end a run.
func TestABossThatLandsEndsTheRun(t *testing.T) {
	g := aGame(t, "wasp", 6)
	g = g.startWave(5)
	if !g.Boss.Alive {
		t.Fatal("wave five has no boss")
	}
	g.Boss.Y = float64(g.Field.Rows - BossRows)
	g = Tick(g, None)
	if g.Phase != Over {
		t.Errorf("a boss on the floor left the run in %v", g.Phase)
	}
	if g.Banner != BannerLanded {
		t.Errorf("the banner is %q", g.Banner)
	}
}

// A wave is over when everything it was going to release has been released and
// killed, and then the next one starts on its own.
func TestClearingTheWaveMovesOnToTheNext(t *testing.T) {
	g := aGame(t, "wasp", 6)
	g.Released = g.Wave.Count
	g = Tick(g, None)
	if g.Phase != Cleared {
		t.Fatalf("an empty wave left the run in %v", g.Phase)
	}
	if g.Banner != BannerCleared {
		t.Errorf("the banner is %q", g.Banner)
	}
	g = drive(g, None, clearedFor+2)
	if g.Wave.N != 2 || g.Phase != Playing {
		t.Errorf("after the rest it is wave %d in %v", g.Wave.N, g.Phase)
	}
}

func TestABossClearedHealsToFull(t *testing.T) {
	g := aGame(t, "wasp", 6)
	g = g.startWave(5)
	g.HP = 1
	g.Released = g.Wave.Count
	g.Boss.HP = 1
	// Two rows into the sprite, not one: a shot travels 1.1 rows a tick, so one
	// fired at the boss's own top row is above it by the time hits are resolved.
	g.Shots = []Shot{{X: g.Boss.X + 1, Y: g.Boss.Y + 3, Damage: 50}}
	g = drive(g, None, 3)
	if g.Boss.Alive {
		t.Fatal("the boss survived fifty damage")
	}
	if g.HP != g.Kit.MaxHP {
		t.Errorf("a cleared boss left %d of %d hp", g.HP, g.Kit.MaxHP)
	}
}

func TestTheBossShootsAndDescends(t *testing.T) {
	g := aGame(t, "wasp", 6)
	g = g.startWave(5)
	y, x := g.Boss.Y, g.Boss.X
	// A boss leans down a row every hundred and eighty ticks, which is four and a
	// half seconds at forty a second.
	g = drive(g, None, 200)
	if g.Boss.Y <= y {
		t.Errorf("the boss is still at row %g", g.Boss.Y)
	}
	if g.Boss.X == x {
		t.Error("the boss never moved sideways")
	}
	if len(g.Bombs) == 0 {
		t.Error("the boss never fired")
	}
}

func TestPhoenixRevivesOnceAndOnlyOnce(t *testing.T) {
	g := aGame(t, "phoenix", 6)
	if !g.Kit.Revive {
		t.Fatal("the phoenix has no revival")
	}
	g.HP = 1
	g = g.wound(50)
	if g.Phase == Over {
		t.Fatal("the phoenix died the first time")
	}
	if !g.Revived || g.HP <= 0 {
		t.Errorf("revived=%v with %d hp", g.Revived, g.HP)
	}
	g = g.wound(50)
	if g.Phase != Over {
		t.Error("the phoenix rose twice")
	}
}

func TestAPausedGameDoesNotMoveAnything(t *testing.T) {
	g := drive(aGame(t, "bughunter", 4), Fire, 300)
	g = Tick(g, Pause)
	if g.Phase != Paused {
		t.Fatalf("p left it in %v", g.Phase)
	}
	still := drive(g, Fire, 100)
	if !reflect.DeepEqual(g, still) {
		t.Error("a hundred ticks of a paused game changed something")
	}
	g = Tick(g, Pause)
	if g.Phase != Playing {
		t.Errorf("p again left it in %v", g.Phase)
	}
}

func TestTheRunResumesAtTheTopOfTheWaveWithTheHpItHad(t *testing.T) {
	g := aGame(t, "wasp", 6)
	g.HP = 3
	g.Score = 900
	g.Wave = WaveFor(12, g.Field)
	save := g.ToSave(Save{})
	if save.Wave != 12 || save.HP != 3 || save.Score != 900 {
		t.Fatalf("the save is %+v", save)
	}
	back := NewGame(g.Field, "wasp", 6, save)
	if back.Wave.N != 12 || back.HP != 3 || back.Score != 900 {
		t.Errorf("it came back as wave %d with %d hp and %d points",
			back.Wave.N, back.HP, back.Score)
	}
	if back.Released != 0 || len(back.Aliens) != 0 {
		t.Error("it came back in the middle of a wave")
	}
}

func TestAtZeroHpTheRunEndsAndTheRecordsKeepTheBestWave(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Wave = WaveFor(9, g.Field)
	g.Score = 1234
	g = g.wound(g.HP)
	if g.Phase != Over {
		t.Fatalf("zero hp left it in %v", g.Phase)
	}
	save := g.ToSave(Save{Runs: 2})
	if save.Wave != 1 || save.Score != 0 {
		t.Errorf("the next run starts at wave %d with %d points", save.Wave, save.Score)
	}
	if save.BestWave != 9 || save.BestScore != 1234 || save.Runs != 3 {
		t.Errorf("the records are %+v", save)
	}
}

// A kill is worth its ship's own value times how deep the stage is, so the same
// drone pays more in a later wave.
func TestAKillIsWorthItsShipTimesItsStage(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Wave = WaveFor(30, g.Field) // stage 8
	a := Alien{Of: 0, X: 10, Y: 4, HP: 1, MaxHP: 1}
	before := g.Score
	g = g.killAlien(a)
	if want := Fleet[0].Points * g.Wave.Stage; g.Score-before != want {
		t.Errorf("a %s in stage %d paid %d, want %d",
			Fleet[0].Name, g.Wave.Stage, g.Score-before, want)
	}
	if g.Kills != 1 {
		t.Errorf("%d kills", g.Kills)
	}
}

// The score buys upgrades, and taking one stops the game: a choice made while a
// bomb is in the air is not a choice.
func TestTheScoreBuysAnUpgradeAndTheGameStopsToAsk(t *testing.T) {
	g := aGame(t, "spark", 1)
	if g.NextUp <= 0 {
		t.Fatalf("the first upgrade is at %d points", g.NextUp)
	}
	g.Score = g.NextUp
	g = Tick(g, None)
	if g.Phase != Choosing {
		t.Fatalf("passing the mark left it in %v", g.Phase)
	}
	if g.Banner != BannerChoose {
		t.Errorf("the banner is %q", g.Banner)
	}

	frozen := drive(g, Fire, 50)
	if !reflect.DeepEqual(g, frozen) {
		t.Error("the field moved while the menu was up")
	}

	damage := g.Kit.Damage
	g = Tick(g, One)
	if g.Phase != Playing {
		t.Errorf("picking one left it in %v", g.Phase)
	}
	if g.Kit.Damage != damage+1 {
		t.Errorf("power went from %d to %d", damage, g.Kit.Damage)
	}
	if g.NextUp <= g.Score {
		t.Error("the next upgrade is already paid for")
	}
}

func TestEachUpgradeChangesTheGunItPromises(t *testing.T) {
	base := aGame(t, "bughunter", 4).Kit
	if got := boost(base, UpPower); got.Damage != base.Damage+1 {
		t.Errorf("power: damage %d, was %d", got.Damage, base.Damage)
	}
	if got := boost(base, UpSpeed); got.Cadence >= base.Cadence {
		t.Errorf("rate: cadence %d, was %d", got.Cadence, base.Cadence)
	}
	if got := boost(base, UpCap); got.Cap <= base.Cap || got.Reload >= base.Reload {
		t.Errorf("magazine: %d rounds in %d ticks, was %d in %d",
			got.Cap, got.Reload, base.Cap, base.Reload)
	}
	// And they stack, because a run is twenty waves long.
	k := base
	for i := 0; i < 5; i++ {
		k = boost(k, UpPower)
	}
	if k.Damage != base.Damage+5 {
		t.Errorf("five upgrades gave %d damage, was %d", k.Damage, base.Damage)
	}
}

// The gun you built survives being quit and resumed, which happens every turn
// the arena pauses: coming back with the pet's bare kit would read as the game
// having forgotten.
func TestTheGunYouBuiltComesBackAfterASave(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	for _, pick := range []Key{One, Two, Three, One} {
		g.Phase = Choosing
		g = Tick(g, pick)
	}
	built := g.Kit

	save := g.ToSave(Save{})
	if save.Power != 2 || save.Speed != 1 || save.Mag != 1 {
		t.Fatalf("the save remembers %d power, %d rate, %d magazine",
			save.Power, save.Speed, save.Mag)
	}
	back := NewGame(g.Field, "bughunter", 4, save)
	if back.Kit != built {
		t.Errorf("the gun came back different:\nwas  %+v\nnow  %+v", built, back.Kit)
	}
}

// A health kit falls, is caught by flying into it, and is spent when you ask.
func TestAHealthKitIsCaughtAndThenSpent(t *testing.T) {
	g := aGame(t, "marathon", 4)
	g.Ship = 20
	g.HP = 2
	g.Drops = []Drop{{X: 22, Y: float64(g.Field.ShipRow())}}

	g = Tick(g, None)
	if g.Kits != 1 {
		t.Fatalf("flying into a kit put %d in the hold", g.Kits)
	}
	if len(g.Drops) != 0 {
		t.Error("the kit is still falling")
	}
	hp := g.HP
	g = Tick(g, Heal)
	if g.Kits != 0 || g.HP <= hp {
		t.Errorf("spending it left %d kits and %d hp, was %d", g.Kits, g.HP, hp)
	}
	// And there is nothing to spend now.
	g = Tick(g, Heal)
	if g.Kits != 0 {
		t.Errorf("%d kits out of an empty hold", g.Kits)
	}
}

// A kit is not thrown away on full life: it stays in the hold for when it is
// worth something.
func TestAKitIsNotWastedOnFullLife(t *testing.T) {
	g := aGame(t, "marathon", 4)
	g.Kits = 1
	g = Tick(g, Heal)
	if g.Kits != 1 {
		t.Error("a kit was spent on a full hull")
	}
}

// An asteroid is on nobody's side: breaking one throws meteoroids that hurt the
// fleet and hurt you.
func TestABrokenAsteroidThrowsMeteoroidsAtEverybody(t *testing.T) {
	g := aGame(t, "marathon", 5)
	g = g.breakStone(Stone{X: 30, Y: 6, HP: 0, MaxHP: Rock.HP})
	if len(g.Motes) == 0 {
		t.Fatal("a broken rock threw nothing")
	}
	hurting := 0
	for _, m := range g.Motes {
		if m.Hurt > 0 {
			hurting++
		}
	}
	if hurting == 0 {
		t.Error("the meteoroids are decoration: none of them hurts anything")
	}

	// One of them, put on the hull by hand, costs life.
	hp := g.HP
	g.Ship = 20
	g.Motes = []Mote{{X: 22, Y: float64(g.Field.ShipRow() + 1), Life: 5, Hurt: 1}}
	g = Tick(g, None)
	if g.HP >= hp {
		t.Error("a meteoroid through the hull cost nothing")
	}
}

// Breaking a rock pays nothing, or the safest way to farm the game is to stand
// still and shoot stones.
func TestARockIsNotAKill(t *testing.T) {
	g := aGame(t, "marathon", 5)
	score, kills := g.Score, g.Kills
	g = g.breakStone(Stone{X: 30, Y: 6})
	if g.Score != score || g.Kills != kills {
		t.Errorf("a rock paid %d points and %d kills", g.Score-score, g.Kills-kills)
	}
}

// A shot hits what it passes through. It travels 1.1 rows a tick, so this is
// only safe because every craft is at least two rows tall - the guard for that
// lives in the bestiary's tests, and this is the other half of it.
func TestAShotHitsTheShipItPassesThrough(t *testing.T) {
	for of, c := range Fleet {
		g := aGame(t, "spark", 1)
		g.Ship = 20
		g = oneAlien(g, of, 21, 4)
		g.Aliens[0].HP = 99
		g.Aliens[0].MaxHP = 99
		g.Shots = []Shot{{X: 22, Y: 12, Damage: 1}}

		hit := false
		for i := 0; i < 20 && len(g.Aliens) > 0; i++ {
			g = Tick(g, None)
			if g.Aliens[0].HP < 99 {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("a shot went straight through a %s without touching it", c.Name)
		}
	}
}

func TestAShotThatLeavesTheFieldIsGone(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Released = g.Wave.Count
	g = Tick(g, Fire)
	g = drive(g, None, g.Field.Rows+5)
	if len(g.Shots) != 0 {
		t.Errorf("%d shots are still in the air above the field", len(g.Shots))
	}
}

// The tick may not reach the pet or the terminal. The run does cost the creature
// a level, but that happens once, in run.go, when it is over - a tick that could
// reach pet.json would punish it twenty times a second.
func TestTheTickNeverTouchesThePetOrTheTerminal(t *testing.T) {
	raw, err := os.ReadFile("game.go")
	if err != nil {
		t.Fatal(err)
	}
	body := stripComments(string(raw))
	for _, forbidden := range []string{
		"os.", "syscall.", "\\033", "pet.Update(", "pet.Save(", "pet.Path(",
		"time.", "fmt.", "i18n.",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("game.go mentions %q: the tick is meant to be pure", forbidden)
		}
	}
}

// stripComments takes the prose out, so a test that scans for "os." is not
// tripped by a comment saying "at a time."
func stripComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// The ship climbs, and it stops at the roof rather than at the top of the field.
// Half the screen is the player's; the other half is where the fleet comes from,
// and a ship that could sit on the spawn line would shoot every ship before it
// had drawn a frame.
func TestTheShipClimbsAsFarAsTheRoofAndNoFurther(t *testing.T) {
	g := aGame(t, "spark", 1)
	if g.Row != g.Field.ShipRow() {
		t.Fatalf("a run starts at row %d and the floor is %d", g.Row, g.Field.ShipRow())
	}

	for i := 0; i < 200; i++ {
		g = Tick(g, Up)
	}
	if g.Row != g.Field.ShipRoof() {
		t.Errorf("all the way up is row %d, want the roof at %d", g.Row, g.Field.ShipRoof())
	}
	if g.Rise.Moving() {
		t.Error("it is still trying to climb through the roof")
	}
	if g.Field.ShipRoof() <= 0 {
		t.Error("the roof is the top of the field, so there is no descent left")
	}

	for i := 0; i < 200; i++ {
		g = Tick(g, Down)
	}
	if g.Row != g.Field.ShipRow() {
		t.Errorf("all the way down is row %d, want the floor at %d", g.Row, g.Field.ShipRow())
	}
}

// A diagonal is two keys held at once, which a terminal cannot report - so what
// arrives is the two streams interleaved, and each axis has to keep its glide
// through the other one's presses.
func TestTwoStreamsInterleavedMakeADiagonal(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	col, row := g.Ship, g.Row
	for i := 0; i < 60; i++ {
		in := Left
		if i%2 == 1 {
			in = Up
		}
		g = Tick(g, in)
	}
	if g.Ship >= col {
		t.Errorf("it went from column %d to %d", col, g.Ship)
	}
	if g.Row >= row {
		t.Errorf("it went from row %d to %d", row, g.Row)
	}
}

// The brake stops both axes at once. It is nearly redundant now that letting go
// stops you, and it is kept because a stream that jams - or a terminal that
// repeats a key after it was released - is a ship nobody can park.
func TestTheBrakeStopsBothAxes(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	for i := 0; i < 20; i++ {
		in := Left
		if i%2 == 1 {
			in = Up
		}
		g = Tick(g, in)
	}
	g = Tick(g, Stop)
	col, row := g.Ship, g.Row
	g = drive(g, None, 40)
	if g.Ship != col || g.Row != row {
		t.Errorf("after the brake it drifted from %d,%d to %d,%d", col, row, g.Ship, g.Row)
	}
	if g.Side.Moving() || g.Rise.Moving() {
		t.Errorf("the brake left it going %d,%d", g.Side.Way, g.Rise.Way)
	}
}

// Climbing is slower than strafing, because a terminal cell is about twice as
// tall as it is wide and a row a tick reads as twice the speed.
func TestClimbingIsSlowerThanStrafing(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 30
	across, up := g, g
	for i := 0; i < 40; i++ {
		across = Tick(across, Left)
		up = Tick(up, Up)
	}
	if cols, rows := 30-across.Ship, g.Row-up.Row; rows >= cols {
		t.Errorf("in forty ticks it moved %d columns and %d rows", cols, rows)
	}
}

// Everything that can hit the ship has to follow it up the field: the hitbox, the
// muzzle and the ram.
func TestWhatCanHitTheShipFollowsItUpTheField(t *testing.T) {
	g := aGame(t, "marathon", 4)
	g.Ship = 20
	g.Row = g.Field.ShipRoof()

	// A bomb where the ship used to be is a bomb that misses.
	hp := g.HP
	g.Bombs = []Bomb{{X: 22, Y: float64(g.Field.ShipRow()) - 1, Hurt: 1}}
	g = drive(g, None, 20)
	if g.HP != hp {
		t.Errorf("a bomb aimed at the floor hit a ship that had climbed: %d hp of %d", g.HP, hp)
	}

	// One where it is now is a bomb that lands.
	g.Bombs = []Bomb{{X: 22, Y: float64(g.Row) - 1, Hurt: 1}}
	g = drive(g, None, 20)
	if g.HP >= hp {
		t.Error("a bomb on the hull cost nothing")
	}

	// And the shots leave from where it is.
	g.Ammo = g.Kit.Cap
	g.Cool = 0
	g = Tick(g, Fire)
	if len(g.Shots) == 0 {
		t.Fatal("it did not fire")
	}
	if got := g.Shots[0].Y; got > float64(g.Row) {
		t.Errorf("the shot left from row %g and the ship is at %d", got, g.Row)
	}
}

// Flying into something costs you, wherever you did it. Climbing is not a way to
// take the fleet's ships out of play.
func TestClimbingIntoAShipIsARamWhereverItHappens(t *testing.T) {
	g := aGame(t, "marathon", 5)
	g.Ship = 20
	g.Row = g.Field.ShipRoof() + 2
	hp := g.HP
	g = oneAlien(g, 0, 21, float64(g.Row)-1)

	g = untilLanded(t, g)
	if g.HP != hp-ramDrop {
		t.Errorf("a ram in the middle of the field cost %d, want %d", hp-g.HP, ramDrop)
	}
}

// The row survives everything the field can do to it, including a window that
// changes size under a run.
func TestTheShipStaysBetweenTheRoofAndTheFloorForAWholeRun(t *testing.T) {
	g := aGame(t, "wasp", 6)
	keys := []Key{Up, Up, Fire, Left, Left, Fire, Down, Down, Fire, Right, Stop, Up, Fire}
	for i := 0; i < 6000 && g.Phase != Over; i++ {
		g = Tick(g, keys[i%len(keys)])
		if g.Row < g.Field.ShipRoof() || g.Row > g.Field.ShipRow() {
			t.Fatalf("tick %d: the ship is at row %d, and its half is %d..%d",
				i, g.Row, g.Field.ShipRoof(), g.Field.ShipRow())
		}
	}
}
