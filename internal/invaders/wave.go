package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// The field, the ladder and the things coming down it.

const (
	// MinCols and MinRows are the smallest terminal worth drawing in.
	//
	// The design said 60x14. It cannot: the ship is the creature, five rows of
	// it, and its hitbox is the three inside. At 14 rows the field is 12, the
	// ship can stand in 8 places and its body blocks 3 of them - 37% of the
	// playfield is where the player is standing, and there is nowhere to go when
	// three lanes fire at once. At 18 the field is 16, twelve lanes, 25%. Still
	// inside the universal 24x80 with room for a prompt.
	MinCols = 60
	MinRows = 18

	HUDRows  = 1
	HelpRows = 1

	// The ship is pet.Draw's five rows and pet.SpriteWidth's nine columns, and
	// only the middle three rows are solid. The crest and the feet are cosmetic
	// and things pass through them, which is what lets a creature-shaped ship be
	// a fair one.
	ShipRows  = 5
	ShipCols  = pet.SpriteWidth
	HitTop    = 1
	HitBottom = 3

	TicksPerSecond = 20

	// MaxSeconds caps how long one wave may last. Waves go from short to long as
	// they climb, and without a ceiling "long" eventually stops being a wave.
	MaxSeconds = 180

	// baseLanes is the lane count the curve's figures are written for. A taller
	// terminal gets proportionally more enemies so that the density - and with
	// it the difficulty - does not depend on the size of the window.
	baseLanes = 12

	// squadEvery is the gap between squads, in ticks.
	squadEvery = 40
)

// Field is the geometry a terminal affords: the playfield itself, with the HUD
// row and the help row already taken off.
type Field struct{ Cols, Rows int }

// FieldFor is the field inside a terminal, and false when the terminal is too
// small to draw a fair game in. Refusing is the right answer: a cramped shmup
// reads as a broken one.
func FieldFor(cols, rows int) (Field, bool) {
	if cols < MinCols || rows < MinRows {
		return Field{}, false
	}
	return Field{Cols: cols, Rows: rows - HUDRows - HelpRows}, true
}

// Lanes are the field rows an enemy may occupy: the ones the ship's face can
// reach, which is rows 2 to Rows-3 because the face sits two rows down a
// five-row sprite.
//
// Anything outside them is a leak the player could not have stopped. Without
// this rule an enemy on row 0 is unkillable and costs a life every single wave,
// which is a bleed no amount of skill touches.
func (f Field) Lanes() (first, last int) { return HitTop + 1, f.Rows - ShipRows + HitBottom }

// LaneCount is how many rows an enemy can be in.
func (f Field) LaneCount() int {
	first, last := f.Lanes()
	return last - first + 1
}

// ShipRowMax is the lowest row the ship's sprite may start on.
func (f Field) ShipRowMax() int { return f.Rows - ShipRows }

// BossBounds is the x range a boss may move in. It is the right third and it
// never crosses out of it: a boss does not leak past you, it duels you.
func (f Field) BossBounds() (min, max float64) {
	return float64(2 * f.Cols / 3), float64(f.Cols - ShipCols)
}

// Body is a silhouette: a glyph, a width, and what it does to the wave's HP and
// speed. Trait is one behaviour on top of it. Seven bodies by five traits is the
// bestiary - thirty-five kinds composed the way a form is a trade by a mark,
// rather than thirty-five unrelated things written out by hand and impossible to
// balance or to test.
type Body uint8

const (
	Mote Body = iota
	Dart
	Shard
	Spore
	Husk
	Slab
	Crawler
	bodyCount
)

// Trait is one behaviour. Plain is half the bestiary on purpose: without
// something boring to compare against, nothing else reads as interesting.
type Trait uint8

const (
	Plain Trait = iota
	Weaver
	Darter
	Plated
	Splitter
	traitCount
)

// bodies is the table, indexed by Body. Glyphs come from the same block-drawing
// range as the sprites so that theme.RuneWidth counts each one as a single cell.
var bodies = [bodyCount]struct {
	ID    string
	Glyph string
	W     int
	HP    int     // multiplier on the wave's HP
	Speed float64 // multiplier on the wave's speed
}{
	Mote:    {"mote", "▪", 1, 1, 1.0},
	Dart:    {"dart", "»", 1, 1, 1.6},
	Shard:   {"shard", "◆", 1, 2, 1.3},
	Spore:   {"spore", "∘", 1, 1, 1.2},
	Husk:    {"husk", "▚▚", 2, 2, 1.0},
	Slab:    {"slab", "▰▰", 2, 3, 0.7},
	Crawler: {"crawler", "▬▬▬", 3, 4, 0.5},
}

// traitIDs are the keys into i18n.G().Traits, indexed by Trait.
var traitIDs = [traitCount]string{
	Plain: "plain", Weaver: "weaver", Darter: "darter",
	Plated: "plated", Splitter: "splitter",
}

// Enemy is one thing coming at you. A boss is an Enemy with Rival set: the form
// it is wearing, which decides its sprite, its ramp and its gun.
type Enemy struct {
	Body  Body
	Trait Trait

	X     float64
	Row   int
	HP    int
	Speed float64

	// Phase is the weaver's clock, and Drift which way it is going.
	Phase int
	Drift int

	// Split marks an enemy that came out of a splitter. A splitter's children
	// never split again, or one lucky wave becomes an allocation with no bound.
	Split bool

	// The boss half.
	Rival string // "" for everything that is not a boss
	Kit   Kit    // the rival's own kit: it fights with the gun that form flies
	MaxHP int
	Fire  int // ticks to its next shot
}

// Boss says whether this is a rival rather than one of the swarm.
func (e Enemy) Boss() bool { return e.Rival != "" }

// W is how many cells wide it is.
func (e Enemy) W() int {
	if e.Boss() {
		return ShipCols
	}
	return bodies[e.Body].W
}

// Rows is how many rows tall it is: one for the swarm, five for a rival, because
// a rival is drawn with pet.Draw like everything else in this theme.
func (e Enemy) Rows() int {
	if e.Boss() {
		return ShipRows
	}
	return 1
}

// Solid is the row range that can actually be hit. For a rival it is the middle
// three of its five, the same rule as the ship's: symmetric, and one sentence to
// explain.
func (e Enemy) Solid() (top, bottom int) {
	if e.Boss() {
		return e.Row + HitTop, e.Row + HitBottom
	}
	return e.Row, e.Row
}

// Vital is a rival's state for pet.Draw, so it wears down through the same seven
// states the ship does: the head goes down, the colour sinks, and at zero it
// lies down. The phases of the fight are those states, which is why they need no
// machinery of their own.
func (e Enemy) Vital() pet.Vital {
	if e.MaxHP <= 0 {
		return pet.Vitals[0]
	}
	hurt := float64(e.MaxHP-e.HP) / float64(e.MaxHP)
	return pet.StateFor(100 * hurt)
}

// Glyph is what a swarm enemy is drawn with.
func (e Enemy) Glyph() string { return bodies[e.Body].Glyph }

// TraitID is the key into i18n.G().Traits.
func (e Enemy) TraitID() string { return traitIDs[e.Trait] }

// Wave is a wave's recipe: how long it lasts, how many come, how tough and how
// fast, and whether a rival is standing at the end of it.
type Wave struct {
	N       int
	Boss    bool
	Seconds int // the target length, from which Count follows
	Count   int
	HP      int
	Speed   float64
	Squad   int
	Every   int
	BossHP  int
}

// WaveFor is the recipe for a wave in a field. Defined for every n >= 1: there
// is no last wave, so there is no n it refuses.
//
// The duration is the primary dial and the enemy count follows from it. The
// design derived it the other way round and the result was almost flat - 20
// seconds at wave 1 and 52 at wave 99 - which is not "short at the start and long
// later", it is one length with a rounding error.
//
// Two axes are capped and two are not, and that asymmetry is the whole design.
// Capped: the duration, the speed and how many arrive at once. A half-hour wave
// is not a wave; faster than three quarters of a cell per tick is not harder,
// it is unreactable; and a squad wider than the field is an allocation with no
// bound. Uncapped: the HP of the swarm and of the rivals. That is where the
// ladder climbs, and since a kit tops out at level 6 there comes a wave your
// damage cannot clear - which is how a run ends, and is a better ending than a
// number somebody typed.
func WaveFor(n int, f Field) Wave {
	if n < 1 {
		n = 1
	}
	if n > MaxWave {
		n = MaxWave
	}
	lanes := f.LaneCount()

	seconds := 15 + 3*(n-1)
	if seconds > MaxSeconds {
		seconds = MaxSeconds
	}

	// One squad places at most one enemy per lane, which is what keeps every
	// allocation in here bounded by the size of the terminal.
	squad := (3 + n/20) * lanes / baseLanes
	if squad < 1 {
		squad = 1
	}
	if squad > lanes {
		squad = lanes
	}

	speed := 0.18 + 0.006*float64(n-1)
	if speed > 0.75 {
		speed = 0.75
	}

	return Wave{
		N:       n,
		Boss:    n%5 == 0,
		Seconds: seconds,
		Count:   squad * seconds / 2,
		HP:      1 + (n-1)/8,
		Speed:   speed,
		Squad:   squad,
		Every:   squadEvery,
		BossHP:  40 + 25*(n/5),
	}
}

// Ticks is how long the wave is meant to take.
func (w Wave) Ticks() int { return w.Seconds * TicksPerSecond }

// Unlocked is how many bodies and traits a wave may draw from: one more body
// every three waves, one more trait every eight. So wave 1 is a mote going in a
// straight line, and by wave 33 the whole zoo is out. The variety arrives
// progressively, the same way the length does.
func Unlocked(n int) (bodyKinds, traitKinds int) {
	bodyKinds = 1 + (n-1)/3
	if bodyKinds > int(bodyCount) {
		bodyKinds = int(bodyCount)
	}
	if bodyKinds < 1 {
		bodyKinds = 1
	}
	traitKinds = 1 + (n-1)/8
	if traitKinds > int(traitCount) {
		traitKinds = int(traitCount)
	}
	if traitKinds < 1 {
		traitKinds = 1
	}
	return bodyKinds, traitKinds
}

// Spawn is the next squad, placed in lanes the ship can reach, one enemy per
// lane at most.
func (w Wave) Spawn(f Field, rand *uint64) []Enemy {
	first, last := f.Lanes()
	lanes := make([]int, 0, f.LaneCount())
	for row := first; row <= last; row++ {
		lanes = append(lanes, row)
	}
	// Shuffled rather than picked-and-retried, so a squad is a set of distinct
	// lanes in one pass and the number of rolls does not depend on luck.
	for i := len(lanes) - 1; i > 0; i-- {
		j := roll(rand, i+1)
		lanes[i], lanes[j] = lanes[j], lanes[i]
	}
	if len(lanes) > w.Squad {
		lanes = lanes[:w.Squad]
	}

	bodyKinds, traitKinds := Unlocked(w.N)
	out := make([]Enemy, 0, len(lanes))
	for _, row := range lanes {
		body := Body(roll(rand, bodyKinds))
		trait := Trait(roll(rand, traitKinds))
		out = append(out, newEnemy(w, f, body, trait, row, rand))
	}
	return out
}

// newEnemy builds one of the swarm at the right edge.
func newEnemy(w Wave, f Field, body Body, trait Trait, row int, rand *uint64) Enemy {
	spec := bodies[body]
	drift := 1
	if chance(rand, 50) {
		drift = -1
	}
	return Enemy{
		Body:  body,
		Trait: trait,
		X:     float64(f.Cols - spec.W),
		Row:   row,
		HP:    w.HP * spec.HP,
		Speed: w.Speed * spec.Speed,
		Phase: roll(rand, 16),
		Drift: drift,
	}
}

// SplitInto is the two motes a splitter leaves behind, in the lanes on either
// side of it. They are marked Split so they cannot split again.
func (e Enemy) SplitInto(f Field) []Enemy {
	if e.Trait != Splitter || e.Split || e.Boss() {
		return nil
	}
	first, last := f.Lanes()
	out := make([]Enemy, 0, 2)
	for _, row := range [2]int{e.Row - 1, e.Row + 1} {
		if row < first || row > last {
			continue
		}
		out = append(out, Enemy{
			Body: Mote, Trait: Plain,
			X: e.X, Row: row,
			HP: 1, Speed: e.Speed,
			Split: true,
		})
	}
	return out
}

// BossFor is the rival standing at the end of a wave: one of the forms you are
// not, wearing its own sprite and firing its own kit.
func (w Wave) BossFor(f Field, mine string, level int) Enemy {
	rival := RivalFor(w.N, mine)
	_, right := f.BossBounds()
	kit := KitFor(rival, level)
	return Enemy{
		Rival: rival,
		Kit:   kit,
		X:     right,
		Row:   f.ShipRowMax() / 2,
		HP:    w.BossHP,
		MaxHP: w.BossHP,
		Speed: 0.08,
		Drift: 1,
		Fire:  kit.Cadence,
	}
}

// rivals is the order bosses arrive in: the seven trades first, then the
// fourteen marks, then the fourteen titles, and the two secrets last, which
// makes them the rare ones. Thirty-seven rivals, so the roster repeats every
// hundred and eighty-five waves - and a rival that comes back comes back with
// the HP of the wave it is standing on.
//
// Built by walking pet.Tree in slice order rather than ranging over a map, so
// the order is the same in every process. Rung-2 forms are left out: a pattern
// is not a boss.
var rivals = func() []string {
	trades := []string{}
	for _, second := range pet.Tree[pet.Root] {
		trades = append(trades, pet.Tree[second]...)
	}
	marks := []string{}
	for _, trade := range trades {
		marks = append(marks, pet.Tree[trade]...)
	}
	out := append([]string{}, trades...)
	out = append(out, marks...)
	for _, mark := range marks {
		if title, ok := pet.Titles[mark]; ok {
			out = append(out, title)
		}
	}
	return append(out, pet.Secrets[:]...)
}()

// RivalFor is the form a wave's boss wears. Deterministic in the wave number,
// climbing the tiers as the waves go up, and never the form you are flying: you
// do not fight yourself.
func RivalFor(n int, mine string) string {
	if len(rivals) == 0 {
		return pet.Root
	}
	i := (n/5 - 1) % len(rivals)
	if i < 0 {
		i = 0
	}
	if rivals[i] == mine {
		i = (i + 1) % len(rivals)
	}
	return rivals[i]
}
