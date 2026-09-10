package invaders

import "testing"

// The ship stands on the floor and the fleet arrives at the top, which is the
// whole geometry of the thing.
func TestTheShipStandsOnTheFloorAndTheFleetArrivesAtTheTop(t *testing.T) {
	f := aField(t, 80, 24)
	if f.Rows != 24-HUDRows-HelpRows {
		t.Errorf("a 24-row terminal gives a field of %d rows", f.Rows)
	}
	if got, want := f.ShipRow(), f.Rows-ShipRows; got != want {
		t.Errorf("the ship starts at row %d, want %d", got, want)
	}
	if f.ShipRow()+ShipRows != f.Rows {
		t.Error("the ship does not reach the floor")
	}
	if got := f.ShipColMax(); got != f.Cols-ShipCols {
		t.Errorf("the ship may reach column %d of %d", got, f.Cols)
	}
}

// A terminal below the floor is refused rather than drawn badly: half a game in
// a window too small to dodge in is worse than a sentence saying so.
func TestAFieldSmallerThanTheFloorIsRefusedAndNotDrawn(t *testing.T) {
	for _, c := range [][2]int{{59, 24}, {80, 17}, {40, 10}, {0, 0}} {
		if _, ok := FieldFor(c[0], c[1]); ok {
			t.Errorf("%dx%d was accepted and the floor is %dx%d",
				c[0], c[1], MinCols, MinRows)
		}
	}
	if _, ok := FieldFor(MinCols, MinRows); !ok {
		t.Errorf("%dx%d is the floor and it was refused", MinCols, MinRows)
	}
}

// The biggest ship in the fleet has to fit in the narrowest terminal, with room
// left to get out from under it.
func TestTheWholeFleetFitsEveryTerminalItAccepts(t *testing.T) {
	f := aField(t, MinCols, MinRows)
	for _, c := range append(append([]Craft{}, Fleet...), Rock) {
		if c.W > f.Cols/2 {
			t.Errorf("%s is %d cells wide in a field of %d: there is nowhere to dodge",
				c.Name, c.W, f.Cols)
		}
		if c.H > f.ShipRow()/2 {
			t.Errorf("%s is %d rows tall and the ship is at row %d", c.Name, c.H, f.ShipRow())
		}
	}
	if BossCols > f.Cols/3 || BossRows > f.ShipRow()/2 {
		t.Errorf("a boss is %dx%d in a field of %dx%d", BossCols, BossRows, f.Cols, f.Rows)
	}
}

// There is no last wave, so there is no wave the recipe may refuse: a run that
// gets further than anybody expected must not walk into a division by zero.
func TestEveryWaveGeneratesNoMatterHowFarYouGet(t *testing.T) {
	f := aField(t, 80, 24)
	for _, n := range []int{1, 2, 5, 19, 100, 999, 5000, MaxWave, MaxWave + 1, 0, -7} {
		w := WaveFor(n, f)
		if w.Count < 1 {
			t.Errorf("wave %d releases %d ships", n, w.Count)
		}
		if w.Every < 1 {
			t.Errorf("wave %d releases one every %d ticks", n, w.Every)
		}
		if w.Stage < 1 || w.Stage > stagesDeep {
			t.Errorf("wave %d is stage %d", n, w.Stage)
		}
		if len(w.Unlocked()) == 0 {
			t.Errorf("wave %d has nothing to draw from", n)
		}
		for _, of := range w.Unlocked() {
			if of < 0 || of >= len(Fleet) {
				t.Errorf("wave %d may spawn ship %d of %d", n, of, len(Fleet))
			}
		}
	}
}

// A boss every fifth wave, for ever: there is no final boss because there is no
// final wave, so what closes a wave is a checkpoint rather than an ending.
func TestABossStandsOnEveryFifthWaveForever(t *testing.T) {
	f := aField(t, 80, 24)
	for n := 1; n <= 500; n++ {
		w := WaveFor(n, f)
		if want := n%5 == 0; w.Boss != want {
			t.Fatalf("wave %d: boss=%v, want %v", n, w.Boss, want)
		}
		if !w.Boss {
			continue
		}
		if w.BossHP <= 0 {
			t.Errorf("wave %d has a boss with %d hp", n, w.BossHP)
		}
		if w.BossOf < 0 || w.BossOf >= len(Bosses) {
			t.Errorf("wave %d wants boss %d of %d", n, w.BossOf, len(Bosses))
		}
		// A boss and a full wave at once is noise, not a fight.
		if w.Count > 6 {
			t.Errorf("wave %d puts a boss and %d ships on the field", n, w.Count)
		}
	}
}

// The ranks climb with the waves, so the first bosses are the canvas's larvae
// and the four it calls jefes turn up while a run is still going.
func TestTheBossRanksClimbWithTheWaves(t *testing.T) {
	f := aField(t, 80, 24)
	first := Bosses[WaveFor(5, f).BossOf].Rank
	last := Bosses[WaveFor(120, f).BossOf].Rank
	if first != 1 {
		t.Errorf("the first boss is rank %d, want the larvae", first)
	}
	if last <= first {
		t.Errorf("wave 120's boss is rank %d and wave 5's was %d", last, first)
	}
	best := 0
	for n := 5; n <= 200; n += 5 {
		if r := Bosses[WaveFor(n, f).BossOf].Rank; r > best {
			best = r
		}
	}
	if best != len(Ranks) {
		t.Errorf("the deepest rank a run reaches is %d of %d", best, len(Ranks))
	}
}

// The ladder never gets easier. Three axes climb and none of them may dip, or
// there is a wave somewhere that is a rest.
func TestTheLadderNeverGetsEasier(t *testing.T) {
	f := aField(t, 80, 24)
	prev := WaveFor(1, f)
	for n := 2; n <= 2000; n++ {
		w := WaveFor(n, f)
		if w.Tough < prev.Tough {
			t.Fatalf("wave %d ships are softer than wave %d's", n, n-1)
		}
		if w.Haste < prev.Haste {
			t.Fatalf("wave %d comes down slower than wave %d", n, n-1)
		}
		if w.Every > prev.Every {
			t.Fatalf("wave %d releases slower than wave %d", n, n-1)
		}
		if !w.Boss && !prev.Boss && w.Count < prev.Count {
			t.Fatalf("wave %d is %d ships and wave %d was %d", n, w.Count, n-1, prev.Count)
		}
		prev = w
	}
}

// Two axes are capped and one is not, and that asymmetry is the design. Speed
// and pace are capped because faster stops being harder and starts being
// unreadable; the toughness is not, and that is the wall a run ends against.
func TestTwoAxesAreCappedAndTheOneThatEndsRunsIsNot(t *testing.T) {
	f := aField(t, 80, 24)
	deep := WaveFor(MaxWave, f)
	if deep.Haste > hasteCap {
		t.Errorf("at wave %d the fleet moves at %g times its speed", MaxWave, deep.Haste)
	}
	if deep.Every < releaseFloor {
		t.Errorf("at wave %d a ship arrives every %d ticks", MaxWave, deep.Every)
	}
	if deep.Count > countCap {
		t.Errorf("at wave %d there are %d ships to allocate", MaxWave, deep.Count)
	}
	if deep.Tough <= WaveFor(100, f).Tough {
		t.Error("the toughness stopped climbing, so nothing ends a run")
	}
}

// The same wave twice is the same wave: the recipe is pure in the number and the
// field, so a run resumed off disk is the wave it left.
func TestAWaveIsTheSameRecipeEveryTimeYouReachIt(t *testing.T) {
	f := aField(t, 80, 24)
	for _, n := range []int{1, 3, 5, 40, 41} {
		if a, b := WaveFor(n, f), WaveFor(n, f); a != b {
			t.Errorf("wave %d came out twice: %+v and %+v", n, a, b)
		}
	}
}

// The fleet is unlocked stage by stage, and a wave never draws from a ship it has
// not reached.
func TestNoWaveDrawsFromAShipItHasNotUnlocked(t *testing.T) {
	f := aField(t, 80, 24)
	was := 0
	for n := 1; n <= 200; n++ {
		w := WaveFor(n, f)
		pool := w.Unlocked()
		if len(pool) < was {
			t.Fatalf("wave %d may draw from %d ships and wave %d could use %d",
				n, len(pool), n-1, was)
		}
		was = len(pool)
		for _, of := range pool {
			if Fleet[of].Stage > w.Stage {
				t.Fatalf("wave %d is stage %d and may spawn %s, which is stage %d",
					n, w.Stage, Fleet[of].Name, Fleet[of].Stage)
			}
		}
	}
	if was != len(Fleet) {
		t.Errorf("after two hundred waves only %d of the %d ships are out", was, len(Fleet))
	}
}
