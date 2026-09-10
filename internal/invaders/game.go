package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// The tick. No terminal, no files, no clock: this file takes a state and a key
// and returns the next state, which is what makes waves, kits, rivals and the
// life rules testable without a tty - the same way internal/pet is testable
// without a statusline.
//
// It must never write pet.json either. A run does cost the creature a level, but
// that happens once, in run.go, when the run is over. A tick that could reach the
// pet would punish it twenty times a second. See TestTheTickNeverTouchesThePetOrTheTerminal.

// Key is one decoded keypress.
type Key uint8

const (
	None Key = iota
	Up
	Down
	Fire // the ability; the gun is automatic
	Pause
	Quit
)

// Phase is what the run is doing.
type Phase uint8

const (
	Playing Phase = iota
	Cleared       // the pause between waves
	Paused
	Over
)

const (
	// moveWait is ticks between one row of movement and the next. Terminal key
	// repeat is uneven across emulators, so movement is gated here rather than
	// left to however fast the keyboard happens to autorepeat.
	moveWait = 3

	// shotSpeed and boltSpeed are cells per tick.
	shotSpeed = 1.4
	boltSpeed = 0.9

	// clearedFor is how long the banner between waves stays up.
	clearedFor = 30

	// invulnFor is how long the mole's ability lasts.
	invulnFor = 40

	// turretLife and turretCadence are the architect's.
	turretLife    = 200
	turretCadence = 12

	// sweepDamage is what a beam does to everything in a row. Big enough to be
	// worth timing, not big enough to replace the gun.
	sweepDamage = 8

	// darterNear is how close a darter has to be before it lunges, and
	// darterBoost is by how much.
	darterNear  = 18
	darterBoost = 2.2

	// weaverEvery is ticks between a weaver's changes of row.
	weaverEvery = 12

	// bossDrop is what one of the swarm costs when it reaches you, and boltDrop
	// what a rival's shot costs.
	leakDrop = 1
	boltDrop = 2
)

// Shot is one of yours, travelling right.
type Shot struct {
	X      float64
	Row    int
	Damage int
	Pierce int
	Homing bool
	Splash int
	Hit    int // enemies already passed through
}

// Bolt is a rival's, travelling left.
type Bolt struct {
	X   float64
	Row int
}

// Turret is what the architect's branch drops: it stays put and fires on its own.
type Turret struct {
	Row    int
	Life   int
	Next   int
	Damage int
}

// Game is a whole run in one value.
//
// Tick takes one and returns another and never writes through the slices it was
// handed, so a test can hold a state, tick it twice from the same value and get
// the same answer both times. At twenty ticks a second with a few hundred
// entities the copying costs nothing measurable, and it buys the one property
// every other test in this package leans on.
type Game struct {
	Field Field
	Kit   Kit
	Form  string
	Level int

	Wave  Wave
	Phase Phase
	Frame int    // frames since the run started; the walk cycle comes off it too
	Rand  uint64 // splitmix64, straight out of the save file

	Ship     int // the sprite's top row
	MoveWait int
	HP       int
	Volley   int // ticks to the next automatic volley
	Ready    int // ticks until the ability is ready; 0 is ready
	Invuln   int
	Revived  bool

	Score     int
	Kills     int
	Released  int // enemies of this wave already sent
	NextSquad int
	Rest      int // ticks left of the between-waves banner

	Enemies []Enemy
	Shots   []Shot
	Bolts   []Bolt
	Turrets []Turret

	Banner string // a message id, resolved by render.go; "" for none
}

// Message ids for Banner. render.go turns them into words; the tick does not
// know any.
const (
	BannerCleared = "cleared"
	BannerBoss    = "boss"
	BannerBossOff = "bossdown"
	BannerRevived = "revived"
	BannerOver    = "over"
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
		Ship:  f.ShipRowMax() / 2,
		HP:    hp,
		Score: s.Score, Kills: s.Kills,
		Revived: s.Revived,
		Volley:  kit.Cadence,
	}
	return g.startWave(s.Wave)
}

// startWave sets up the wave at the top of it, which is the only granularity a
// run ever resumes at.
func (g Game) startWave(n int) Game {
	g.Wave = WaveFor(n, g.Field)
	g.Released = 0
	g.NextSquad = 0
	g.Enemies = nil
	g.Shots = nil
	g.Bolts = nil
	g.Turrets = nil
	g.Rest = 0
	g.Phase = Playing
	g.Banner = ""
	if g.Wave.Boss {
		g.Enemies = []Enemy{g.Wave.BossFor(g.Field, g.Form, g.Level)}
		g.Released = g.Wave.Count
		g.Banner = BannerBoss
	}
	return g
}

// Vital is the ship's state for pet.Draw: HP drives it, so the creature visibly
// droops as it is worn down and lies down when the run is over. Seven states
// that already exist and are already tested, instead of a health bar.
func (g Game) Vital() pet.Vital {
	if g.Kit.MaxHP <= 0 {
		return pet.Vitals[0]
	}
	hurt := float64(g.Kit.MaxHP-g.HP) / float64(g.Kit.MaxHP)
	return pet.StateFor(100 * hurt)
}

// Face is the row the ship's eyes and its gun are on.
func (g Game) Face() int { return g.Ship + HitTop + 1 }

// ToSave is the run as it goes to disk: the top of the wave it is on, with the
// life it had. Quit mid-wave and you come back to the start of it - forgiving,
// and it is what makes the auto-pause cheap to get right.
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
	g = g.tickAbility(in)
	g = g.fire()
	g = g.release()
	g = g.moveShots()
	g = g.moveTurrets()
	g = g.moveEnemies()
	g = g.moveBolts()
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
	case Up:
		if g.Ship > 0 {
			g.Ship--
			g.MoveWait = moveWait
		}
	case Down:
		if g.Ship < g.Field.ShipRowMax() {
			g.Ship++
			g.MoveWait = moveWait
		}
	}
	return g
}

func (g Game) tickAbility(in Key) Game {
	if g.Ready > 0 {
		g.Ready--
	}
	if g.Invuln > 0 {
		g.Invuln--
	}
	if in != Fire || g.Ready > 0 {
		return g
	}
	g.Ready = g.Kit.Cooldown

	switch g.Kit.Special {
	case AbilitySweep:
		g = g.beam(g.Face(), sweepDamage)
	case AbilityThree:
		for _, row := range [3]int{g.Face() - 1, g.Face(), g.Face() + 1} {
			g = g.beam(row, sweepDamage)
		}
	case AbilityTurret, AbilityTurret2:
		n := 1
		if g.Kit.Special == AbilityTurret2 {
			n = 2
		}
		for i := 0; i < n; i++ {
			row := g.Face() + i*2 - (n - 1)
			if row >= 0 && row < g.Field.Rows {
				g.Turrets = append(append([]Turret{}, g.Turrets...), Turret{
					Row: row, Life: turretLife, Next: turretCadence, Damage: g.Kit.Damage,
				})
			}
		}
	case AbilityInvuln:
		g.Invuln = invulnFor
	case AbilityBlast:
		for row := 0; row < g.Field.Rows; row++ {
			g = g.beam(row, sweepDamage*2)
		}
	case AbilityChimera:
		g = g.beam(g.Face(), sweepDamage)
		g = g.volley(3)
	default: // AbilityVolley
		g = g.volley(3)
	}
	return g
}

// beam hurts everything in one row at once, which is what a sweep is.
func (g Game) beam(row, damage int) Game {
	next := make([]Enemy, 0, len(g.Enemies))
	for _, e := range g.Enemies {
		top, bottom := e.Solid()
		if row >= top && row <= bottom {
			e.HP -= damage
			if e.HP <= 0 {
				var kids []Enemy
				g, kids = g.kill(e)
				next = append(next, kids...)
				continue
			}
		}
		next = append(next, e)
	}
	g.Enemies = next
	return g
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

// volley fires n projectiles, spread across the rows the gun can cover.
func (g Game) volley(n int) Game {
	if n < 1 {
		n = 1
	}
	shots := append([]Shot{}, g.Shots...)
	face := g.Face()
	for i := 0; i < n; i++ {
		row := face
		switch {
		case n == 2:
			row = face - 1 + 2*i
		case n >= 3:
			row = face - 1 + i%3
		}
		if row < 0 || row >= g.Field.Rows {
			row = face
		}
		shots = append(shots, Shot{
			X:      float64(ShipCols),
			Row:    row,
			Damage: g.damageFor(),
			Pierce: g.Kit.Pierce,
			Homing: g.Kit.Homing,
			Splash: g.Kit.Splash,
		})
	}
	g.Shots = shots
	return g
}

func (g Game) fire() Game {
	if g.Volley > 0 {
		g.Volley--
		return g
	}
	g.Volley = g.Kit.Cadence
	return g.volley(g.Kit.Shots)
}

// release sends the next squad when it is due.
func (g Game) release() Game {
	if g.Wave.Boss || g.Released >= g.Wave.Count {
		return g
	}
	if g.NextSquad > 0 {
		g.NextSquad--
		return g
	}
	g.NextSquad = g.Wave.Every
	squad := g.Wave.Spawn(g.Field, &g.Rand)
	if left := g.Wave.Count - g.Released; len(squad) > left {
		squad = squad[:left]
	}
	g.Enemies = append(append([]Enemy{}, g.Enemies...), squad...)
	g.Released += len(squad)
	return g
}

func (g Game) moveShots() Game {
	next := make([]Shot, 0, len(g.Shots))
	// The homing target is picked once per tick rather than once per shot: with
	// a wide field and a full wave the per-shot version is the one loop in here
	// that could actually be felt.
	target, found := g.nearestEnemy()
	for _, s := range g.Shots {
		if s.Homing && found && g.Frame%2 == 0 {
			if target < s.Row {
				s.Row--
			} else if target > s.Row {
				s.Row++
			}
		}
		s.X += shotSpeed
		if s.X < float64(g.Field.Cols) {
			next = append(next, s)
		}
	}
	g.Shots = next
	return g
}

func (g Game) nearestEnemy() (row int, found bool) {
	best := 0.0
	for _, e := range g.Enemies {
		if !found || e.X < best {
			row, best, found = e.Row, e.X, true
		}
	}
	return row, found
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
			shots = append(shots, Shot{X: float64(ShipCols), Row: t.Row, Damage: t.Damage})
		}
		next = append(next, t)
	}
	g.Turrets = next
	g.Shots = shots
	return g
}

func (g Game) moveEnemies() Game {
	next := make([]Enemy, 0, len(g.Enemies))
	first, last := g.Field.Lanes()
	for _, e := range g.Enemies {
		if e.Boss() {
			g, e = g.moveBoss(e)
			next = append(next, e)
			continue
		}

		speed := e.Speed
		switch e.Trait {
		case Weaver:
			e.Phase++
			if e.Phase%weaverEvery == 0 {
				if row := e.Row + e.Drift; row < first || row > last {
					e.Drift = -e.Drift
				} else {
					e.Row = row
				}
			}
		case Darter:
			if e.X < darterNear {
				speed *= darterBoost
			}
		}
		e.X -= speed

		// It reaches you either by touching the ship or by getting past it, and
		// both cost the same one point of life. It is never killed by arriving:
		// it costs a life and leaves.
		if g.hitsShip(e) || e.X < 0 {
			g = g.wound(leakDrop)
			continue
		}
		next = append(next, e)
	}
	g.Enemies = next
	return g
}

// hitsShip is whether an enemy has reached the ship's solid rows. The crest and
// the feet are not part of the ship - things pass through them - which is what
// lets a five-row creature be a fair hitbox.
func (g Game) hitsShip(e Enemy) bool {
	if e.Boss() {
		return false
	}
	if int(e.X) > ShipCols-1 {
		return false
	}
	top, bottom := e.Solid()
	return bottom >= g.Ship+HitTop && top <= g.Ship+HitBottom
}

func (g Game) moveBoss(e Enemy) (Game, Enemy) {
	left, right := g.Field.BossBounds()
	e.X += e.Speed * float64(e.Drift)
	if e.X < left {
		e.X, e.Drift = left, 1
	}
	if e.X > right {
		e.X, e.Drift = right, -1
	}
	// It drifts toward your row, so standing still is not a strategy.
	if g.Frame%14 == 0 {
		switch {
		case e.Row+HitTop+1 < g.Face() && e.Row < g.Field.ShipRowMax():
			e.Row++
		case e.Row+HitTop+1 > g.Face() && e.Row > 0:
			e.Row--
		}
	}
	if e.Fire > 0 {
		e.Fire--
	} else {
		// Its cadence is its own kit's, and it quickens as it is worn down: the
		// phases of the fight are the seven vital states, and this is what they
		// feel like from your side.
		e.Fire = e.Kit.Cadence * (7 - e.Vital().Rank) / 7
		if e.Fire < 4 {
			e.Fire = 4
		}
		g.Bolts = append(append([]Bolt{}, g.Bolts...), Bolt{X: e.X, Row: e.Row + HitTop + 1})
	}
	return g, e
}

func (g Game) moveBolts() Game {
	next := make([]Bolt, 0, len(g.Bolts))
	for _, b := range g.Bolts {
		b.X -= boltSpeed
		if b.X < 0 {
			continue
		}
		if int(b.X) <= ShipCols-1 && b.Row >= g.Ship+HitTop && b.Row <= g.Ship+HitBottom {
			g = g.wound(boltDrop)
			continue
		}
		next = append(next, b)
	}
	g.Bolts = next
	return g
}

// resolveHits walks the shots against the enemies.
//
// The enemies are indexed by row first, once, so a shot only ever looks at its
// own lane. Walking every enemy for every shot is fine at wave 5 and is ten to
// the fifth per frame at wave 200 with a full field, which is the one loop in
// here that could actually be felt.
func (g Game) resolveHits() Game {
	if len(g.Shots) == 0 || len(g.Enemies) == 0 {
		return g
	}
	enemies := append([]Enemy{}, g.Enemies...)

	byLane := make(map[int][]int, len(enemies))
	for i := range enemies {
		top, bottom := enemies[i].Solid()
		for row := top; row <= bottom; row++ {
			byLane[row] = append(byLane[row], i)
		}
	}

	shots := make([]Shot, 0, len(g.Shots))
	for _, s := range g.Shots {
		alive := true
		for _, i := range byLane[s.Row] {
			if enemies[i].HP <= 0 {
				continue
			}
			if s.X < enemies[i].X || s.X > enemies[i].X+float64(enemies[i].W()) {
				continue
			}

			enemies[i].HP -= plateAdjusted(enemies[i], s)
			if s.Splash > 0 {
				for row := s.Row - s.Splash; row <= s.Row+s.Splash; row++ {
					for _, j := range byLane[row] {
						if j == i || enemies[j].HP <= 0 {
							continue
						}
						enemies[j].HP -= plateAdjusted(enemies[j], s)
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

	next := make([]Enemy, 0, len(enemies))
	for _, e := range enemies {
		if e.HP > 0 {
			next = append(next, e)
			continue
		}
		var kids []Enemy
		g, kids = g.kill(e)
		next = append(next, kids...)
	}
	g.Enemies = next
	return g
}

// plateAdjusted is the one place the bestiary and the kits meet: plate holds a
// shot to a single point unless it pierces, which is what the cannon family and
// the sniper mark are for.
func plateAdjusted(e Enemy, s Shot) int {
	if e.Trait == Plated && s.Pierce == 0 && s.Damage > 1 {
		return 1
	}
	return s.Damage
}

// kill scores an enemy and hands back whatever it leaves behind.
//
// The children come back rather than being appended to g.Enemies, because every
// caller of this is in the middle of rebuilding that slice: the first draft
// appended them and then overwrote the slice it had appended to, so a splitter
// scored and split into nothing at all.
func (g Game) kill(e Enemy) (Game, []Enemy) {
	g.Kills++
	if e.Boss() {
		g.Score += 500 + 50*g.Wave.N
		return g, nil
	}
	g.Score += 10 + g.Wave.N
	return g, e.SplitInto(g.Field)
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
	// The phoenix gets one second life, at half of what it had, and only one:
	// Revived stays true for the rest of the run and rides in the save file so
	// quitting and coming back does not buy another.
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

// bookkeep closes a wave when there is nothing left of it.
func (g Game) bookkeep() Game {
	if g.Phase != Playing {
		return g
	}
	if g.Released < g.Wave.Count || len(g.Enemies) > 0 {
		return g
	}

	if g.Wave.Boss {
		// A cleared rival heals you to full. That is what makes every fifth wave
		// a rhythm instead of a slow bleed, and it is the run's checkpoint.
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
	g.Bolts = nil
	return g
}
