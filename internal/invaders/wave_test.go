package invaders

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
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

// The design's 60x14 does not survive the ship being the creature. Five rows of
// sprite with three of them solid leaves the player standing in 37% of a
// twelve-row field, with nowhere to go when three lanes fire at once. This is
// the arithmetic, asserted, so nobody lowers the floor back without redoing it.
func TestTheFieldHasRoomToDodgeAFiveRowShip(t *testing.T) {
	f := aField(t, MinCols, MinRows)
	if f.Rows != MinRows-HUDRows-HelpRows {
		t.Fatalf("the field is %d rows of a %d row terminal", f.Rows, MinRows)
	}
	positions := f.ShipRowMax() + 1
	blocked := HitBottom - HitTop + 1
	if share := float64(blocked) / float64(f.LaneCount()); share > 0.30 {
		t.Errorf("the ship's body blocks %.0f%% of the lanes; 14 rows was 37%% and that is why it went",
			share*100)
	}
	if positions < 12 {
		t.Errorf("the ship has %d places to stand, want at least 12", positions)
	}
}

// Below the floor it refuses rather than drawing a mess. A cramped shmup reads
// as a broken one, not as a small one.
func TestAFieldSmallerThanTheFloorIsRefusedAndNotDrawn(t *testing.T) {
	for _, c := range [][2]int{{59, 24}, {80, 17}, {0, 0}, {MinCols - 1, MinRows - 1}, {-1, -1}} {
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

// Every lane an enemy can be in has to be a lane the ship's face can reach. An
// enemy one row above the topmost face position is unkillable, and unkillable
// means one life gone every wave for ever - a bleed no amount of skill touches.
func TestNoEnemyEverSpawnsInARowTheShipCannotReach(t *testing.T) {
	for rows := MinRows; rows <= 60; rows++ {
		f := aField(t, 80, rows)
		first, last := f.Lanes()

		// The face sits two rows into the sprite, so these are the rows it can
		// be on, and the lanes must not exceed them.
		faceTop, faceBottom := HitTop+1, f.ShipRowMax()+HitBottom
		if first < faceTop || last > faceBottom {
			t.Errorf("%d rows: lanes %d..%d, the face reaches %d..%d",
				rows, first, last, faceTop, faceBottom)
		}

		seed := uint64(rows)
		for _, n := range []int{1, 5, 17, 60, 200, MaxWave} {
			w := WaveFor(n, f)
			for i := 0; i < 20; i++ {
				for _, e := range w.Spawn(f, &seed) {
					if e.Row < first || e.Row > last {
						t.Fatalf("%d rows, wave %d: an enemy in row %d, lanes are %d..%d",
							rows, n, e.Row, first, last)
					}
				}
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
		if w.Count < 1 || w.Squad < 1 || w.HP < 1 || w.Seconds < 1 {
			t.Errorf("wave %d came out as %+v", n, w)
		}
		if w.Speed <= 0 || w.Speed > 0.75 {
			t.Errorf("wave %d moves at %v", n, w.Speed)
		}
	}
}

// A boss every fifth wave, for ever, and no wave that is the last one.
func TestABossStandsOnEveryFifthWaveForever(t *testing.T) {
	f := aField(t, 80, 24)
	for n := 1; n <= 2000; n++ {
		if got, want := WaveFor(n, f).Boss, n%5 == 0; got != want {
			t.Fatalf("wave %d: boss = %v, want %v", n, got, want)
		}
	}
	// Ninety-nine used to be the end of the game. It is a boss wave like any
	// other now, and 100 exists.
	if WaveFor(100, f).Count < WaveFor(99, f).Count {
		t.Error("the ladder stops climbing after 99")
	}
}

// The ladder only ever goes up. Two of its axes are capped and two are not, and
// this is the half that says nothing goes backwards.
func TestTheLadderNeverGetsEasier(t *testing.T) {
	f := aField(t, 80, 24)
	prev := WaveFor(1, f)
	for n := 2; n <= 2000; n++ {
		w := WaveFor(n, f)
		if w.HP < prev.HP || w.Speed < prev.Speed || w.Seconds < prev.Seconds || w.Squad < prev.Squad {
			t.Fatalf("wave %d is easier than %d:\n%+v\n%+v", n, n-1, w, prev)
		}
		prev = w
	}
}

// Short at the start, long later, and never endless. This is the test that
// defends the decision against a future "let's make it snappier".
func TestAWaveIsShortAtTheStartAndLongLaterAndNeverEndless(t *testing.T) {
	f := aField(t, 80, 24)
	if got := WaveFor(1, f).Seconds; got > 20 {
		t.Errorf("wave 1 lasts %ds, want a short one", got)
	}
	if got := WaveFor(60, f).Seconds; got < 150 {
		t.Errorf("wave 60 lasts %ds, want a long one", got)
	}
	for _, n := range []int{1, 56, 200, 2000, MaxWave} {
		if got := WaveFor(n, f).Seconds; got > MaxSeconds {
			t.Errorf("wave %d lasts %ds, past the %ds ceiling", n, got, MaxSeconds)
		}
	}
}

// The HP is the axis with no ceiling, and it is on purpose: a kit tops out at
// level 6, so there comes a wave your damage cannot clear. That is how a run
// ends now that no number ends it.
func TestTheEnemiesGetTougherWithoutEndBecauseThatIsWhatEndsARun(t *testing.T) {
	f := aField(t, 80, 24)
	if WaveFor(MaxWave, f).HP <= WaveFor(200, f).HP {
		t.Error("the swarm's HP has a ceiling, which means a run has no ending")
	}
	if WaveFor(MaxWave, f).BossHP <= WaveFor(200, f).BossHP {
		t.Error("a rival's HP has a ceiling")
	}
	// And the strongest kit there is falls behind it eventually, or the ladder
	// is decoration.
	best := 0
	for form := range pet.Sprites {
		if k := KitFor(form, len(pet.Levels)); k.Damage*k.Shots > best {
			best = k.Damage * k.Shots
		}
	}
	if WaveFor(MaxWave, f).HP <= best {
		t.Errorf("the toughest enemy has %d hp and the best kit lands %d a volley: nothing ever wins",
			WaveFor(MaxWave, f).HP, best)
	}
}

// Nothing in here may allocate more than the terminal can hold. A squad places
// at most one enemy per lane, which is what bounds it - a wave number off a
// corrupt save must not become a memory request.
func TestNoSquadEverAsksForMoreEnemiesThanThereAreLanes(t *testing.T) {
	for _, rows := range []int{MinRows, 24, 60} {
		f := aField(t, 80, rows)
		seed := uint64(7)
		for _, n := range []int{1, 40, 200, MaxWave} {
			w := WaveFor(n, f)
			if w.Squad > f.LaneCount() {
				t.Errorf("%d rows, wave %d: a squad of %d into %d lanes", rows, n, w.Squad, f.LaneCount())
			}
			got := w.Spawn(f, &seed)
			if len(got) > f.LaneCount() {
				t.Errorf("%d rows, wave %d: spawned %d into %d lanes", rows, n, len(got), f.LaneCount())
			}
			seen := map[int]bool{}
			for _, e := range got {
				if seen[e.Row] {
					t.Errorf("%d rows, wave %d: two enemies in row %d", rows, n, e.Row)
				}
				seen[e.Row] = true
			}
		}
	}
}

// A taller window must not be a harder game. The figures are written for twelve
// lanes and scale with the lane count, so the density stays put.
func TestATallerTerminalIsNotAHarderGame(t *testing.T) {
	short := aField(t, 80, MinRows)
	tall := aField(t, 80, 48)

	s, l := WaveFor(40, short), WaveFor(40, tall)
	density := func(w Wave, f Field) float64 { return float64(w.Squad) / float64(f.LaneCount()) }
	if a, b := density(s, short), density(l, tall); a-b > 0.1 || b-a > 0.1 {
		t.Errorf("a squad fills %.2f of the lanes in %d rows and %.2f in %d rows",
			a, short.Rows, b, tall.Rows)
	}
}

// Thirty-five kinds, and no two of them the same thing twice. The bestiary is
// composed rather than enumerated for exactly this reason: a table of thirty-five
// hand-written enemies has no way to prove this at all.
func TestTheBestiaryIsThirtyFiveThingsAndNoTwoAreAlike(t *testing.T) {
	if int(bodyCount)*int(traitCount) != 35 {
		t.Fatalf("%d bodies by %d traits is %d, want 35",
			bodyCount, traitCount, int(bodyCount)*int(traitCount))
	}
	type kind struct {
		Glyph string
		W, HP int
		Speed float64
		Trait Trait
	}
	seen := map[kind]string{}
	for b := Body(0); b < bodyCount; b++ {
		for tr := Trait(0); tr < traitCount; tr++ {
			spec := bodies[b]
			k := kind{spec.Glyph, spec.W, spec.HP, spec.Speed, tr}
			name := spec.ID + "/" + traitIDs[tr]
			if other, dup := seen[k]; dup {
				t.Errorf("%s is exactly %s", name, other)
			}
			seen[k] = name
		}
	}
	if len(seen) != 35 {
		t.Errorf("%d distinct kinds, want 35", len(seen))
	}
}

// Every body and every trait needs a name in both languages, or the HUD says
// nothing about what just killed you in one of them.
func TestEveryBodyAndTraitHasAName(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		names := i18n.G().Traits
		for tr := Trait(0); tr < traitCount; tr++ {
			if names[traitIDs[tr]] == "" {
				t.Errorf("%s: the trait %q has no name", lang, traitIDs[tr])
			}
		}
		for b := Body(0); b < bodyCount; b++ {
			if bodies[b].ID == "" || bodies[b].Glyph == "" {
				t.Errorf("body %d has no id or no glyph", b)
			}
		}
	}
	i18n.Use("")
}

// The zoo arrives progressively, the same way the length does: wave 1 is one
// body going straight, and by the thirties everything is out.
func TestTheZooComesOutProgressively(t *testing.T) {
	b, tr := Unlocked(1)
	if b != 1 || tr != 1 {
		t.Errorf("wave 1 draws from %d bodies and %d traits, want one of each", b, tr)
	}
	b, tr = Unlocked(40)
	if b != int(bodyCount) || tr != int(traitCount) {
		t.Errorf("wave 40 draws from %d bodies and %d traits, want all of them", b, tr)
	}
	prevB, prevT := Unlocked(1)
	for n := 2; n <= 500; n++ {
		b, tr := Unlocked(n)
		if b < prevB || tr < prevT {
			t.Fatalf("wave %d unlocks fewer than %d: %d/%d after %d/%d", n, n-1, b, tr, prevB, prevT)
		}
		if b > int(bodyCount) || tr > int(traitCount) {
			t.Fatalf("wave %d unlocks %d bodies and %d traits", n, b, tr)
		}
		prevB, prevT = b, tr
	}
}

// A wave may only use what it has unlocked, or the progression is a comment
// rather than a rule.
func TestNoWaveEverUsesABodyItHasNotUnlocked(t *testing.T) {
	f := aField(t, 80, 24)
	seed := uint64(3)
	for n := 1; n <= 200; n++ {
		w := WaveFor(n, f)
		maxBody, maxTrait := Unlocked(n)
		for i := 0; i < 10; i++ {
			for _, e := range w.Spawn(f, &seed) {
				if int(e.Body) >= maxBody {
					t.Fatalf("wave %d used body %d of %d unlocked", n, e.Body, maxBody)
				}
				if int(e.Trait) >= maxTrait {
					t.Fatalf("wave %d used trait %d of %d unlocked", n, e.Trait, maxTrait)
				}
			}
		}
	}
}

// A splitter's children never split. Without this one wave of good luck is an
// allocation with no bound, which is the one way the bestiary could take the
// process down.
func TestASplitterNeverSpawnsASplitter(t *testing.T) {
	f := aField(t, 80, 24)
	parent := Enemy{Body: Slab, Trait: Splitter, Row: 5, X: 40, Speed: 0.2}
	kids := parent.SplitInto(f)
	if len(kids) != 2 {
		t.Fatalf("a splitter left %d behind, want two", len(kids))
	}
	for _, kid := range kids {
		if kid.Trait == Splitter {
			t.Error("a splitter's child splits too")
		}
		if !kid.Split {
			t.Error("a child is not marked as one")
		}
		if got := kid.SplitInto(f); got != nil {
			t.Errorf("a marked child still splits into %d", len(got))
		}
	}
	// And nothing else splits at all.
	for tr := Trait(0); tr < traitCount; tr++ {
		if tr == Splitter {
			continue
		}
		if got := (Enemy{Trait: tr, Row: 5}).SplitInto(f); got != nil {
			t.Errorf("trait %q splits", traitIDs[tr])
		}
	}
}

// A splitter on the edge lane leaves one child, not one in a row nothing can
// reach. Same rule as spawning, applied to the one thing that creates enemies
// mid-wave.
func TestASplittersChildrenStayInTheLanes(t *testing.T) {
	f := aField(t, 80, 24)
	first, last := f.Lanes()
	for _, row := range []int{first, last} {
		for _, kid := range (Enemy{Body: Slab, Trait: Splitter, Row: row}).SplitInto(f) {
			if kid.Row < first || kid.Row > last {
				t.Errorf("a splitter in row %d left a child in row %d", row, kid.Row)
			}
		}
	}
}

// A boss is one of the forty-one forms and never the one you are flying. You do
// not fight yourself.
func TestABossWearsAFormAndNeverTheOneYouAreFlying(t *testing.T) {
	for mine := range pet.Sprites {
		for n := 5; n <= 1000; n += 5 {
			rival := RivalFor(n, mine)
			if rival == mine {
				t.Fatalf("flying a %s, wave %d sent a %s", mine, n, rival)
			}
			if _, ok := pet.Sprites[rival]; !ok {
				t.Fatalf("wave %d sent %q, which is not a form", n, rival)
			}
		}
	}
}

// The rivals climb the tree as the waves climb: trades first, then marks, then
// titles, and the two secrets last of all.
//
// The secrets sit outside that climb rather than at the top of it, and they are
// last on purpose. pet.Tier calls them 5, the same rung as a mark, because they
// are not on the tree at all - so this asserts the climb over the on-tree
// rivals and then asserts the secrets are the two that close the roster, which
// is what makes them the rare ones. The first draft demanded one monotonic
// sequence over all thirty-seven and failed on the phoenix, correctly.
func TestTheBossTiersClimbWithTheWaves(t *testing.T) {
	onTree := rivals[:len(rivals)-len(pet.Secrets)]
	prev := 0
	for i, r := range onTree {
		tier := pet.Tier(r)
		if tier < prev {
			t.Errorf("boss %d, the %s, is tier %d after tier %d", i+1, r, tier, prev)
		}
		prev = tier
	}
	for i, secret := range pet.Secrets {
		if got := rivals[len(onTree)+i]; got != secret {
			t.Errorf("the roster closes with %s where %s should be", got, secret)
		}
	}
	if got := pet.Tier(RivalFor(5, "")); got != 3 {
		t.Errorf("the first boss is tier %d, want a trade", got)
	}
	// The roster is every form worth fighting, once: the rung-2 forms are not.
	if len(rivals) != 7+14+14+len(pet.Secrets) {
		t.Errorf("the roster is %d rivals, want the trades, marks, titles and secrets", len(rivals))
	}
	for _, r := range rivals {
		if pet.Tier(r) == 2 {
			t.Errorf("%s is a rung-2 form and not a boss", r)
		}
	}
	// And it is the same order in every process: built off Tree's slice order,
	// not off ranging a map.
	for i := 0; i < 50; i++ {
		if RivalFor(5, "") != rivals[0] {
			t.Fatal("the roster is not stable between calls")
		}
	}
}

// A rival fights with the gun its own form flies, which is where the variety
// comes from: forty-one bosses with their own behaviour and no new code for any
// of them.
func TestARivalFightsWithItsOwnKit(t *testing.T) {
	f := aField(t, 80, 24)
	for _, n := range []int{5, 25, 105, 200} {
		w := WaveFor(n, f)
		boss := w.BossFor(f, "spark", 4)
		if !boss.Boss() {
			t.Fatalf("wave %d's boss is not a boss: %+v", n, boss)
		}
		if want := KitFor(boss.Rival, 4); boss.Kit != want {
			t.Errorf("the %s fights with %+v, want its own %+v", boss.Rival, boss.Kit, want)
		}
		if boss.HP != w.BossHP || boss.MaxHP != w.BossHP {
			t.Errorf("wave %d's boss has %d/%d hp, want %d", n, boss.HP, boss.MaxHP, w.BossHP)
		}
	}
}

// The phases of a boss fight are the pet's own seven states, which is why they
// need no machinery: the rival's head goes down, its colour sinks, and at zero it
// lies down, all through pet.StateFor.
func TestABossWearsDownThroughTheSevenStates(t *testing.T) {
	f := aField(t, 80, 24)
	boss := WaveFor(25, f).BossFor(f, "spark", 5)

	ranks := map[int]bool{}
	for hp := boss.MaxHP; hp >= 0; hp-- {
		boss.HP = hp
		ranks[boss.Vital().Rank] = true
	}
	if len(ranks) != len(pet.Vitals) {
		t.Errorf("a whole fight passes through %d of the %d states", len(ranks), len(pet.Vitals))
	}
	boss.HP = boss.MaxHP
	if got := boss.Vital().Rank; got != 0 {
		t.Errorf("an untouched rival is in state %d, want the freshest", got)
	}
	boss.HP = 0
	if got := boss.Vital(); got.Rank != pet.KO.Rank {
		t.Errorf("a dead rival is in state %d, want k.o.", got.Rank)
	}
}

// A boss lives in the right third and never crosses out of it. It does not leak
// past you: it duels you, which is what makes every fifth wave a different shape
// of problem rather than a bigger pile of the same one.
func TestABossStartsAndStaysInTheRightThird(t *testing.T) {
	for _, cols := range []int{MinCols, 80, 116, 200} {
		f := aField(t, cols, 24)
		left, right := f.BossBounds()
		if left < float64(cols)/2 {
			t.Errorf("%d cols: the boss may reach x=%v, which is past halfway", cols, left)
		}
		if right+ShipCols > float64(cols) {
			t.Errorf("%d cols: the boss may reach x=%v and it is %d wide", cols, right, ShipCols)
		}
		boss := WaveFor(5, f).BossFor(f, "spark", 3)
		if boss.X < left || boss.X > right {
			t.Errorf("%d cols: the boss starts at %v, bounds are %v..%v", cols, boss.X, left, right)
		}
	}
}

// A rival is five rows tall and only its middle three can be hit, the same rule
// as the ship's. Symmetric, and one sentence to explain to a player.
func TestTheHitboxIsThreeRowsInsideTheFiveTheShipDraws(t *testing.T) {
	f := aField(t, 80, 24)
	boss := WaveFor(5, f).BossFor(f, "spark", 3)
	if boss.Rows() != ShipRows {
		t.Errorf("a rival is %d rows tall, want %d", boss.Rows(), ShipRows)
	}
	top, bottom := boss.Solid()
	if top != boss.Row+HitTop || bottom != boss.Row+HitBottom {
		t.Errorf("a rival is solid on %d..%d of %d..%d", top, bottom, boss.Row, boss.Row+ShipRows-1)
	}
	if bottom-top+1 != 3 {
		t.Errorf("a rival is solid on %d rows, want 3", bottom-top+1)
	}

	swarm := Enemy{Body: Mote, Row: 4}
	if swarm.Rows() != 1 {
		t.Errorf("one of the swarm is %d rows tall", swarm.Rows())
	}
	if top, bottom := swarm.Solid(); top != 4 || bottom != 4 {
		t.Errorf("one of the swarm is solid on %d..%d", top, bottom)
	}
}

// The same seed plays the same wave. Everything the tick is tested on rests on
// this, and it is here rather than in game_test.go because Spawn is the one
// place in the recipe that rolls dice.
func TestTheSameSeedSpawnsTheSameSquad(t *testing.T) {
	f := aField(t, 80, 24)
	w := WaveFor(37, f)
	a, b := uint64(0xDEFACED), uint64(0xDEFACED)
	for i := 0; i < 50; i++ {
		x, y := w.Spawn(f, &a), w.Spawn(f, &b)
		if len(x) != len(y) {
			t.Fatalf("squad %d: %d enemies against %d", i, len(x), len(y))
		}
		for j := range x {
			if x[j] != y[j] {
				t.Fatalf("squad %d, enemy %d:\n%+v\n%+v", i, j, x[j], y[j])
			}
		}
	}
	if c := uint64(1); w.Spawn(f, &c)[0] == w.Spawn(f, &a)[0] {
		t.Error("two different seeds spawned the same thing, which means the seed does nothing")
	}
}
