package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// The field, the ladder and the formation that comes down it.
//
// Vertical, like the game it is named after: the creature sits at the bottom and
// moves left and right, and the swarm comes down from the top in a block that
// walks sideways and steps down when it reaches a wall.

const (
	// MinCols and MinRows are the smallest terminal worth drawing in.
	//
	// The width has to hold a formation of at least four columns; the height a
	// formation of at least two rows, the creature, and enough between them for
	// the descent to mean anything.
	MinCols = 60
	MinRows = 18

	HUDRows  = 1
	HelpRows = 1

	// The creature at the bottom is pet.DrawTiny: two rows of five cells, its
	// mark over its eyes.
	//
	// The five-row card is the pet's portrait and this is its cannon. Cropping
	// the compact form was tried and is not the same thing as drawing a smaller
	// one: at nine cells it was still nearly twice the gap between two invaders,
	// where the arcade's cannon is about one of them wide. DrawTiny is a fourth
	// size in internal/pet rather than a crop out here, because that package is
	// the only thing that knows how to draw the creature.
	ShipRows = pet.TinyRows
	ShipCols = pet.TinyWidth

	// A troop and the cell it lives in: one glyph, in a cell five across and two
	// down.
	//
	// One cell each is what makes the arcade's own formation fit - eleven
	// columns by five rows, fifty-five on the screen - which no drawn sprite
	// could do in eighty columns. The spacing is what gives the block its shape:
	// at five apart, eleven columns fill about two thirds of a wide terminal,
	// which is roughly what the cabinet showed.
	TroopCols = 1
	TroopRows = 1
	cellCols  = 5
	cellRows  = 2

	// TroopHit is how wide one of them is to a bullet: its own cell and one
	// either side.
	//
	// A one-cell target hit by a one-cell bullet is not the arcade's game, it is
	// a coin toss - the cabinet's aliens are eight to twelve pixels across and
	// its bullet is one. Three cells of a five-cell pitch leaves two columns of
	// clear air between neighbours, so aiming still means something and missing
	// is still your fault.
	TroopHit = 3

	// A rank sprite: the bigger ones, and every boss.
	BossCols = 9
	BossRows = 5

	TicksPerSecond = 20

	// MaxSeconds caps how long one wave may last.
	MaxSeconds = 180
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

// ShipRow is the top row of the creature: it stands on the floor of the field.
func (f Field) ShipRow() int { return f.Rows - ShipRows }

// ShipColMax is the leftmost column the creature may not pass.
func (f Field) ShipColMax() int { return f.Cols - ShipCols }

// FormationCols and FormationRows are how big a block this terminal can hold.
//
// Both are derived rather than fixed so that the game is the same shape in a
// narrow window as in a wide one: eleven columns is the arcade's number and the
// most this will ever draw, four is the fewest that still reads as a formation.
func (f Field) FormationCols() int {
	return clamp((f.Cols-2)/cellCols, 4, 11)
}

func (f Field) FormationRows() int {
	// The floor, the creature, and eight rows of daylight between the block and
	// the creature when it starts - the descent has to be worth watching. Two
	// rows is the fewest worth calling a formation, five is the arcade's.
	return clamp((f.Rows-ShipRows-8)/cellRows, 2, 5)
}

// BlockCols is how wide the whole formation is, in cells.
func (f Field) BlockCols() int { return f.FormationCols()*cellCols - 1 }

// Wave is a wave's recipe.
type Wave struct {
	N       int
	Boss    bool
	Stage   int   // the deepest stage this wave draws from, 1..8
	Species []int // one troop index per formation row, top row first
	HP      int   // hit points per troop member
	Step    int   // ticks between one sideways step of the block and the next
	Drop    int   // ticks between one bomb and the next
	BossOf  int   // index into Bosses, when Boss
	BossHP  int
}

// WaveFor is the recipe for a wave in a field. Defined for every n >= 1: there
// is no last wave, so there is no n it refuses.
func WaveFor(n int, f Field) Wave {
	n = clamp(n, 1, MaxWave)
	stage := clamp(1+(n-1)/4, 1, len(Stages))

	w := Wave{
		N:     n,
		Boss:  n%5 == 0,
		Stage: stage,
		HP:    1 + (n-1)/10,
		// The block starts slow and quickens with the wave, and quickens again
		// as it is emptied - see Formation.Step. Never under two ticks, or a
		// row crosses the screen before a frame has been drawn.
		Step: clamp(26-n/2, 5, 40),
		// Fifty-five of them drop a lot of bombs between them, and a wave with a
		// one-shot gun is long. Seven seconds between bombs at the start,
		// closing to one: the early waves have to be survivable by a larva.
		Drop: clamp(140-2*n, 20, 140),
	}
	w.Species = speciesFor(f.FormationRows())
	if w.Boss {
		w.BossOf = bossFor(n, stage)
		w.BossHP = 30 + 20*(n/5)
	}
	return w
}

// bossFor picks which of the thirty-five stands at the end of a wave.
//
// The rank is tied to the stage the troops have reached rather than to a count
// of bosses, and that is a fix rather than a detail: walking the roster in order
// spends nine bosses - forty-five waves - on the rank the canvas calls larvae,
// and almost nobody gets that far. Tying it to the stage means the four the
// canvas calls jefes turn up while a run is still going.
func bossFor(n, stage int) int {
	rank := clamp(1+(stage-1)*len(Ranks)/len(Stages), 1, len(Ranks))

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

// speciesFor stacks the three the way the cabinet stacks them: one row of squid
// on top, two of crab, octopuses the rest of the way down.
//
// It does not vary with the wave, and that is the point of using the arcade's
// bestiary rather than forty of our own - what a later wave changes is the
// colour and what they are worth, not the shape. See Wave.Stage.
func speciesFor(rows int) []int {
	out := make([]int, rows)
	for r := 0; r < rows; r++ {
		switch {
		case r == 0:
			out[r] = 0 // squid
		case r <= 2:
			out[r] = 1 // crab
		default:
			out[r] = 2 // octopus
		}
	}
	return out
}

// Member is one of the block: where it sits in the grid and what is left of it.
type Member struct {
	Species  int
	Col, Row int
	HP       int
}

// Formation is the block, which moves as one thing.
type Formation struct {
	X, Y  float64 // the top-left cell of the grid
	Dir   int     // 1 right, -1 left
	Wait  int     // ticks to the next sideways step
	Steps int     // steps taken, for the two-frame leg animation

	Members []Member
}

// NewFormation lines a wave up at the top of the field.
func NewFormation(w Wave, f Field) Formation {
	cols := f.FormationCols()
	members := make([]Member, 0, cols*len(w.Species))
	for row, species := range w.Species {
		for col := 0; col < cols; col++ {
			members = append(members, Member{Species: species, Col: col, Row: row, HP: w.HP})
		}
	}
	return Formation{
		X:       float64((f.Cols - f.BlockCols()) / 2),
		Y:       0,
		Dir:     1,
		Wait:    w.Step,
		Members: members,
	}
}

// At is where a member is drawn, in cells.
func (fm Formation) At(m Member) (x, y int) {
	return int(fm.X) + m.Col*cellCols, int(fm.Y) + m.Row*cellRows
}

// Edges are the leftmost and rightmost cells the block currently occupies.
//
// Taken from the members that are still alive rather than from the grid, so
// clearing a flank lets the rest of the block use the whole screen - which is
// the arcade's behaviour and the reason shooting the edges first is a tactic.
func (fm Formation) Edges() (left, right int) {
	first := true
	for _, m := range fm.Members {
		x, _ := fm.At(m)
		if first || x < left {
			left = x
		}
		if first || x+TroopCols > right {
			right = x + TroopCols
		}
		first = false
	}
	return left, right
}

// Bottom is the lowest row any member reaches.
func (fm Formation) Bottom() int {
	low := 0
	for _, m := range fm.Members {
		if _, y := fm.At(m); y+TroopRows > low {
			low = y + TroopRows
		}
	}
	return low
}

// Frame is which of the two leg frames the block is showing.
func (fm Formation) Frame() int { return fm.Steps % 2 }

// StepEvery is how long between steps: the wave's own pace, quickened as the
// block empties. The last few always come down fast, which is the arcade's most
// famous accident and worth keeping on purpose.
func StepEvery(w Wave, left, total int) int {
	if total <= 0 {
		return w.Step
	}
	share := float64(left) / float64(total)
	every := int(float64(w.Step) * (0.25 + 0.75*share))
	return clamp(every, 2, w.Step)
}

// BossFor is the rival that closes a wave: one of the thirty-five off the
// canvas, climbing the ranks as the waves go up, so the first ones you meet are
// larvae and the ones deep in a run are the four the canvas calls jefes.
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
