package invaders

// The field and the ladder.
//
// Vertical: the creature's ship sits on the floor and moves left and right, and
// the fleet comes down from the top. What comes down is no longer one block
// that steps in unison - it is ships, arriving in ones and twos, each falling
// and drifting and shooting on its own clock.

const (
	// MinCols and MinRows are the smallest terminal worth drawing in.
	//
	// The floor is what the biggest enemy needs to be drawable with room to
	// dodge it, plus the ship, plus the two rows the HUD and the help take.
	MinCols = 60
	MinRows = 18

	HUDRows  = 1
	HelpRows = 1

	// The ship: three rows, five cells. See ship.go for why it is a
	// representation of the creature and not the creature itself.
	ShipRows = 3
	ShipCols = 5

	// A rank sprite off the canvas: every boss.
	BossCols = 9
	BossRows = 5

	// TicksPerSecond is the frame rate, and forty rather than twenty.
	//
	// Everything below and in game.go is expressed in ticks, so doubling this
	// meant halving every distance-per-tick and doubling every count-of-ticks in
	// the package, kept in one commit on purpose. It is worth it: at twenty a
	// frame is fifty milliseconds, and a ship crossing one column per frame was
	// as smooth as the grid can be while everything else - a fleet falling at a
	// row a second, a bomb at two - moved in visible lurches. Forty halves the
	// step everything takes without changing how fast anything travels.
	TicksPerSecond = 40

	// Stages is the deepest the colour ladder goes; the fleet is unlocked
	// against the same number.
	stagesDeep = 8

	// The three ceilings, named because a test asserts them and a number
	// repeated in a test is a number that drifts.
	//
	// The pace and the speed are capped because past a point faster is not
	// harder, it is unreadable; the count is capped because Spawn cannot ask for
	// memory without a limit off a wave number that came from a file. What is
	// NOT capped is the toughness, and that is the wall a run ends against.
	releaseFloor = 24
	countCap     = 30
	hasteCap     = 2.0
)

// Field is the geometry a terminal affords: the playfield itself, with the HUD
// row and the help row already taken off.
type Field struct{ Cols, Rows int }

// FieldFor is the field inside a terminal, and false when the terminal is too
// small to draw a fair game in.
func FieldFor(cols, rows int) (Field, bool) {
	if cols < MinCols || rows < MinRows {
		return Field{}, false
	}
	return Field{Cols: cols, Rows: rows - HUDRows - HelpRows}, true
}

// ShipRow is the floor: the top row of the ship when it is standing on it, and
// where every run starts.
func (f Field) ShipRow() int { return f.Rows - ShipRows }

// ShipRoof is as high as the ship may climb, and it is half the field.
//
// The reference gives the player the whole screen, and it can afford to: its
// enemies come at you from everywhere. Ours all spawn on row zero and come down,
// so a ship that could reach the top would sit on the spawn line and shoot each
// one before it had drawn a frame - which is not a harder game or an easier one,
// it is a different game with no descent in it.
//
// Half is enough to be worth having: nine rows of it in an eighty-by-twenty-four
// terminal, which is room to climb over a bomb, meet something before it reaches
// the floor, or back off from a boss.
func (f Field) ShipRoof() int { return f.Rows / 2 }

// ShipColMax is the rightmost column the ship's left edge may reach.
func (f Field) ShipColMax() int { return f.Cols - ShipCols }

// Wave is a wave's recipe. There is no last one: every n >= 1 has an answer.
type Wave struct {
	N     int
	Boss  bool
	Stage int // 1..8, the colour and the depth of the fleet it may draw from

	Count int     // ships released over the whole wave
	Every int     // ticks between one release and the next
	Pack  int     // ships per release: ones at first, twos and threes later
	Tough int     // hit points added to every ship in it
	Haste float64 // multiplier on fall and drift

	BossOf int // index into Bosses, when Boss
	BossHP int
}

// WaveFor is the recipe for a wave in a field.
//
// Three axes climb and two are capped, which is the same asymmetry the block
// had and for the same reasons. The COUNT and the toughness climb without a
// ceiling: that is the ladder, and the point where your kit stops clearing a
// wave before it lands on you is where the run ends - a wall nobody wrote down,
// which is what replaced a final boss. The pace of the releases and the speed
// of the ships are capped, because faster is not harder past a point, it is
// unreadable.
func WaveFor(n int, f Field) Wave {
	n = clamp(n, 1, MaxWave)
	stage := clamp(1+(n-1)/4, 1, stagesDeep)

	w := Wave{
		N:     n,
		Boss:  n%5 == 0,
		Stage: stage,
		Count: clamp(6+n, 6, countCap),
		Every: clamp(120-4*n, releaseFloor, 120),
		Pack:  clamp(1+n/12, 1, 3),
		Tough: (n - 1) / 6,
		Haste: 1 + float64(n)/50,
	}
	if w.Haste > hasteCap {
		w.Haste = hasteCap
	}
	if w.Boss {
		w.BossOf = bossFor(n, stage)
		// Measured, not guessed: at thirty plus twenty a checkpoint, a level-six
		// title killed the wave-fifteen boss in forty ticks - two seconds, and
		// the arrival banner was still on screen. The magazine is the reason.
		// The old in-flight cap of two presses held throughput down to about a
		// third of this; a fourteen-round magazine at seven ticks a shot does
		// not.
		w.BossHP = 45 + 40*(n/5)
		// A boss wave is the boss and a thin escort, not a boss on top of a
		// full wave: two things to read at once is a fight, five is noise.
		w.Count = clamp(n/5, 1, 6)
		w.Pack = 1
	}
	return w
}

// Unlocked is how many of the fleet a wave may draw from: everything up to its
// own stage. Wave one is drones and wasps, and by stage eight the whole zoo is
// out.
func (w Wave) Unlocked() []int {
	out := make([]int, 0, len(Fleet))
	for i, c := range Fleet {
		if c.Stage <= w.Stage {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		out = append(out, 0)
	}
	return out
}

// bossFor picks which of the thirty-five stands at the end of a wave.
//
// The rank is tied to the stage the wave has reached rather than to a count of
// bosses, and that is a fix rather than a detail: walking the roster in order
// spends nine bosses - forty-five waves - on the rank the canvas calls larvae,
// and almost nobody gets that far. Tying it to the stage means the four the
// canvas calls jefes turn up while a run is still going.
func bossFor(n, stage int) int {
	rank := clamp(1+(stage-1)*len(Ranks)/stagesDeep, 1, len(Ranks))

	first, count := 0, 0
	for i, b := range Bosses {
		if b.Rank != rank {
			continue
		}
		if count == 0 {
			first = i
		}
		count++
	}
	if count == 0 {
		return 0
	}
	return first + (n/5-1)%count
}

// BossFor is the rival that closes a wave: one of the thirty-five off the
// canvas, climbing the ranks as the waves go up.
func BossFor(w Wave, f Field) Boss { return Bosses[clamp(w.BossOf, 0, len(Bosses)-1)] }

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
