package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// The tick. No terminal, no files, no clock: this file takes a state and a key
// and returns the next state, which is what makes the fleet, the kits and the
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
	Up
	Down
	Stop
	Fire
	Ability
	Rearm // reload the magazine before it runs out
	Heal  // spend a health kit
	One   // the three upgrade picks
	Two
	Three
	Pause
	Quit
	// The terminal's own two: the window took or lost the focus. They never
	// reach the tick - loop takes them out of the stream - because what they
	// steer is the keyboard's autorepeat and not the ship. See keyboard.go.
	FocusIn
	FocusOut
)

// Phase is what the run is doing.
type Phase uint8

const (
	Playing Phase = iota
	Cleared
	Paused
	// Choosing is the level-up menu: the field is frozen and the only keys that
	// mean anything are the three picks. Frozen rather than running underneath,
	// because a choice made while a bomb is in the air is not a choice.
	Choosing
	Over
)

const (
	// streamFor is how long a direction keeps going on its own after a keypress,
	// in ticks, and repeatWindow is how close together two presses have to be to
	// count as one key being held down.
	//
	// This is what "hold the arrow" means in a terminal, and it is worth being
	// exact about because the first version was wrong. A terminal is never told
	// that a key went UP: what arrives while you hold one is the operating
	// system's autorepeat, a stream of presses. So a press on its own moves ONE
	// cell and stops - a tap is a tap - and a press that arrives while the last
	// one is still recent is a key being held, which starts the glide. The glide
	// is on a short leash refreshed by every press in the stream, so it ends
	// within a couple of frames of you letting go.
	//
	// The leash is twice the gap it just measured, between two and eight ticks,
	// rather than a fixed number: desktops repeat at anything from ten to fifty a
	// second and a fixed leash is either a stutter on the slow ones or a skid on
	// the fast ones.
	//
	// The version before this LATCHED - one press and it went until you said
	// otherwise - which is smooth, needs no autorepeat at all, and is not what
	// anybody expects from an arrow key.
	streamFor    = 2
	streamMax    = 8
	turnFor      = 5
	repeatWindow = 8

	// moveEvery and climbEvery are TICKS PER CELL: one column every tick, one row
	// every four.
	//
	// A column a tick is as smooth as a character grid can be - the position
	// changes on every frame that is drawn - and at forty a second it crosses
	// eighty columns in two seconds, which is a shooter rather than a barge. It
	// used to be a column every other frame and it was the first thing that
	// looked wrong from the outside.
	//
	// Climbing is four times slower, which is ten rows a second. A terminal cell
	// is about twice as tall as it is wide, so a row reads as twice the distance
	// of a column, and the field is nine rows tall for the ship against forty-odd
	// columns wide: the same cadence on both axes had the ship crossing its own
	// half of the field before you could let go of the key.
	moveEvery  = 1
	climbEvery = 4

	// The ship LATCHES: an arrow sets it going and it keeps going until you
	// point it the other way or tell it to stop.
	//
	// This is the only thing that works. A terminal has no key-up event and no
	// way to say two keys are down at once - hold left and the operating system
	// streams left, press fire and it starts repeating THAT instead, and the
	// left never comes back until you let go and press it again. Momentum for a
	// few tenths of a second was tried first and is not enough: hold the fire
	// key and you still coast to a halt.

	// shotSpeed and bombSpeed are rows per tick. A shot outruns a bomb by a good
	// margin: you are meant to be able to shoot your way out of one.
	//
	// A shot crosses half a row a tick and a bomb a fifth of one, which is what
	// forty frames a second bought: the same speed through the air in steps half
	// as big. Nothing in the fleet is under two rows tall, so a shot still cannot
	// step clean over one - the bug that made the top row of the old block
	// unkillable.
	shotSpeed = 0.55
	bombSpeed = 0.21

	clearedFor = 80
	invulnFor  = 100

	turretLife    = 480
	turretCadence = 28

	sweepDamage = 8

	// What the nine new abilities are worth, in ticks or in rows. None of them
	// is meant to replace the gun: the longest lasts six seconds and half of
	// them are instant.
	pulseRows  = 4
	shieldFor  = 5 * TicksPerSecond
	rushFor    = 4 * TicksPerSecond
	mirrorFor  = 6 * TicksPerSecond
	netFor     = 3 * TicksPerSecond
	rageFor    = 5 * TicksPerSecond
	mirrorGap  = 7  // columns between you and the ghost
	lanceHit   = 40 // one enormous shot
	dashDamage = 12

	// What a collision between two of theirs is worth: half a cell of daylight
	// each, and enough speed to actually leave.
	bounceStep  = 0.5
	bounceLeast = 0.08

	// bombDrop is what one of their bombs costs, bossDrop one of a boss's, and
	// landDrop what it costs to let one reach the floor.
	bombDrop = 1
	bossDrop = 2
	landDrop = 1
	ramDrop  = 2

	// The two things that fall out of the sky on their own. A health kit a
	// minute is the reference's own rate; the rocks are more often because they
	// are as much a hazard as a gift.
	kitEvery  = 60 * TicksPerSecond
	rockEvery = 22 * TicksPerSecond
	kitsMax   = 3

	// sparkLife is an explosion, moteLife a meteoroid: one is decoration and
	// gone in half a second, the other crosses the field hurting things.
	sparkLife = 16
	moteLife  = 60

	// upStep is the score between one level-up offer and the next, and it grows
	// with the number already taken.
	upStep = 300
)

// The three upgrades, which are the reference's three: more damage, a quicker
// gun, a bigger magazine.
const (
	UpPower = 1
	UpSpeed = 2
	UpCap   = 3
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
	Row    int // where it was dropped: it stays there while the ship moves on
	Life   int
	Next   int
	Damage int
}

// Alien is one enemy ship in the air: which craft it is, where it is, how it is
// drifting and when it fires next.
type Alien struct {
	Of        int
	X, Y      float64
	Vx        float64
	HP, MaxHP int
	Fire      int
}

// Axis is one direction of travel under a key that may or may not still be down.
type Axis struct {
	// Way is the way the last press pointed, and so the way it is travelling
	// while Hold lasts. It outlives the glide on purpose: it is what tells the
	// next press whether it is the same key repeating or a new tap.
	Way   int // -1, 0 or 1
	Hold  int // ticks it may keep going without another press
	Since int // ticks since the last press, capped
	Wait  int // ticks until the next cell, which is what sets the speed
}

// Moving says whether this axis is carrying the ship along right now, which is
// not the same as which way it last pointed.
func (a Axis) Moving() bool { return a.Way != 0 && a.Hold > 0 }

// press is a key arriving on this axis: a tap if the last one was long ago, a
// repeat - and so a key being held - if it was not.
func (a Axis) press(way, every int) Axis {
	// A press counts as the same key held down if it points the same way, and
	// ALSO if the axis is still gliding: changing direction mid-glide is one
	// finger moving from one arrow to the other, and treating it as a fresh tap
	// stopped the ship dead for a repeat delay. Reported from play as "a veces
	// estoy presionando y se queda parado".
	turn := a.Way != way && a.Hold > 0 && a.Since <= repeatWindow
	repeat := a.Way == way && a.Since <= repeatWindow
	a.Way = way
	switch {
	case turn:
		// One finger moving from one arrow to the other. The new key is a fresh
		// press as far as the operating system is concerned, so its autorepeat
		// waits out the whole initial delay before the stream starts: the leash
		// has to be long enough to carry the ship across that gap, or the turn
		// reads as the ship parking itself for a tenth of a second.
		a.Hold = turnFor
	case repeat:
		a.Hold = clamp(2*a.Since, streamFor, streamMax)
	default:
		// A tap: one cell, now, and then nothing until another press.
		a.Hold = 0
		a.Wait = 0
	}
	a.Since = 0
	return a
}

// tick is a frame going by with no key on this axis.
func (a Axis) tick() Axis {
	if a.Since < repeatWindow+1 {
		a.Since++
	}
	if a.Hold > 0 {
		a.Hold--
	}
	if a.Wait > 0 {
		a.Wait--
	}
	return a
}

// steps says whether the ship moves this tick, and takes the step.
func (a Axis) steps(every int) (Axis, bool) {
	if a.Way == 0 || a.Wait > 0 {
		return a, false
	}
	a.Wait = every - 1
	return a, true
}

// stop is the brake, and a wall.
func (a Axis) stop() Axis {
	a.Way, a.Hold = 0, 0
	return a
}

// Craft is the kind of ship this is.
func (a Alien) Craft() Craft { return Fleet[clamp(a.Of, 0, len(Fleet)-1)] }

// Stone is an asteroid on its way down. It is on nobody's side.
type Stone struct {
	X, Y, Vx  float64
	HP, MaxHP int
}

// Mote is a fleck of something: an explosion spark when Hurt is zero, and a
// meteoroid off a broken asteroid when it is not.
type Mote struct {
	X, Y, Vx, Vy float64
	Life         int
	Hurt         int
}

// Drop is a health kit falling. Catching it puts it in the hold; the Heal key
// spends it.
type Drop struct{ X, Y float64 }

// Star is the sky. It does nothing at all, and the game looks half finished
// without it.
type Star struct {
	X, Y float64
	V    float64
}

// BossState is the one big sprite off the canvas that closes every fifth wave.
type BossState struct {
	Of    int
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

	Ship int // the leftmost column of the ship
	Row  int // the top row of the ship: the floor to start with, and up to Field.ShipRoof

	// The two axes. Each one remembers which way it is going, how long it may
	// keep going without another press, and how long since the last one - which
	// is what tells a tap from a key being held.
	Side    Axis
	Rise    Axis
	HP      int
	Ammo    int // rounds in the magazine
	Loading int // ticks left of a reload, 0 when loaded
	Cool    int // ticks until the gun may fire again
	Ready   int // ticks until the ability is ready; 0 is ready
	Invuln  int
	// The abilities that last rather than happen, each a count of ticks. Every
	// one of them is visible from the outside - see drawShip and the hud -
	// because an effect nobody can see is an effect nobody trusts.
	Shield  int // bombs burn up on the way in
	Rush    int // twice the rate of fire, and the rounds are free
	Mirror  int // a second ship beside you, firing with you
	Net     int // the fleet stops descending
	Rage    int // double damage, bought with a point of life
	Revived bool
	Kits    int // health kits in the hold

	Score int
	Kills int
	Rest  int

	// The upgrades taken, counted by kind so that a run resumed off disk comes
	// back with the gun the player built rather than the one the pet gave it.
	Power, Quick, Mag int
	NextUp            int

	Released int // ships of this wave that have been let out
	Next     int // ticks to the next release
	RockIn   int
	KitIn    int

	Aliens  []Alien
	Boss    BossState
	Shots   []Shot
	Bombs   []Bomb
	Turrets []Turret
	Stones  []Stone
	Motes   []Mote
	Drops   []Drop
	Stars   []Star

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
	BannerChoose  = "choose"
	BannerKit     = "kit"
	// BannerAgain is the game-over screen, which is a different thing from the
	// game-over banner: one says the run ended and the other asks whether to
	// start again. run.go sets it, because a tick cannot know that the shell is
	// still there to go back to.
	BannerAgain = "again"
)

// Ups is how many upgrades have been taken, of any kind.
func (g Game) Ups() int { return g.Power + g.Quick + g.Mag }

// NewGame starts or resumes a run. The form and level are the pet's, read once:
// the ship represents whatever creature you have right now, and it does not
// change mid-run even if a hook feeds the pet while you play.
func NewGame(f Field, form string, level int, s Save) Game {
	kit := KitFor(form, level)
	g := Game{
		Field: f, Form: form, Level: level,
		Phase: Playing,
		Rand:  seedOf(s),
		Ship:  f.ShipColMax() / 2,
		Row:   f.ShipRow(),
		Score: s.Score, Kills: s.Kills,
		Revived: s.Revived,
		Power:   s.Power, Quick: s.Speed, Mag: s.Mag,
	}
	// The upgrades are re-applied on the way in, in a fixed order, so that a
	// resumed run flies exactly the gun it quit with.
	g.Kit = kit
	for i := 0; i < g.Power; i++ {
		g.Kit = boost(g.Kit, UpPower)
	}
	for i := 0; i < g.Quick; i++ {
		g.Kit = boost(g.Kit, UpSpeed)
	}
	for i := 0; i < g.Mag; i++ {
		g.Kit = boost(g.Kit, UpCap)
	}

	hp := s.HP
	if hp < 1 || hp > g.Kit.MaxHP {
		hp = g.Kit.MaxHP
	}
	g.HP = hp
	g.Ammo = g.Kit.Cap
	g.NextUp = s.Score + upStep*(g.Ups()+1)
	g.Stars = sky(f, &g.Rand)
	return g.startWave(s.Wave)
}

func seedOf(s Save) uint64 {
	if s.Seed == 0 {
		return 0x9E3779B97F4A7C15
	}
	return s.Seed
}

// sky scatters the background. One star every five columns, at one of three
// speeds, which is enough to read as depth and not enough to read as weather -
// one every three was weather.
func sky(f Field, rand *uint64) []Star {
	stars := make([]Star, 0, f.Cols/5+1)
	for i := 0; i < f.Cols/5+1; i++ {
		stars = append(stars, Star{
			X: float64(roll(rand, f.Cols)),
			Y: float64(roll(rand, f.Rows)),
			V: []float64{0.03, 0.07, 0.13}[roll(rand, 3)],
		})
	}
	return stars
}

// startWave sets up the wave at the top of it, which is the only granularity a
// run ever resumes at.
func (g Game) startWave(n int) Game {
	g.Wave = WaveFor(n, g.Field)
	g.Aliens = nil
	g.Shots = nil
	g.Bombs = nil
	g.Turrets = nil
	g.Stones = nil
	g.Motes = nil
	g.Drops = nil
	g.Rest = 0
	g.Phase = Playing
	g.Banner = ""
	g.Boss = BossState{}
	g.Released = 0
	g.Next = 0
	g.Shield, g.Rush, g.Mirror, g.Net, g.Rage = 0, 0, 0, 0, 0
	g.RockIn = rockEvery
	g.KitIn = kitEvery

	if g.Wave.Boss {
		g.Boss = BossState{
			Of: g.Wave.BossOf, X: float64((g.Field.Cols - BossCols) / 2), Y: 1,
			Dir: 1, HP: g.Wave.BossHP, MaxHP: g.Wave.BossHP,
			Fire: g.Kit.Cadence, Alive: true,
		}
		g.Banner = BannerBoss
	}
	return g
}

// Vital is the ship's state, which is what its eyes show: HP drives it, so the
// representation droops as the run wears on and its eyes go out when it is over.
func (g Game) Vital() pet.Vital {
	if g.Kit.MaxHP <= 0 {
		return pet.Vitals[0]
	}
	hurt := float64(g.Kit.MaxHP-g.HP) / float64(g.Kit.MaxHP)
	return pet.StateFor(100 * hurt)
}

// Muzzle is the column your shots leave from: the middle of the ship.
func (g Game) Muzzle() float64 { return float64(g.Ship) + float64(ShipCols)/2 }

// ToSave is the run as it goes to disk: the top of the wave it is on, with the
// life, the score and the gun it had.
func (g Game) ToSave(prev Save) Save {
	out := prev
	out.Wave = g.Wave.N
	out.HP = g.HP
	out.Score = g.Score
	out.Kills = g.Kills
	out.Revived = g.Revived
	out.Seed = g.Rand
	out.Power, out.Speed, out.Mag = g.Power, g.Quick, g.Mag
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
		out.Power, out.Speed, out.Mag = 0, 0, 0
		out.Runs = prev.Runs + 1
	}
	return out.sane()
}

// Tick advances one frame on one key, which is what a test wants to say.
func Tick(g Game, in Key) Game { return TickWith(g, in, in) }

// TickWith advances one frame on a direction AND an action, which is what the
// loop has: a terminal delivers both in the same twenty-five milliseconds all the
// time - the arrow's autorepeat and the shot you just pressed - and the version
// that took one key threw one of them away. Whichever it threw away, something
// the player did did not happen.
func TickWith(g Game, move, in Key) Game {
	switch g.Phase {
	case Over:
		return g
	case Paused:
		if in == Pause {
			g.Phase = Playing
			g.Banner = ""
		}
		return g
	case Choosing:
		return g.choose(in)
	}
	if in == Pause {
		g.Phase = Paused
		g.Banner = BannerPaused
		return g
	}

	g.Frame++
	// The sky moves through everything, including the pause between waves: a
	// frozen starfield reads as a hung game.
	g = g.moveStars()
	if g.Rest > 0 {
		g.Rest--
		if g.Rest == 0 {
			return g.startWave(g.Wave.N + 1)
		}
		return g
	}

	g = g.moveShip(move)
	g = g.tickWeapon(in)
	g = g.tickAbility(in)
	g = g.tickHeal(in)
	g = g.release()
	g = g.moveShots()
	g = g.moveTurrets()
	g = g.moveAliens()
	g = g.moveBoss()
	g = g.moveStones()
	g = g.moveMotes()
	g = g.moveDrops()
	g = g.moveBombs()
	g = g.resolveHits()
	g = g.offerUpgrade()
	return g.bookkeep()
}

// choose is the level-up menu: three keys, and nothing else happens until one
// of them is pressed.
func (g Game) choose(in Key) Game {
	pick := 0
	switch in {
	case One:
		pick = UpPower
	case Two:
		pick = UpSpeed
	case Three:
		pick = UpCap
	default:
		return g
	}
	g.Kit = boost(g.Kit, pick)
	switch pick {
	case UpPower:
		g.Power++
	case UpSpeed:
		g.Quick++
	case UpCap:
		g.Mag++
	}
	// A bigger magazine you have to reload for is not a reward.
	if g.Ammo < g.Kit.Cap {
		g.Ammo = g.Kit.Cap
	}
	g.NextUp = g.Score + upStep*(g.Ups()+1)
	g.Phase = Playing
	g.Banner = ""
	return g
}

// boost is one upgrade applied to a kit. Pure, so NewGame can replay a run's
// upgrades from the three counts on disk.
func boost(k Kit, pick int) Kit {
	switch pick {
	case UpPower:
		k.Damage++
	case UpSpeed:
		k.Cadence = k.Cadence * 4 / 5
		if k.Cadence < 2 {
			k.Cadence = 2
		}
	case UpCap:
		k.Cap += 3
		k.Reload -= 4
		if k.Reload < 20 {
			k.Reload = 20
		}
	}
	return k
}

// offerUpgrade stops the game to ask, once the score has passed the next mark.
func (g Game) offerUpgrade() Game {
	if g.Phase != Playing || g.Score < g.NextUp {
		return g
	}
	g.Phase = Choosing
	g.Banner = BannerChoose
	return g
}

// moveShip steers both axes: a press moves one cell, a key held down glides, and
// letting go stops within a frame or two.
//
// It reads a stream of presses and never a release, because a terminal has no
// such thing. See the comment on streamFor for what that costs and why the glide
// is on a leash rather than latched.
func (g Game) moveShip(in Key) Game {
	switch in {
	case Left:
		g.Side = g.Side.press(-1, moveEvery)
	case Right:
		g.Side = g.Side.press(1, moveEvery)
	case Up:
		g.Rise = g.Rise.press(-1, climbEvery)
	case Down:
		g.Rise = g.Rise.press(1, climbEvery)
	case Stop:
		g.Side, g.Rise = g.Side.stop(), g.Rise.stop()
	}

	// The frame goes by for whichever axis this key was not.
	if in != Left && in != Right {
		g.Side = g.Side.tick()
	}
	if in != Up && in != Down {
		g.Rise = g.Rise.tick()
	}
	// And a key that arrived spends its own tick too, once the press has been
	// read: the wait between cells is a wait either way.
	if in == Left || in == Right {
		g.Side.Wait = spend(g.Side.Wait)
	}
	if in == Up || in == Down {
		g.Rise.Wait = spend(g.Rise.Wait)
	}

	if moved, step := g.Side.steps(moveEvery); step && g.holding(g.Side, in, Left, Right) {
		next := g.Ship + moved.Way
		if next < 0 || next > g.Field.ShipColMax() {
			// A wall is a stop. Leaving it pressed against one would mean the
			// next thing you press is a key you did not know you had to press.
			g.Side = g.Side.stop()
		} else {
			g.Ship = next
			g.Side = moved
		}
	}
	if moved, step := g.Rise.steps(climbEvery); step && g.holding(g.Rise, in, Up, Down) {
		next := g.Row + moved.Way
		if next < g.Field.ShipRoof() || next > g.Field.ShipRow() {
			g.Rise = g.Rise.stop()
		} else {
			g.Row = next
			g.Rise = moved
		}
	}
	return g
}

// holding says whether an axis may move this tick: because a key for it just
// arrived, or because it is still inside the leash a held key left it.
func (g Game) holding(a Axis, in, low, high Key) bool {
	return in == low || in == high || a.Hold > 0
}

func spend(wait int) int {
	if wait > 0 {
		return wait - 1
	}
	return 0
}

// tickWeapon is the gun: a magazine, a cadence and a reload.
//
// The old rule was a cap on how many of your PRESSES could be in the air at
// once, which is the arcade's discipline. This is the reference's instead, and it
// is a better fit for a fleet: what limits you is how much you can shoot before
// you have to stand still and reload, not how far your last shot has travelled.
// Firing on empty starts the reload for you, so nobody loses a run to not having
// read the help row.
func (g Game) tickWeapon(in Key) Game {
	if g.Cool > 0 {
		g.Cool--
	}
	if g.Loading > 0 {
		g.Loading--
		if g.Loading == 0 {
			g.Ammo = g.Kit.Cap
		}
		return g
	}
	if in == Rearm && g.Ammo < g.Kit.Cap {
		g.Loading = g.Kit.Reload
		return g
	}
	// Empty comes before the cadence: pressing fire on an empty magazine has to
	// start the reload even in the tick after a shot, or whether the gun reloads
	// itself depends on exactly when you pressed - which is unlearnable.
	if in == Fire && g.Ammo <= 0 && g.Rush == 0 {
		g.Loading = g.Kit.Reload
		return g
	}
	if in != Fire || g.Cool > 0 {
		return g
	}
	if g.Rush > 0 {
		// Free rounds and twice the rate: the rapid branch's four seconds of
		// not having to think about the magazine.
		g.Cool = max(g.Kit.Cadence/2, 1)
	} else {
		g.Ammo--
		g.Cool = g.Kit.Cadence
	}
	g = g.volley(g.Kit.Shots)
	if g.Mirror > 0 {
		g = g.ghostVolley()
	}
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
	if g.Rage > 0 {
		damage *= 2
	}
	return damage
}

// volley fires n shots, spread across the ship's own width.
func (g Game) volley(n int) Game { return g.volleyFrom(g.Ship, n) }

// volleyFrom is the same volley from any column, which is what the twin branch's
// second ship needs.
//
// It exists because the ghost used to fire its own way: the same number of shots,
// all from one column, stacked on top of each other. Three shots that look like
// one, beside a real ship firing three that look like three - "el espejo dispara
// 3, pero el bicho real solo 1", exactly backwards and exactly right.
func (g Game) volleyFrom(col, n int) Game {
	if n < 1 {
		n = 1
	}
	shots := append([]Shot{}, g.Shots...)
	top := float64(g.Row)
	for i := 0; i < n; i++ {
		x := float64(col) + float64(ShipCols)/2
		if n > 1 {
			span := float64(ShipCols - 3)
			x = float64(col) + 1 + span*float64(i)/float64(n-1)
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
	for _, t := range []*int{&g.Invuln, &g.Shield, &g.Rush, &g.Mirror, &g.Net, &g.Rage} {
		if *t > 0 {
			*t--
		}
	}
	if in != Ability || g.Ready > 0 {
		return g
	}
	g.Ready = g.Kit.Cooldown

	switch g.Kit.Special {
	case AbilitySweep:
		g = g.column(g.Ship, ShipCols, sweepDamage)
	case AbilityThree:
		g = g.column(g.Ship-ShipCols, 3*ShipCols, sweepDamage)
	case AbilityPulse:
		// The larva's, and the only one that hurts nothing: everything in the
		// air is shoved back up and every bomb burns. A way out rather than a
		// way through, which is what a larva needs.
		g = g.shove(pulseRows)
	case AbilityShield:
		g.Shield = shieldFor
	case AbilityMark:
		g = g.darts(3)
	case AbilityRush:
		g.Rush = rushFor
		g.Ammo = g.Kit.Cap
	case AbilityMirror:
		g.Mirror = mirrorFor
	case AbilityNet:
		g.Net = netFor
	case AbilityDash:
		g = g.dash()
	case AbilityLance:
		g = g.lance()
	case AbilityFrenzy:
		// Bought with a point of life, which is that branch's whole idea: the
		// less of you there is, the harder you hit.
		g.Rage = rageFor
		if g.HP > 1 {
			g.HP--
		}
	case AbilityTurret, AbilityTurret2:
		n := 1
		if g.Kit.Special == AbilityTurret2 {
			n = 2
		}
		turrets := append([]Turret{}, g.Turrets...)
		for i := 0; i < n; i++ {
			col := clamp(g.Ship+1+i*3, 0, g.Field.Cols-1)
			turrets = append(turrets, Turret{
				Col: col, Row: g.Row, Life: turretLife,
				Next: turretCadence, Damage: g.Kit.Damage,
			})
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

// tickHeal spends a kit out of the hold, and only when there is something to
// heal: a kit thrown away on full life is a kit somebody swears at.
func (g Game) tickHeal(in Key) Game {
	if in != Heal || g.Kits <= 0 || g.HP >= g.Kit.MaxHP {
		return g
	}
	g.Kits--
	g.HP += g.Kit.MaxHP / 3
	if g.HP < 1 {
		g.HP = 1
	}
	if g.HP > g.Kit.MaxHP {
		g.HP = g.Kit.MaxHP
	}
	g.Banner = BannerKit
	return g
}

// column hurts everything standing over a stretch of the field, which is what a
// beam fired straight up is.
func (g Game) column(from, width, damage int) Game {
	to := float64(from + width)
	left := float64(from)

	if g.Boss.Alive && to > g.Boss.X && left < g.Boss.X+BossCols {
		g.Boss.HP -= damage
		if g.Boss.HP <= 0 {
			g = g.killBoss()
		}
	}
	alive := make([]Alien, 0, len(g.Aliens))
	for _, a := range g.Aliens {
		if to > a.X && left < a.X+float64(a.Craft().W) {
			a.HP -= damage
			if a.HP <= 0 {
				g = g.killAlien(a)
				continue
			}
		}
		alive = append(alive, a)
	}
	g.Aliens = alive

	stones := make([]Stone, 0, len(g.Stones))
	for _, s := range g.Stones {
		if to > s.X && left < s.X+float64(Rock.W) {
			s.HP -= damage
			if s.HP <= 0 {
				g = g.breakStone(s)
				continue
			}
		}
		stones = append(stones, s)
	}
	g.Stones = stones
	return g
}

// shove pushes everything in the air back up the field and burns every bomb. It
// kills nothing, which is the point: it buys room.
func (g Game) shove(rows int) Game {
	aliens := make([]Alien, 0, len(g.Aliens))
	for _, a := range g.Aliens {
		a.Y -= float64(rows)
		if a.Y < 0 {
			a.Y = 0
		}
		aliens = append(aliens, a)
	}
	g.Aliens = aliens
	for _, b := range g.Bombs {
		g = g.burst(b.X, b.Y, 0)
	}
	g.Bombs = nil
	return g
}

// darts are the seeker's: three shots that home, spread wide enough to pick three
// different ships rather than pile onto one.
func (g Game) darts(n int) Game {
	shots := append([]Shot{}, g.Shots...)
	for i := 0; i < n; i++ {
		shots = append(shots, Shot{
			X: float64(g.Ship) + float64(i*(ShipCols-1))/float64(n-1),
			Y: float64(g.Row),
			// Twice a normal shot, homing whatever the kit does, and through one
			// body: a dart is not a volley.
			Damage: 2 * g.damageFor(),
			Pierce: 1,
			Homing: true,
		})
	}
	g.Shots = shots
	return g
}

// dash crosses the field in one frame, hurting everything the ship passes through
// on the way.
//
// The sprinter's, and the only ability that moves you: half the value of it is
// that it is also a way out.
func (g Game) dash() Game {
	from, to := g.Ship, g.Field.ShipColMax()-g.Ship
	low, high := min(from, to), max(from, to)
	aliens := make([]Alien, 0, len(g.Aliens))
	for _, a := range g.Aliens {
		c := a.Craft()
		across := a.X+float64(c.W) > float64(low) && a.X < float64(high+ShipCols)
		level := a.Y+float64(c.H) > float64(g.Row) && a.Y < float64(g.Row+ShipRows)
		if across && level {
			a.HP -= dashDamage
			if a.HP <= 0 {
				g = g.killAlien(a)
				continue
			}
		}
		aliens = append(aliens, a)
	}
	g.Aliens = aliens
	g.Ship = to
	g.Side = g.Side.stop()
	return g
}

// lance is one shot the width of the ship that goes through everything on the
// field: the cannon branch's, and the only thing in the game that can take a boss
// down in one press.
func (g Game) lance() Game {
	shots := append([]Shot{}, g.Shots...)
	for i := 1; i < ShipCols-1; i++ {
		shots = append(shots, Shot{
			X: float64(g.Ship + i), Y: float64(g.Row),
			Damage: lanceHit, Pierce: 99,
		})
	}
	g.Shots = shots
	return g
}

// ghostVolley is the twin branch's second ship firing alongside you: the same
// volley from the column it stands in, which is what makes the two look like two
// of the same ship.
func (g Game) ghostVolley() Game { return g.volleyFrom(g.ghostAt(), g.Kit.Shots) }

// ghostAt is the column the second ship stands in: the far side if there is room
// for it, and the near side if there is not.
func (g Game) ghostAt() int {
	if g.Ship+mirrorGap <= g.Field.ShipColMax() {
		return g.Ship + mirrorGap
	}
	return max(g.Ship-mirrorGap, 0)
}

// release lets the wave out a ship at a time, and drops the two things that
// arrive on their own clock.
func (g Game) release() Game {
	if g.RockIn > 0 {
		g.RockIn--
	} else {
		g.RockIn = rockEvery
		g.Stones = append(append([]Stone{}, g.Stones...), Stone{
			X:  float64(roll(&g.Rand, max(g.Field.Cols-Rock.W, 1))),
			Y:  0,
			Vx: Rock.Drift * pick(&g.Rand),
			HP: Rock.HP + g.Wave.Tough, MaxHP: Rock.HP + g.Wave.Tough,
		})
	}
	if g.KitIn > 0 {
		g.KitIn--
	} else {
		g.KitIn = kitEvery
		g.Drops = append(append([]Drop{}, g.Drops...), Drop{
			X: float64(roll(&g.Rand, max(g.Field.Cols-1, 1))), Y: 0,
		})
	}

	if g.Released >= g.Wave.Count {
		return g
	}
	if g.Next > 0 {
		g.Next--
		return g
	}
	g.Next = g.Wave.Every

	// Ones at first and twos or threes later: what makes a late wave hard is
	// how much arrives together, and it is a gentler axis than speed because
	// what you have to do about it - pick an order and shoot it - is the same
	// thing you were already doing.
	pool := g.Wave.Unlocked()
	aliens := append([]Alien{}, g.Aliens...)
	// A lane each, so a pack of three arrives spread across the width. Without
	// it two ships regularly spawned on the same columns and came down as one
	// unreadable smear of line art.
	lane := max(g.Field.Cols/max(g.Wave.Pack, 1), 1)
	for i := 0; i < g.Wave.Pack && g.Released < g.Wave.Count; i++ {
		of := pool[roll(&g.Rand, len(pool))]
		c := Fleet[of]
		g.Released++
		aliens = append(aliens, Alien{
			Of: of,
			X:  float64(clamp(lane*i+roll(&g.Rand, max(lane-c.W, 1)), 0, max(g.Field.Cols-c.W, 0))),
			Y:  0,
			Vx: c.Drift * g.Wave.Haste * pick(&g.Rand),
			HP: c.HP + g.Wave.Tough, MaxHP: c.HP + g.Wave.Tough,
			Fire: c.Cadence,
		})
	}
	g.Aliens = aliens
	return g
}

// pick is -1 or 1: which way something that has just arrived is drifting.
func pick(rand *uint64) float64 {
	if roll(rand, 2) == 0 {
		return -1
	}
	return 1
}

func (g Game) moveStars() Game {
	stars := make([]Star, 0, len(g.Stars))
	for _, s := range g.Stars {
		s.Y += s.V
		if s.Y >= float64(g.Field.Rows) {
			s.Y = 0
			s.X = float64(roll(&g.Rand, max(g.Field.Cols, 1)))
		}
		stars = append(stars, s)
	}
	g.Stars = stars
	return g
}

func (g Game) moveShots() Game {
	next := make([]Shot, 0, len(g.Shots))
	target, found := g.nearest()
	for _, s := range g.Shots {
		if s.Homing && found && g.Frame%2 == 0 {
			if target < s.X-0.5 {
				s.X--
			} else if target > s.X+0.5 {
				s.X++
			}
		}
		s.Y -= shotSpeed
		if s.Y < 0 {
			continue
		}
		next = append(next, s)
	}
	g.Shots = next
	return g
}

// nearest is the column a homing shot leans towards: the lowest thing in the
// air, because that is what is about to reach the floor.
func (g Game) nearest() (float64, bool) {
	best, low, found := 0.0, -1.0, false
	if g.Boss.Alive {
		best, low, found = g.Boss.X+BossCols/2, g.Boss.Y, true
	}
	for _, a := range g.Aliens {
		if !found || a.Y > low {
			best, low, found = a.X+float64(a.Craft().W)/2, a.Y, true
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
			shots = append(shots, Shot{X: float64(t.Col), Y: float64(t.Row), Damage: t.Damage})
		}
		next = append(next, t)
	}
	g.Turrets = next
	g.Shots = shots
	return g
}

// moveAliens falls, drifts, bounces off the walls, fires, and works out what
// happens to the ones that reach the floor.
func (g Game) moveAliens() Game {
	alive := make([]Alien, 0, len(g.Aliens))
	bombs := append([]Bomb{}, g.Bombs...)

	for _, a := range g.Aliens {
		c := a.Craft()
		if g.Net == 0 {
			// The homing branch's net stops the descent and nothing else: they
			// still drift and they still shoot, so it is time bought rather
			// than a pause button.
			a.Y += c.Fall * g.Wave.Haste
		}
		a.X += a.Vx
		if a.X < 0 {
			a.X, a.Vx = 0, -a.Vx
		}
		if right := float64(g.Field.Cols - c.W); a.X > right {
			a.X, a.Vx = right, -a.Vx
		}

		if a.Fire > 0 {
			a.Fire--
		} else {
			a.Fire = c.Cadence
			bombs = append(bombs, Bomb{
				X: a.X + float64(c.W)/2, Y: a.Y + float64(c.H), Hurt: bombDrop,
			})
		}

		// Running into the ship costs more than slipping past it, and both of
		// them are gone. The ram is checked against where the ship IS rather
		// than against the floor, because the ship no longer lives on the floor:
		// climbing into something is a way to get hurt now, and it should be.
		if g.rams(a) {
			g = g.wound(ramDrop)
			g = g.burst(a.X+float64(c.W)/2, a.Y, 0)
			continue
		}
		if a.Y+float64(c.H) >= float64(g.Field.Rows) {
			g = g.wound(landDrop)
			g = g.burst(a.X+float64(c.W)/2, a.Y, 0)
			continue
		}
		alive = append(alive, a)
	}
	g.Aliens = separate(alive, g.Field)
	g.Bombs = bombs
	return g
}

// away is a drift's speed, pointing right. A ship with no sideways speed at all
// still has to leave, or a collision with it is a wall.
func away(vx float64) float64 {
	if vx < 0 {
		vx = -vx
	}
	if vx < bounceLeast {
		return bounceLeast
	}
	return vx
}

// separate pushes two ships apart when they have drifted into each other.
//
// They fly on their own clocks and nothing stops two of them arriving at the same
// columns, and line art on top of line art is not two ships that overlap - it is
// one shape nobody can read. A frame with a tejedora inside an avispa is what
// sent this in.
//
// Cheap on purpose: one pass, the pair swaps direction and the lower one is
// nudged clear. It is not a physics engine and it does not have to be, because
// the drift is what carries them apart a tick later.
func separate(aliens []Alien, f Field) []Alien {
	for i := range aliens {
		for j := i + 1; j < len(aliens); j++ {
			a, b := aliens[i], aliens[j]
			aw, bw := float64(a.Craft().W), float64(b.Craft().W)
			ah, bh := float64(a.Craft().H), float64(b.Craft().H)
			if a.X+aw <= b.X || b.X+bw <= a.X {
				continue
			}
			if a.Y+ah <= b.Y || b.Y+bh <= a.Y {
				continue
			}
			// Each one leaves the way it came from: the one on the left goes
			// left and the one on the right goes right, whatever they were
			// doing before.
			//
			// The first version only turned one of them round when both
			// happened to be going the same way, which left the other pair of
			// cases jittering against each other for a second or two - reported
			// as "un efecto de rebote raro", and it was: two ships taking turns
			// to push each other rather than one collision and done.
			leftmost, rightmost := i, j
			if aliens[j].X < aliens[i].X {
				leftmost, rightmost = j, i
			}
			aliens[leftmost].X -= bounceStep
			aliens[rightmost].X += bounceStep
			aliens[leftmost].Vx = -away(aliens[leftmost].Vx)
			aliens[rightmost].Vx = away(aliens[rightmost].Vx)
			// The nudge is still inside the walls. Without this the pair by the
			// left edge walked each other off the field, which the autopilot's
			// invariant sweep caught at tick 6082 of a run.
			for _, k := range [2]int{i, j} {
				aliens[k].X = clampf(aliens[k].X, 0,
					float64(max(f.Cols-aliens[k].Craft().W, 0)))
			}
		}
	}
	return aliens
}

func (g Game) moveBoss() Game {
	if !g.Boss.Alive {
		return g
	}
	speed := 0.125 + 0.175*float64(g.Boss.MaxHP-g.Boss.HP)/float64(max(g.Boss.MaxHP, 1))
	g.Boss.X += speed * float64(g.Boss.Dir)
	if g.Boss.X < 0 {
		g.Boss.X, g.Boss.Dir = 0, 1
	}
	if g.Boss.X > float64(g.Field.Cols-BossCols) {
		g.Boss.X, g.Boss.Dir = float64(g.Field.Cols-BossCols), -1
	}
	// It leans down as it is worn, so a long fight is a closing one.
	if g.Frame%180 == 0 && int(g.Boss.Y)+BossRows < g.Field.Rows-1 {
		g.Boss.Y++
	}
	if g.Boss.Fire > 0 {
		g.Boss.Fire--
		return g
	}
	g.Boss.Fire = clamp(80-g.Wave.N, 16, 120)
	bombs := append([]Bomb{}, g.Bombs...)
	for _, dx := range [3]float64{1, BossCols / 2, BossCols - 2} {
		bombs = append(bombs, Bomb{X: g.Boss.X + dx, Y: g.Boss.Y + BossRows, Hurt: bossDrop})
	}
	g.Bombs = bombs
	return g
}

func (g Game) moveStones() Game {
	next := make([]Stone, 0, len(g.Stones))
	for _, s := range g.Stones {
		s.Y += Rock.Fall
		s.X += s.Vx
		if s.X < 0 {
			s.X, s.Vx = 0, -s.Vx
		}
		if right := float64(g.Field.Cols - Rock.W); s.X > right {
			s.X, s.Vx = right, -s.Vx
		}
		hit := s.X+float64(Rock.W) > float64(g.Ship) && s.X < float64(g.Ship+ShipCols) &&
			s.Y+float64(Rock.H) > float64(g.Row) && s.Y < float64(g.Row+ShipRows)
		if hit {
			g = g.wound(ramDrop)
			g = g.breakStone(s)
			continue
		}
		if s.Y+float64(Rock.H) >= float64(g.Field.Rows) {
			g = g.breakStone(s)
			continue
		}
		next = append(next, s)
	}
	g.Stones = next
	return g
}

// moveMotes walks the sparks and the meteoroids. A spark is decoration; a
// meteoroid hurts the first thing it touches, whoever's side it is on.
func (g Game) moveMotes() Game {
	next := make([]Mote, 0, len(g.Motes))
	for _, m := range g.Motes {
		m.Life--
		if m.Life <= 0 {
			continue
		}
		m.X += m.Vx
		m.Y += m.Vy
		if m.X < 0 || m.X >= float64(g.Field.Cols) || m.Y < 0 || m.Y >= float64(g.Field.Rows) {
			continue
		}
		if m.Hurt > 0 {
			if hit, ok := g.alienAt(m.X, m.Y); ok {
				g = g.hurtAlien(hit, m.Hurt)
				continue
			}
			if g.hitsShip(m.X, m.Y) {
				g = g.wound(m.Hurt)
				continue
			}
		}
		next = append(next, m)
	}
	g.Motes = next
	return g
}

func (g Game) moveDrops() Game {
	next := make([]Drop, 0, len(g.Drops))
	for _, d := range g.Drops {
		d.Y += 0.06
		if d.Y >= float64(g.Field.Rows) {
			continue
		}
		if g.hitsShip(d.X, d.Y) {
			if g.Kits < kitsMax {
				g.Kits++
			}
			continue
		}
		next = append(next, d)
	}
	g.Drops = next
	return g
}

func (g Game) moveBombs() Game {
	next := make([]Bomb, 0, len(g.Bombs))
	for _, b := range g.Bombs {
		b.Y += bombSpeed
		if b.Y >= float64(g.Field.Rows) {
			continue
		}
		if g.Shield > 0 && b.Y >= float64(g.Row-1) && b.Y < float64(g.Row+ShipRows) &&
			b.X >= float64(g.Ship-1) && b.X <= float64(g.Ship+ShipCols) {
			// The steady branch's shield: bombs burn on the way in rather than
			// passing through you. A wall and not invulnerability - a ship that
			// lands on you still lands on you.
			g = g.burst(b.X, b.Y, 0)
			continue
		}
		if g.hitsShip(b.X, b.Y) {
			g = g.wound(b.Hurt)
			g = g.burst(b.X, b.Y, 0)
			continue
		}
		next = append(next, b)
	}
	g.Bombs = next
	return g
}

// hitsShip is the ship's hitbox: the whole middle row, and only the three
// middle cells of the crest and the tail.
//
// The corners are cosmetic on purpose. A five-cell box would mean the tips of
// the antennae kill you, which is unreadable in a moving frame - the same rule
// the creature's own crest and feet had when it was the ship.
func (g Game) hitsShip(x, y float64) bool {
	col, row := int(x), int(y)
	if col < g.Ship || col >= g.Ship+ShipCols {
		return false
	}
	switch row - g.Row {
	case 1:
		return true
	case 0, 2:
		return col >= g.Ship+1 && col <= g.Ship+ShipCols-2
	}
	return false
}

// rams says whether a ship has flown into the player's, box against box.
func (g Game) rams(a Alien) bool {
	c := a.Craft()
	return a.X+float64(c.W) > float64(g.Ship) && a.X < float64(g.Ship+ShipCols) &&
		a.Y+float64(c.H) > float64(g.Row) && a.Y < float64(g.Row+ShipRows)
}

// alienAt is the index of whatever is in the air at a point, if anything.
func (g Game) alienAt(x, y float64) (int, bool) {
	for i, a := range g.Aliens {
		c := a.Craft()
		if x >= a.X && x < a.X+float64(c.W) && y >= a.Y && y < a.Y+float64(c.H) {
			return i, true
		}
	}
	return 0, false
}

// hurtAlien takes hit points off one of them by index, and scores it if that was
// the last of them.
func (g Game) hurtAlien(i, damage int) Game {
	if i < 0 || i >= len(g.Aliens) {
		return g
	}
	aliens := append([]Alien{}, g.Aliens...)
	aliens[i].HP -= damage
	if aliens[i].HP > 0 {
		g.Aliens = aliens
		return g
	}
	dead := aliens[i]
	g.Aliens = append(aliens[:i:i], aliens[i+1:]...)
	return g.killAlien(dead)
}

// resolveHits walks your shots against the fleet, the boss and the rocks.
func (g Game) resolveHits() Game {
	if len(g.Shots) == 0 {
		return g
	}
	aliens := append([]Alien{}, g.Aliens...)
	stones := append([]Stone{}, g.Stones...)

	shots := make([]Shot, 0, len(g.Shots))
	for _, s := range g.Shots {
		alive := true

		if g.Boss.Alive && g.hitsBoss(s) {
			g.Boss.HP -= s.Damage
			g = g.burst(s.X, s.Y, 0)
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

		for i := range aliens {
			if aliens[i].HP <= 0 {
				continue
			}
			c := aliens[i].Craft()
			if s.X < aliens[i].X || s.X >= aliens[i].X+float64(c.W) {
				continue
			}
			if s.Y < aliens[i].Y || s.Y >= aliens[i].Y+float64(c.H) {
				continue
			}
			aliens[i].HP -= s.Damage
			g = g.burst(s.X, s.Y, 0)
			if s.Splash > 0 {
				for j := range aliens {
					if j == i || aliens[j].HP <= 0 {
						continue
					}
					if absf(aliens[j].X-aliens[i].X) <= float64(s.Splash)+float64(c.W) &&
						absf(aliens[j].Y-aliens[i].Y) <= float64(s.Splash) {
						aliens[j].HP -= s.Damage
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
			for i := range stones {
				if stones[i].HP <= 0 {
					continue
				}
				if s.X < stones[i].X || s.X >= stones[i].X+float64(Rock.W) {
					continue
				}
				if s.Y < stones[i].Y || s.Y >= stones[i].Y+float64(Rock.H) {
					continue
				}
				stones[i].HP -= s.Damage
				s.Hit++
				if s.Hit > s.Pierce {
					alive = false
				}
				break
			}
		}

		if alive {
			shots = append(shots, s)
		}
	}
	g.Shots = shots

	kept := make([]Alien, 0, len(aliens))
	for _, a := range aliens {
		if a.HP > 0 {
			kept = append(kept, a)
			continue
		}
		g = g.killAlien(a)
	}
	g.Aliens = kept

	rocks := make([]Stone, 0, len(stones))
	for _, s := range stones {
		if s.HP > 0 {
			rocks = append(rocks, s)
			continue
		}
		g = g.breakStone(s)
	}
	g.Stones = rocks
	return g
}

func (g Game) hitsBoss(s Shot) bool {
	return s.Y >= g.Boss.Y && s.Y < g.Boss.Y+BossRows &&
		s.X >= g.Boss.X && s.X < g.Boss.X+BossCols
}

// burst is an explosion: sparks when hurt is zero, meteoroids when it is not.
func (g Game) burst(x, y float64, hurt int) Game {
	n, life := 5, sparkLife
	if hurt > 0 {
		n, life = 6, moteLife
	}
	motes := append([]Mote{}, g.Motes...)
	for i := 0; i < n; i++ {
		motes = append(motes, Mote{
			X: x, Y: y,
			Vx:   (float64(roll(&g.Rand, 9)) - 4) / 20,
			Vy:   (float64(roll(&g.Rand, 9)) - 4) / 20,
			Life: life - roll(&g.Rand, 3),
			Hurt: hurt,
		})
	}
	g.Motes = motes
	return g
}

// killAlien scores one of the fleet: its own value, multiplied by how deep the
// stage is, so the same drone is worth more in a later wave.
func (g Game) killAlien(a Alien) Game {
	g.Kills++
	g.Score += a.Craft().Points * g.Wave.Stage
	return g.burst(a.X+float64(a.Craft().W)/2, a.Y+float64(a.Craft().H)/2, 0)
}

// breakStone throws the meteoroids. It scores nothing: a rock is not a kill, and
// paying for one would make the safest way to farm the game standing still and
// shooting stones.
func (g Game) breakStone(s Stone) Game {
	return g.burst(s.X+float64(Rock.W)/2, s.Y+float64(Rock.H)/2, 1)
}

func (g Game) killBoss() Game {
	g.Kills++
	g.Boss.Alive = false
	g.Boss.HP = 0
	// The arrival banner goes with it. Leaving it up meant a dead boss was still
	// being announced as coming down, for the rest of the wave.
	if g.Banner == BannerBoss {
		g.Banner = ""
	}
	g.Score += Ranks[Bosses[clamp(g.Boss.Of, 0, len(Bosses)-1)].Rank-1].Points
	return g.burst(g.Boss.X+BossCols/2, g.Boss.Y+BossRows/2, 0)
}

// wound takes life, unless the mole's ability is up.
func (g Game) wound(hp int) Game {
	if g.Invuln > 0 || g.Phase == Over {
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

// bookkeep closes a wave when everything it held is gone, and ends the run when
// a boss reaches the floor.
func (g Game) bookkeep() Game {
	if g.Phase != Playing {
		return g
	}

	// A boss landing is over, whatever life you had left. The fleet's own ships
	// only cost life when they get through - it is a wave of individuals and
	// letting one past should not be the end of a run - but a boss coming down
	// on top of you is the fight lost.
	if g.Boss.Alive && int(g.Boss.Y)+BossRows >= g.Field.Rows {
		g.HP = 0
		g.Phase = Over
		g.Banner = BannerLanded
		return g
	}

	if g.Released < g.Wave.Count || len(g.Aliens) > 0 || g.Boss.Alive {
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
	g.Motes = nil
	return g
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
