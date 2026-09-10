package invaders

import (
	"strings"
	"testing"
)

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

// Everything else in this package leans on this. Two runs from one seed and one
// key sequence have to end up in the same state, or none of the wave, life or
// boss tests are testing anything repeatable - and a resumed run would play a
// different game from the one it was saved in.
func TestTheTickIsDeterministicGivenASeed(t *testing.T) {
	keys := []Key{None, Up, None, None, Fire, Down, None, Pause, Pause, None, Up}
	play := func() Game {
		g := aGame(t, "bughunter", 4)
		for i := 0; i < 5000; i++ {
			g = Tick(g, keys[i%len(keys)])
		}
		return g
	}
	a, b := play(), play()

	if a.Rand != b.Rand || a.Frame != b.Frame || a.HP != b.HP || a.Score != b.Score {
		t.Fatalf("two runs of one seed diverged:\n%+v\n%+v", a, b)
	}
	if len(a.Enemies) != len(b.Enemies) {
		t.Fatalf("%d enemies against %d", len(a.Enemies), len(b.Enemies))
	}
	for i := range a.Enemies {
		if a.Enemies[i] != b.Enemies[i] {
			t.Fatalf("enemy %d:\n%+v\n%+v", i, a.Enemies[i], b.Enemies[i])
		}
	}
}

// Tick takes a value and returns one. If it wrote through the slices it was
// handed, holding a state and ticking it twice would give two different answers,
// and every test in this file that keeps a state around would be lying.
func TestTickingTheSameStateTwiceGivesTheSameAnswer(t *testing.T) {
	g := drive(aGame(t, "marathon", 3), None, 400)
	a, b := Tick(g, Fire), Tick(g, Fire)

	if len(a.Enemies) != len(b.Enemies) || len(a.Shots) != len(b.Shots) {
		t.Fatalf("the same state ticked twice gave %d/%d and %d/%d",
			len(a.Enemies), len(a.Shots), len(b.Enemies), len(b.Shots))
	}
	for i := range a.Enemies {
		if a.Enemies[i] != b.Enemies[i] {
			t.Errorf("enemy %d differs between two ticks of one state", i)
		}
	}
	if a.HP != b.HP || a.Score != b.Score || a.Rand != b.Rand {
		t.Error("two ticks of one state disagree on hp, score or the seed")
	}
}

// It is not killed by arriving. It costs a life and leaves, which is the whole
// difference between a shmup and a lane defender.
func TestAnEnemyReachingTheLeftEdgeCostsExactlyOneHpAndLeaves(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 0
	// Hold the squad back so the only enemy in the field is the planted one;
	// otherwise the wave's own first squad arrives on the same tick.
	g.NextSquad = 9999
	g.Enemies = []Enemy{{Body: Mote, X: 0.5, Row: g.Face() + 3, HP: 99, Speed: 1}}
	was := g.HP

	g = Tick(g, None)
	if len(g.Enemies) != 0 {
		t.Fatalf("it is still there: %+v", g.Enemies)
	}
	if g.HP != was-1 {
		t.Errorf("it cost %d hp, want 1", was-g.HP)
	}
	if g.Kills != 0 {
		t.Error("getting past you counted as a kill")
	}
}

// The crest and the feet are cosmetic. An enemy passes through them; only the
// middle three rows of the five are the ship. This is what lets the ship be the
// creature rather than a block.
func TestTheCrestAndTheFeetAreNotPartOfTheShip(t *testing.T) {
	for _, c := range []struct {
		name string
		row  int
		hurt bool
	}{
		{"the crest", 0, false},
		{"the upper body", HitTop, true},
		{"the face", HitTop + 1, true},
		{"the lower body", HitBottom, true},
		{"the feet", ShipRows - 1, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := aGame(t, "spark", 1)
			g.Ship = 5
			g.Enemies = []Enemy{{Body: Mote, X: 4, Row: g.Ship + c.row, HP: 99, Speed: 0.1}}
			was := g.HP

			g = Tick(g, None)
			if hurt := g.HP < was; hurt != c.hurt {
				t.Errorf("hit on %s: hp went %d -> %d, want hurt = %v",
					c.name, was, g.HP, c.hurt)
			}
		})
	}
}

// A rival's shot costs two, which is what makes a boss wave a duel worth
// respecting rather than a bigger pile of the same thing.
func TestABossShotCostsTwo(t *testing.T) {
	g := aGame(t, "spark", 3)
	g.Ship = 4
	g.Bolts = []Bolt{{X: float64(ShipCols) - 1, Row: g.Ship + HitTop + 1}}
	was := g.HP

	g = Tick(g, None)
	if g.HP != was-boltDrop {
		t.Errorf("a bolt cost %d hp, want %d", was-g.HP, boltDrop)
	}
	if len(g.Bolts) != 0 {
		t.Error("the bolt went through and kept going")
	}
}

// A rival never crosses into the left of the field. It does not leak past you:
// it duels you, and a duel you can walk away from is not one.
func TestABossNeverCrossesIntoTheLeftThird(t *testing.T) {
	f := aField(t, 80, 24)
	left, _ := f.BossBounds()
	g := NewGame(f, "spark", 1, Save{Wave: 5, Seed: 7})
	if len(g.Enemies) != 1 || !g.Enemies[0].Boss() {
		t.Fatalf("wave 5 did not start with a rival: %+v", g.Enemies)
	}
	for i := 0; i < 4000; i++ {
		g = Tick(g, Key(i%3))
		for _, e := range g.Enemies {
			if e.Boss() && e.X < left {
				t.Fatalf("tick %d: the rival reached x=%v, the bound is %v", i, e.X, left)
			}
		}
		if g.Phase == Over {
			break
		}
	}
}

// Life never goes over what the kit allows, whatever pays it: regen between
// waves, a cleared rival, or a save file that claims more than the form can hold.
func TestHpNeverGoesAboveTheKitsMaximum(t *testing.T) {
	g := NewGame(aField(t, 80, 24), "gardener", 6, Save{Wave: 1, HP: 9999, Seed: 3})
	if g.HP > g.Kit.MaxHP {
		t.Fatalf("it started with %d of %d hp", g.HP, g.Kit.MaxHP)
	}
	for i := 0; i < 20000 && g.Phase != Over; i++ {
		g = Tick(g, Fire)
		if g.HP > g.Kit.MaxHP {
			t.Fatalf("tick %d: %d of %d hp", i, g.HP, g.Kit.MaxHP)
		}
	}
}

// A clean wave pays for a sloppy one. Without regen the only way life ever comes
// back is a boss every fifth wave, and the four in between are a slow bleed.
func TestACleanWavePaysForASloppyOne(t *testing.T) {
	g := aGame(t, "monk", 3)
	if g.Kit.Regen < 1 {
		t.Fatal("a monk is the regen mark and has none")
	}
	g.HP = 2
	g.Enemies = nil
	g.Released = g.Wave.Count

	g = Tick(g, None)
	if g.Phase != Cleared {
		t.Fatalf("the wave did not close: %v", g.Phase)
	}
	if g.HP != 2+g.Kit.Regen {
		t.Errorf("a cleared wave paid %d, want %d", g.HP-2, g.Kit.Regen)
	}
}

// A cleared rival heals you to full. That is what makes every fifth wave a
// rhythm rather than a countdown, and it is the run's checkpoint.
func TestABossClearedHealsToFull(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "sprinter", 4, Save{Wave: 5, Seed: 11})
	g.HP = 1
	g.Enemies = nil

	g = Tick(g, None)
	if g.HP != g.Kit.MaxHP {
		t.Errorf("a cleared rival left %d of %d hp", g.HP, g.Kit.MaxHP)
	}
	if g.Banner != BannerBossOff {
		t.Errorf("the banner is %q", g.Banner)
	}
}

// The phoenix gets one second life. One: the flag rides in the save file so
// quitting and coming back does not buy another.
func TestPhoenixRevivesOnceAndOnlyOnce(t *testing.T) {
	g := aGame(t, "phoenix", 5)
	if !g.Kit.Revive {
		t.Fatal("the phoenix does not carry the revival")
	}

	g.HP = 1
	g = g.wound(99)
	if g.Phase == Over {
		t.Fatal("it died the first time")
	}
	if !g.Revived || g.HP < 1 {
		t.Errorf("it came back as %+v", struct {
			Revived bool
			HP      int
		}{g.Revived, g.HP})
	}

	g = g.wound(99)
	if g.Phase != Over {
		t.Error("it came back twice")
	}

	// And nothing else comes back at all.
	other := aGame(t, "wasp", 6)
	if other = other.wound(9999); other.Phase != Over {
		t.Error("a form with no revival came back anyway")
	}
}

// At zero the run ends, the records keep the wave you got to, and the next run
// starts over. There is no wave that ends the game, so this is the only ending.
func TestAtZeroHpTheRunEndsAndTheRecordsKeepTheBestWave(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Wave = WaveFor(23, g.Field)
	g = g.wound(9999)

	if g.Phase != Over || g.HP != 0 {
		t.Fatalf("it is in %v with %d hp", g.Phase, g.HP)
	}
	got := g.ToSave(Save{Wave: 23, BestWave: 10, BestScore: 100, Runs: 2})
	if got.BestWave != 23 {
		t.Errorf("best wave = %d, want the 23 it reached", got.BestWave)
	}
	if got.Wave != 1 || got.HP != g.Kit.MaxHP {
		t.Errorf("the next run starts at wave %d with %d hp", got.Wave, got.HP)
	}
	if got.Runs != 3 {
		t.Errorf("runs = %d, want 3", got.Runs)
	}
	if got.Revived {
		t.Error("the next run starts already revived")
	}
}

// Nothing moves while it is paused, and the same key lets it go again. The
// auto-pause leans on this: Claude finishing must not cost you the wave.
func TestAPausedGameDoesNotMoveAnything(t *testing.T) {
	g := drive(aGame(t, "ember", 3), None, 300)
	g = Tick(g, Pause)
	if g.Phase != Paused {
		t.Fatalf("it is in %v", g.Phase)
	}
	frozen := g

	g = drive(g, None, 200)
	if g.Frame != frozen.Frame || g.HP != frozen.HP || len(g.Enemies) != len(frozen.Enemies) {
		t.Error("something moved while it was paused")
	}
	for i := range g.Enemies {
		if g.Enemies[i] != frozen.Enemies[i] {
			t.Fatalf("enemy %d moved while paused", i)
		}
	}

	g = Tick(g, Pause)
	if g.Phase != Playing {
		t.Errorf("it would not resume: %v", g.Phase)
	}
}

// Resuming is at the top of a wave with the life you had, which is the only
// granularity the save has and the reason the auto-pause is cheap to get right.
func TestTheRunResumesAtTheTopOfTheWaveWithTheHpItHad(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "oracle", 5, Save{Wave: 12, HP: 4, Score: 800, Kills: 40, Seed: 5})

	if g.Wave.N != 12 || g.HP != 4 {
		t.Errorf("it resumed on wave %d with %d hp", g.Wave.N, g.HP)
	}
	if g.Score != 800 || g.Kills != 40 {
		t.Errorf("it forgot the score: %d/%d", g.Score, g.Kills)
	}
	if g.Released != 0 || len(g.Enemies) != 0 {
		t.Error("it resumed mid-wave rather than at the top of one")
	}

	got := drive(g, None, 100).ToSave(Save{Wave: 12, BestWave: 12})
	if got.Wave != 12 {
		t.Errorf("saving mid-wave stored wave %d, want the top of the one it is on", got.Wave)
	}
}

// The one place the bestiary and the kits meet. Plate holds a shot to a single
// point unless it pierces, which is what the cannon family and the sniper mark
// are for - and without it plate is either useless or unbeatable.
func TestOnlyAPiercingShotGetsThroughPlate(t *testing.T) {
	plated := Enemy{Trait: Plated, HP: 20}
	for _, c := range []struct {
		name string
		shot Shot
		want int
	}{
		{"a heavy shot is held to one", Shot{Damage: 9}, 1},
		{"a piercing shot lands in full", Shot{Damage: 9, Pierce: 2}, 9},
		{"a shot doing one is unaffected", Shot{Damage: 1}, 1},
	} {
		if got := plateAdjusted(plated, c.shot); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
	bare := Enemy{Trait: Plain, HP: 20}
	if got := plateAdjusted(bare, Shot{Damage: 9}); got != 9 {
		t.Errorf("an unplated enemy took %d of 9", got)
	}
}

// A splitter that dies to a shot has to leave its children in the field the
// shot resolution is rebuilding. The first draft appended them to the slice it
// was about to overwrite, so a splitter split into nothing.
func TestASplitterKilledByAShotActuallyLeavesChildren(t *testing.T) {
	g := aGame(t, "spark", 1)
	g.Ship = 3
	row := g.Face()
	g.Enemies = []Enemy{{Body: Mote, Trait: Splitter, X: 30, Row: row, HP: 1, Speed: 0}}
	g.Shots = []Shot{{X: 30, Row: row, Damage: 5}}

	g = g.resolveHits()
	if len(g.Enemies) != 2 {
		t.Fatalf("a splitter left %d behind, want two", len(g.Enemies))
	}
	for _, e := range g.Enemies {
		if !e.Split {
			t.Error("a child is not marked as one")
		}
	}
}

// The feral branch's passive: the closer to death, the harder it hits. It is the
// one kit whose damage is not a constant, so it gets its own assertion.
func TestTheOverloadBranchHitsHarderTheCloserItIsToDying(t *testing.T) {
	g := aGame(t, "feral", 3)
	if !g.Kit.Overload {
		t.Fatal("the feral branch does not carry the overload")
	}
	g.HP = g.Kit.MaxHP
	full := g.damageFor()
	g.HP = 1
	hurt := g.damageFor()
	if hurt <= full {
		t.Errorf("at full it does %d and at one hp it does %d", full, hurt)
	}

	other := aGame(t, "marathon", 3)
	other.HP = other.Kit.MaxHP
	a := other.damageFor()
	other.HP = 1
	if b := other.damageFor(); b != a {
		t.Errorf("a form without the overload changed damage with hp: %d then %d", a, b)
	}
}

// The mole's ability is the only thing that refuses damage outright, and it has
// to refuse all of it - a partial invulnerability is just a discount nobody can
// feel through a terminal.
func TestTheMolesAbilityRefusesEverythingWhileItLasts(t *testing.T) {
	g := aGame(t, "mole", 5)
	if g.Kit.Special != AbilityInvuln {
		t.Fatalf("a mole's ability is %q", g.Kit.Special)
	}
	g = Tick(g, Fire)
	if g.Invuln <= 0 {
		t.Fatal("the ability did not come up")
	}
	was := g.HP
	g = g.wound(5)
	if g.HP != was {
		t.Errorf("it took %d damage through the ability", was-g.HP)
	}
}

// The run has to end. There is no last wave any more, so if a dumb autopilot can
// survive for ever then the ladder does not climb and the game has no ending at
// all - which is the failure mode that replaced "nobody reaches wave 99".
//
// This is also the headless proof that the whole tick runs: no tty, one seed,
// thousands of waves' worth of frames.
func TestARunPlaysItselfUntilTheLadderOutgrowsIt(t *testing.T) {
	g := NewGame(aField(t, 80, 24), "wasp", 6, Save{Wave: 1, Seed: 0xBEEF})
	best := 0
	ticks := 0
	for ; ticks < 4_000_000 && g.Phase != Over; ticks++ {
		// Chase the nearest enemy's lane and lean on the ability.
		in := None
		if row, ok := g.nearestEnemy(); ok {
			switch {
			case row < g.Face():
				in = Up
			case row > g.Face():
				in = Down
			}
		}
		if g.Ready == 0 {
			in = Fire
		}
		g = Tick(g, in)

		if g.Wave.N > best {
			best = g.Wave.N
		}
		if g.HP < 0 || g.HP > g.Kit.MaxHP {
			t.Fatalf("tick %d: %d of %d hp", ticks, g.HP, g.Kit.MaxHP)
		}
		if g.Score < 0 || g.Kills < 0 {
			t.Fatalf("tick %d: score %d, kills %d", ticks, g.Score, g.Kills)
		}
		for _, e := range g.Enemies {
			if e.X != e.X || e.Row < 0 || e.Row >= g.Field.Rows {
				t.Fatalf("tick %d: an enemy at row %d, x %v", ticks, e.Row, e.X)
			}
		}
	}
	if g.Phase != Over {
		t.Fatalf("it was still alive on wave %d after %d ticks: the ladder does not climb",
			g.Wave.N, ticks)
	}
	if best < 5 {
		t.Errorf("the autopilot died on wave %d, which says the game is unplayable, not hard", best)
	}
	t.Logf("the autopilot got to wave %d in %d ticks (%d minutes of play)",
		best, ticks, ticks/TicksPerSecond/60)
}

// The tick is where the game's rules live and it may touch nothing outside
// itself. Above all it may not reach pet.json: a run does cost the creature a
// level, but that happens once, in run.go, when the run is over - a tick that
// could reach the pet would punish it twenty times a second.
func TestTheTickNeverTouchesThePetOrTheTerminal(t *testing.T) {
	src := mustRead(t, "game.go")
	for _, forbidden := range []string{
		"pet.Update(", "pet.Save(", "pet.Setback(",
		"os.", "syscall.", "\\033", "fmt.Print", "time.",
	} {
		if strings.Contains(src, forbidden) {
			t.Errorf("game.go reaches for %q", forbidden)
		}
	}
	// It is allowed exactly one thing out of internal/pet, and it is read-only.
	for _, allowed := range []string{"pet.Vital", "pet.StateFor"} {
		if !strings.Contains(src, allowed) {
			t.Errorf("game.go no longer uses %q; check this list is still right", allowed)
		}
	}
}
