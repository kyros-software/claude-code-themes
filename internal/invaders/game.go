package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// The tick. No terminal, no files, no clock: this file takes a state and a key
// and returns the next state, which is what makes the waves, the kits and the
// life rules testable without a tty.
//
// It must never write pet.json either. A run does cost the creature a level, but
// that happens once, in run.go, when the run is over. A tick that could reach the
// pet would punish it twenty times a second.

// Key is one decoded keypress.
type Key uint8

const (
	None Key = iota
	Left
	Right
	Fire
	Ability
	Pause
	Quit
)

// Phase is what the run is doing.
type Phase uint8

const (
	Playing Phase = iota
	Cleared
	Paused
	Over
)

const (
	// moveWait is ticks between one column of movement and the next. Terminal
	// key repeat is uneven across emulators, so movement is paced here rather
	// than left to however fast the keyboard happens to autorepeat.
	moveWait = 2

	// shotSpeed and bombSpeed are rows per tick. A shot outruns a bomb by a
	// good margin: you are meant to be able to shoot your way out of one.
	shotSpeed = 1.1
	bombSpeed = 0.42

	clearedFor = 40
	invulnFor  = 50

	turretLife    = 240
	turretCadence = 14

	sweepDamage = 8

	// bombDrop is what one bomb costs, and landDrop what a boss's volley does.
	bombDrop = 1
	bossDrop = 2
)

// Shot is one of yours, travelling up.
type Shot struct {
	X, Y   float64
	Damage int
	Pierce int
	Homing bool
	Splash int
	Hit    int
}

// Bomb is one of theirs, travelling down.
type Bomb struct {
	X, Y float64
	Hurt int
}

// Turret is what the architect's branch drops: it stays put and fires upward on
// its own.
type Turret struct {
	Col    int
	Life   int
	Next   int
	Damage int
}

// BossState is the one big sprite that closes every fifth wave.
type BossState struct {
	Of    int // index into Bosses
	X, Y  float64
	Dir   int
	HP    int
	MaxHP int
	Fire  int
	Alive bool
}

// Game is a whole run in one value.
//
// Tick takes one and returns another and never writes through the slices it was
// handed, so a test can hold a state, tick it twice from the same value and get
// the same answer both times.
type Game struct {
	Field Field
	Kit   Kit
	Form  string
	Level int

	Wave  Wave
	Phase Phase
	Frame int
	Rand  uint64

	Ship     int // the leftmost column of the creature
	MoveWait int
	HP       int
	Reload   int // ticks until you may fire again
	Ready    int // ticks until the ability is ready; 0 is ready
	Invuln   int
	Revived  bool

	Score int
	Kills int
	Rest  int

	Squad   Formation
	Boss    BossState
	Started int // members the wave began with, for the block's pace

	Shots   []Shot
	Bombs   []Bomb
	Turrets []Turret

	Banner string
}

// Message ids for Banner. render.go turns them into words; the tick knows none.
const (
	BannerCleared = "cleared"
	BannerBoss    = "boss"
	BannerBossOff = "bossdown"
	BannerRevived = "revived"
	BannerOver    = "over"
	BannerLanded  = "landed"
	BannerClaude  = "claude"
	BannerPaused  = "paused"
)

// NewGame starts or resumes a run. The form and level are the pet's, read once:
// the ship is whatever creature you have right now, and it does not change
// mid-run even if a hook feeds the pet while you play.
func NewGame(f Field, form string, level int, s Save) Game {
	kit := KitFor(form, level)
	hp := s.HP
	if hp < 1 || hp > kit.MaxHP {
		hp = kit.MaxHP
	}
	seed := s.Seed
	if seed == 0 {
		seed = 0x9E3779B97F4A7C15
	}
	g := Game{
		Field: f, Kit: kit, Form: form, Level: level,
		Phase: Playing,
		Rand:  seed,
		Ship:  f.ShipColMax() / 2,
		HP:    hp,
		Score: s.Score, Kills: s.Kills,
		Revived: s.Revived,
	}
	return g.startWave(s.Wave)
}

// startWave sets up the wave at the top of it, which is the only granularity a
// run ever resumes at.
func (g Game) startWave(n int) Game {
	g.Wave = WaveFor(n, g.Field)
	g.Shots = nil
	g.Bombs = nil
	g.Turrets = nil
	g.Rest = 0
	g.Phase = Playing
	g.Banner = ""
	g.Boss = BossState{}
	g.Squad = Formation{}

	if g.Wave.Boss {
		g.Boss = BossState{
			Of: g.Wave.BossOf, X: float64((g.Field.Cols - BossCols) / 2), Y: 1,
			Dir: 1, HP: g.Wave.BossHP, MaxHP: g.Wave.BossHP,
			Fire: g.Kit.Cadence, Alive: true,
		}
		g.Started = 1
		g.Banner = BannerBoss
		return g
	}
	g.Squad = NewFormation(g.Wave, g.Field)
	g.Started = len(g.Squad.Members)
	return g
}

// Vital is the ship's state for pet.DrawCompact: HP drives it, so the creature
// visibly droops as it is worn down and lies down when the run is over.
func (g Game) Vital() pet.Vital {
	if g.Kit.MaxHP <= 0 {
		return pet.Vitals[0]
	}
	hurt := float64(g.Kit.MaxHP-g.HP) / float64(g.Kit.MaxHP)
	return pet.StateFor(100 * hurt)
}

// Muzzle is the column your shots leave from: the middle of the creature.
func (g Game) Muzzle() float64 { return float64(g.Ship) + float64(ShipCols)/2 }

// ToSave is the run as it goes to disk: the top of the wave it is on, with the
// life it had.
func (g Game) ToSave(prev Save) Save {
	out := prev
	out.Wave = g.Wave.N
	out.HP = g.HP
	out.Score = g.Score
	out.Kills = g.Kills
	out.Revived = g.Revived
	out.Seed = g.Rand
	if g.Wave.N > out.BestWave {
		out.BestWave = g.Wave.N
	}
	if g.Score > out.BestScore {
		out.BestScore = g.Score
	}
	if g.Phase == Over {
		out.Wave = 1
		out.HP = g.Kit.MaxHP
		out.Score = 0
		out.Kills = 0
		out.Revived = false
		out.Runs = prev.Runs + 1
	}
	return out.sane()
}

// Tick advances one frame.
func Tick(g Game, in Key) Game {
	switch g.Phase {
	case Over:
		return g
	case Paused:
		if in == Pause {
			g.Phase = Playing
			g.Banner = ""
		}
		return g
	}
	if in == Pause {
		g.Phase = Paused
		g.Banner = BannerPaused
		return g
	}

	g.Frame++
	if g.Rest > 0 {
		g.Rest--
		if g.Rest == 0 {
			return g.startWave(g.Wave.N + 1)
		}
		return g
	}

	g = g.moveShip(in)
	g = g.tickWeapon(in)
	g = g.tickAbility(in)
	g = g.moveShots()
	g = g.moveTurrets()
	g = g.moveSquad()
	g = g.moveBoss()
	g = g.dropBombs()
	g = g.moveBombs()
	g = g.resolveHits()
	return g.bookkeep()
}

func (g Game) moveShip(in Key) Game {
	if g.MoveWait > 0 {
		g.MoveWait--
	}
	if g.MoveWait > 0 {
		return g
	}
	switch in {
	case Left:
		if g.Ship > 0 {
			g.Ship--
			g.MoveWait = moveWait
		}
	case Right:
		if g.Ship < g.Field.ShipColMax() {
			g.Ship++
			g.MoveWait = moveWait
		}
	}
	return g
}

// volleysInFlight is how many of your PRESSES may be in the air at once.
//
// This is the rule that makes it a game rather than a hose. The arcade allowed
// exactly one shot on the screen, and that is what turns every press into a
// decision: miss, and you wait for it to reach the top before you may try again.
// Two is the concession to a terminal, where a frame is 50ms and one would feel
// like lag rather than like discipline.
//
// Measured: without any cap a level-six title cleared forty-five waves in three
// minutes - four seconds a wave - because nothing limited how much lead was in
// the air.
const volleysInFlight = 2

// InFlight is that cap counted in projectiles, since a wide kit fires several at
// once and half a volley is not a thing.
//
// One volley always fits, whatever the ceiling says. A loom at level six fires
// seven at a time, and with a flat cap of six it could never fire at all: the
// whole-volley check would refuse every press for the length of the run. Its own
// playability test caught that, which is the reason that test exists.
func (k Kit) InFlight() int {
	n := clamp(volleysInFlight*k.Shots, 2, 6)
	if n < k.Shots {
		n = k.Shots
	}
	return n
}

// tickWeapon is the gun, and the gun is yours: it fires when you press and not
// before.
//
// The first draft fired by itself and aimed by itself, which left the player one
// verb - move - and nothing to be good at. The kit still decides everything
// about the shot; what it no longer decides is when.
func (g Game) tickWeapon(in Key) Game {
	if g.Reload > 0 {
		g.Reload--
	}
	// The whole volley has to fit, or a wide kit would creep past the cap one
	// projectile at a time - which its own test caught.
	if in != Fire || g.Reload > 0 || len(g.Shots)+g.Kit.Shots > g.Kit.InFlight() {
		return g
	}
	g.Reload = g.Kit.Cadence
	return g.volley(g.Kit.Shots)
}

// damageFor is what one shot lands, with the feral branch's passive and the
// gremlin's spike applied.
func (g *Game) damageFor() int {
	damage := g.Kit.Damage
	if g.Kit.Overload {
		damage += (g.Kit.MaxHP - g.HP) / 2
	}
	if g.Kit.Spike > 0 && chance(&g.Rand, g.Kit.Spike) {
		damage *= 2
	}
	return damage
}

// volley fires n shots, spread across the creature's own width.
func (g Game) volley(n int) Game {
	if n < 1 {
		n = 1
	}
	shots := append([]Shot{}, g.Shots...)
	top := float64(g.Field.ShipRow())
	for i := 0; i < n; i++ {
		x := g.Muzzle()
		if n > 1 {
			// Spread across the creature, never wider than it is.
			span := float64(ShipCols - 3)
			x = float64(g.Ship) + 1.5 + span*float64(i)/float64(n-1)
		}
		shots = append(shots, Shot{
			X: x, Y: top,
			Damage: g.damageFor(),
			Pierce: g.Kit.Pierce,
			Homing: g.Kit.Homing,
			Splash: g.Kit.Splash,
		})
	}
	g.Shots = shots
	return g
}

func (g Game) tickAbility(in Key) Game {
	if g.Ready > 0 {
		g.Ready--
	}
	if g.Invuln > 0 {
		g.Invuln--
	}
	if in != Ability || g.Ready > 0 {
		return g
	}
	g.Ready = g.Kit.Cooldown

	switch g.Kit.Special {
	case AbilitySweep:
		g = g.column(g.Ship, ShipCols, sweepDamage)
	case AbilityThree:
		g = g.column(g.Ship-cellCols, ShipCols+2*cellCols, sweepDamage)
	case AbilityTurret, AbilityTurret2:
		n := 1
		if g.Kit.Special == AbilityTurret2 {
			n = 2
		}
		turrets := append([]Turret{}, g.Turrets...)
		for i := 0; i < n; i++ {
			col := clamp(g.Ship+2+i*4, 0, g.Field.Cols-1)
			turrets = append(turrets, Turret{Col: col, Life: turretLife, Next: turretCadence, Damage: g.Kit.Damage})
		}
		g.Turrets = turrets
	case AbilityInvuln:
		g.Invuln = invulnFor
	case AbilityBlast:
		g = g.column(0, g.Field.Cols, sweepDamage*2)
	case AbilityChimera:
		g = g.column(g.Ship, ShipCols, sweepDamage)
		g = g.volley(3)
	default: // AbilityVolley
		g = g.volley(3)
	}
	return g
}

// column hurts everything standing over a stretch of the field, which is what a
// beam fired straight up is.
func (g Game) column(from, width, damage int) Game {
	to := from + width
	if g.Boss.Alive && float64(to) > g.Boss.X && float64(from) < g.Boss.X+BossCols {
		g.Boss.HP -= damage
		if g.Boss.HP <= 0 {
			g = g.killBoss()
		}
	}
	next := make([]Member, 0, len(g.Squad.Members))
	for _, m := range g.Squad.Members {
		x, _ := g.Squad.At(m)
		if x+TroopCols > from && x < to {
			m.HP -= damage
			if m.HP <= 0 {
				g = g.killMember(m)
				continue
			}
		}
		next = append(next, m)
	}
	g.Squad.Members = next
	return g
}

func (g Game) moveShots() Game {
	next := make([]Shot, 0, len(g.Shots))
	target, found := g.nearestColumn()
	for _, s := range g.Shots {
		if s.Homing && found && g.Frame%2 == 0 {
			if target < s.X-0.5 {
				s.X--
			} else if target > s.X+0.5 {
				s.X++
			}
		}
		s.Y -= shotSpeed
		if s.Y >= 0 {
			next = append(next, s)
		}
	}
	g.Shots = next
	return g
}

// nearestColumn is the column a homing shot leans towards: the lowest thing on
// the field, because that is what is about to land on you.
func (g Game) nearestColumn() (float64, bool) {
	best, low, found := 0.0, -1.0, false
	if g.Boss.Alive {
		best, low, found = g.Boss.X+BossCols/2, g.Boss.Y, true
	}
	for _, m := range g.Squad.Members {
		x, y := g.Squad.At(m)
		if !found || float64(y) > low {
			best, low, found = float64(x)+TroopCols/2, float64(y), true
		}
	}
	return best, found
}

func (g Game) moveTurrets() Game {
	next := make([]Turret, 0, len(g.Turrets))
	shots := append([]Shot{}, g.Shots...)
	for _, t := range g.Turrets {
		t.Life--
		if t.Life <= 0 {
			continue
		}
		if t.Next > 0 {
			t.Next--
		} else {
			t.Next = turretCadence
			shots = append(shots, Shot{X: float64(t.Col), Y: float64(g.Field.ShipRow()), Damage: t.Damage})
		}
		next = append(next, t)
	}
	g.Turrets = next
	g.Shots = shots
	return g
}

// moveSquad walks the block sideways and steps it down at the walls.
func (g Game) moveSquad() Game {
	if len(g.Squad.Members) == 0 {
		return g
	}
	if g.Squad.Wait > 0 {
		g.Squad.Wait--
		return g
	}
	g.Squad.Wait = StepEvery(g.Wave, len(g.Squad.Members), g.Started)
	g.Squad.Steps++

	left, right := g.Squad.Edges()
	if (g.Squad.Dir > 0 && right >= g.Field.Cols) || (g.Squad.Dir < 0 && left <= 0) {
		g.Squad.Dir = -g.Squad.Dir
		g.Squad.Y++
		return g
	}
	g.Squad.X += float64(g.Squad.Dir)
	return g
}

func (g Game) moveBoss() Game {
	if !g.Boss.Alive {
		return g
	}
	speed := 0.25 + 0.35*float64(g.Boss.MaxHP-g.Boss.HP)/float64(max(g.Boss.MaxHP, 1))
	g.Boss.X += speed * float64(g.Boss.Dir)
	if g.Boss.X < 0 {
		g.Boss.X, g.Boss.Dir = 0, 1
	}
	if g.Boss.X > float64(g.Field.Cols-BossCols) {
		g.Boss.X, g.Boss.Dir = float64(g.Field.Cols-BossCols), -1
	}
	// It leans down as it is worn, so a long fight is a closing one.
	if g.Frame%90 == 0 && int(g.Boss.Y)+BossRows < g.Field.ShipRow()-1 {
		g.Boss.Y++
	}
	if g.Boss.Fire > 0 {
		g.Boss.Fire--
		return g
	}
	g.Boss.Fire = clamp(g.Wave.Drop/2, 8, 60)
	bombs := append([]Bomb{}, g.Bombs...)
	for _, dx := range [3]float64{1, BossCols / 2, BossCols - 2} {
		bombs = append(bombs, Bomb{X: g.Boss.X + dx, Y: g.Boss.Y + BossRows, Hurt: bossDrop})
	}
	g.Bombs = bombs
	return g
}

// dropBombs lets the lowest member of a random column fire down the screen.
func (g Game) dropBombs() Game {
	if len(g.Squad.Members) == 0 || g.Frame%max(g.Wave.Drop, 1) != 0 {
		return g
	}
	lowest := map[int]Member{}
	for _, m := range g.Squad.Members {
		if cur, ok := lowest[m.Col]; !ok || m.Row > cur.Row {
			lowest[m.Col] = m
		}
	}
	cols := make([]int, 0, len(lowest))
	for col := range lowest {
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return g
	}
	// Sorted, because ranging a map is not the same order twice and the tick
	// has to be reproducible from its seed.
	for i := 1; i < len(cols); i++ {
		for j := i; j > 0 && cols[j] < cols[j-1]; j-- {
			cols[j], cols[j-1] = cols[j-1], cols[j]
		}
	}
	m := lowest[cols[roll(&g.Rand, len(cols))]]
	x, y := g.Squad.At(m)
	g.Bombs = append(append([]Bomb{}, g.Bombs...),
		Bomb{X: float64(x) + TroopCols/2, Y: float64(y + TroopRows), Hurt: bombDrop})
	return g
}

func (g Game) moveBombs() Game {
	next := make([]Bomb, 0, len(g.Bombs))
	for _, b := range g.Bombs {
		b.Y += bombSpeed
		if b.Y >= float64(g.Field.Rows) {
			continue
		}
		if int(b.Y) >= g.Field.ShipRow() &&
			b.X >= float64(g.Ship) && b.X < float64(g.Ship+ShipCols) {
			g = g.wound(b.Hurt)
			continue
		}
		next = append(next, b)
	}
	g.Bombs = next
	return g
}

// resolveHits walks the shots against the block and the boss.
//
// The members are indexed by row first, once, so a shot only ever looks at the
// rows it is passing through.
func (g Game) resolveHits() Game {
	if len(g.Shots) == 0 {
		return g
	}
	members := append([]Member{}, g.Squad.Members...)
	byRow := make(map[int][]int, len(members))
	for i := range members {
		_, y := g.Squad.At(members[i])
		for row := y; row < y+TroopRows; row++ {
			byRow[row] = append(byRow[row], i)
		}
	}

	shots := make([]Shot, 0, len(g.Shots))
	for _, s := range g.Shots {
		alive := true
		if g.Boss.Alive && g.hitsBoss(s) {
			g.Boss.HP -= s.Damage
			s.Hit++
			if s.Hit > s.Pierce {
				alive = false
			}
			if g.Boss.HP <= 0 {
				g = g.killBoss()
			}
			if alive {
				shots = append(shots, s)
			}
			continue
		}
		for _, i := range byRow[int(s.Y)] {
			if members[i].HP <= 0 {
				continue
			}
			x, _ := g.Squad.At(members[i])
			if s.X < float64(x) || s.X >= float64(x+TroopCols) {
				continue
			}
			members[i].HP -= s.Damage
			if s.Splash > 0 {
				// Sideways, along the row, and not up and down: a formation is
				// three rows deep and eleven wide, so a vertical splash is
				// either nothing or the whole column.
				for j := range members {
					if j == i || members[j].HP <= 0 {
						continue
					}
					if members[j].Row != members[i].Row {
						continue
					}
					if abs(members[j].Col-members[i].Col) <= s.Splash {
						members[j].HP -= s.Damage
					}
				}
			}
			s.Hit++
			if s.Hit > s.Pierce {
				alive = false
			}
			break
		}
		if alive {
			shots = append(shots, s)
		}
	}
	g.Shots = shots

	next := make([]Member, 0, len(members))
	for _, m := range members {
		if m.HP > 0 {
			next = append(next, m)
			continue
		}
		g = g.killMember(m)
	}
	g.Squad.Members = next
	return g
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (g Game) hitsBoss(s Shot) bool {
	return s.Y >= g.Boss.Y && s.Y < g.Boss.Y+BossRows &&
		s.X >= g.Boss.X && s.X < g.Boss.X+BossCols
}

// killMember scores one of the block. What it is worth is what its stage is
// worth, off the canvas.
func (g Game) killMember(m Member) Game {
	g.Kills++
	g.Score += Stages[Troops[m.Species].Stage-1].Points
	return g
}

func (g Game) killBoss() Game {
	g.Kills++
	g.Boss.Alive = false
	g.Boss.HP = 0
	g.Score += Ranks[Bosses[g.Boss.Of].Rank-1].Points
	return g
}

// wound takes life, unless the mole's ability is up.
func (g Game) wound(hp int) Game {
	if g.Invuln > 0 {
		return g
	}
	g.HP -= hp
	if g.HP > 0 {
		return g
	}
	if g.Kit.Revive && !g.Revived {
		g.Revived = true
		g.HP = g.Kit.MaxHP / 2
		if g.HP < 1 {
			g.HP = 1
		}
		g.Banner = BannerRevived
		return g
	}
	g.HP = 0
	g.Phase = Over
	g.Banner = BannerOver
	return g
}

// bookkeep closes a wave when there is nothing left of it, and ends the run when
// the block lands.
func (g Game) bookkeep() Game {
	if g.Phase != Playing {
		return g
	}

	// They land on you. This is the other way to lose, and the one the arcade
	// is famous for: bombs whittle you down, but the block arriving is over
	// whatever life you had left. It is also what stops a slow gun from simply
	// waiting the wave out.
	if len(g.Squad.Members) > 0 && g.Squad.Bottom() >= g.Field.ShipRow() {
		g.HP = 0
		g.Phase = Over
		g.Banner = BannerLanded
		return g
	}
	if g.Boss.Alive && int(g.Boss.Y)+BossRows >= g.Field.ShipRow() {
		g.HP = 0
		g.Phase = Over
		g.Banner = BannerLanded
		return g
	}

	if len(g.Squad.Members) > 0 || g.Boss.Alive {
		return g
	}

	if g.Wave.Boss {
		g.HP = g.Kit.MaxHP
		g.Banner = BannerBossOff
	} else {
		g.HP += g.Kit.Regen
		if g.HP > g.Kit.MaxHP {
			g.HP = g.Kit.MaxHP
		}
		g.Banner = BannerCleared
	}
	g.Phase = Cleared
	g.Rest = clearedFor
	g.Shots = nil
	g.Bombs = nil
	return g
}
