package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// What a form flies with.

// Kit is the ship. Three layers, the same shape as the tree that produced it:
// the family anchor off the lineage gives the weapon, the mark modifies it, the
// level scales it. Forty-one forms therefore play differently without inventing
// forty-one unrelated mechanics.
//
// Every field is comparable on purpose - see TestAKitIsUsableAsAMapKey - because
// the proof that no two forms fly the same way is a map keyed on the mechanical
// half of this struct.
type Kit struct {
	Family string // the family id; its name comes from i18n.G().Families
	Trait  string // the mark or title that modified it, "" for a bare trade

	Cadence int // ticks between volleys; lower is faster
	Damage  int
	Shots   int  // projectiles per volley
	Pierce  int  // enemies a shot passes through
	Homing  bool // shots track the nearest enemy
	Splash  int  // rows splashed on a hit
	Spike   int  // percent chance a shot does double damage

	// Overload is the feral branch's passive: damage climbs as HP falls. Revive
	// is the phoenix's: one second life per run. Flags and not Specials because
	// neither is what the space bar does, and a form has to have both a passive
	// and an ability or its ability key does nothing.
	Overload bool
	Revive   bool

	Special  string // what the ability does
	Cooldown int    // ticks before the ability is ready again

	MaxHP int
	Regen int // hp per wave cleared

	// The magazine, which is the reference's discipline and now ours. A press
	// spends one round whatever the volley is, so a wide kit empties as fast as
	// a narrow one and the cost of missing is the reload rather than a cap on
	// what may be in the air.
	//
	// Both are DERIVED in scale rather than written per family: they follow the
	// volley width and the level, so they add nothing to the job of keeping
	// forty-one forms apart while still being part of what makes them feel
	// different.
	Cap    int // rounds per magazine
	Reload int // ticks to fill it
}

// minCadence is the fastest any gun may fire: one shot every four ticks, which
// is ten a second. Any lower and a magazine is gone before a finger has left the
// key.
const minCadence = 4

// The abilities. Every kit has one, because the space bar is the only thing the
// player times and a form that ignores it is a form that plays itself.
const (
	AbilityVolley  = "volley"   // one free triple volley
	AbilitySweep   = "sweep"    // a beam that clears a whole row
	AbilityTurret  = "turret"   // drops a turret that fires on its own
	AbilityTurret2 = "turret2"  // two of them
	AbilityInvuln  = "invuln"   // brief invulnerability
	AbilityThree   = "threerow" // strikes three rows at once
	AbilityBlast   = "blast"    // the phoenix: everything on screen, and a long wait
	AbilityChimera = "chimera"  // a volley and a sweep at once, being two things
)

// families is one weapon per family, keyed by the ANCHOR form it hangs off: the
// rung-3 trade, or the shallower form itself when the lineage is shorter than
// that. Values are the kit at level zero, before any mark or scaling.
//
// No base cadence is under 8 ticks, which is not decoration: the level ladder
// multiplies cadence rather than subtracting from it, so the floor below never
// engages and two families that differ by a tick still differ at level 6. The
// first draft subtracted, and a bolt at level 6 came out identical to a bare
// sprinter because both had hit the floor.
var families = map[string]Kit{
	pet.Root:    {Family: "single", Cadence: 28, Damage: 1, Shots: 1, Special: AbilityVolley, Cooldown: 200, MaxHP: 8, Regen: 1},
	"pattern":   {Family: "steady", Cadence: 36, Damage: 3, Shots: 1, Special: AbilityVolley, Cooldown: 200, MaxHP: 10, Regen: 1},
	"probe":     {Family: "seeker", Cadence: 30, Damage: 1, Shots: 1, Homing: true, Special: AbilityVolley, Cooldown: 200, MaxHP: 9, Regen: 1},
	"ember":     {Family: "rapid", Cadence: 18, Damage: 1, Shots: 1, Special: AbilityVolley, Cooldown: 200, MaxHP: 8, Regen: 1},
	"refactor":  {Family: "twin", Cadence: 32, Damage: 2, Shots: 2, Special: AbilityVolley, Cooldown: 200, MaxHP: 10, Regen: 1},
	"tidy":      {Family: "sweep", Cadence: 34, Damage: 2, Shots: 1, Special: AbilitySweep, Cooldown: 240, MaxHP: 11, Regen: 2},
	"bughunter": {Family: "homing", Cadence: 30, Damage: 2, Shots: 1, Homing: true, Special: AbilityVolley, Cooldown: 200, MaxHP: 10, Regen: 1},
	"architect": {Family: "turret", Cadence: 34, Damage: 2, Shots: 1, Special: AbilityTurret, Cooldown: 280, MaxHP: 11, Regen: 1},
	"sprinter":  {Family: "burst", Cadence: 16, Damage: 1, Shots: 2, Special: AbilityVolley, Cooldown: 180, MaxHP: 8, Regen: 1},
	"marathon":  {Family: "cannon", Cadence: 40, Damage: 3, Shots: 1, Pierce: 3, Special: AbilityVolley, Cooldown: 220, MaxHP: 12, Regen: 2},
	"feral":     {Family: "overload", Cadence: 24, Damage: 2, Shots: 1, Overload: true, Special: AbilityVolley, Cooldown: 200, MaxHP: 9},

	// The two that are not on the tree. A phoenix gets the revival its name is
	// for; a chimera carries both of its parents' families at once, which is
	// exactly what a chimera is - and since it has no Parent entry the pair is
	// written down here rather than read off the pet: twin, from the most
	// disciplined branch, welded to overload from the least.
	"phoenix": {Family: "phoenix", Cadence: 26, Damage: 3, Shots: 2, Pierce: 1, Homing: true, Revive: true, Special: AbilityBlast, Cooldown: 400, MaxHP: 10, Regen: 2},
	"chimera": {Family: "chimera", Cadence: 28, Damage: 3, Shots: 2, Overload: true, Special: AbilityChimera, Cooldown: 180, MaxHP: 10, Regen: 1},
}

// marks is one modifier per mark, applied to whatever family it hangs off. The
// keys are exactly the keys of pet.Unlocks, which is what
// TestEveryMarkOnTheTreeModifiesSomething checks.
//
// Two of them cannot be read literally off the design, because the design's own
// table gives them a property their family already has, which would make them
// play identically to a bare trade:
//
//   - bloodhound adds homing to the bughunter, which already homes. So it takes
//     it further: a bloodhound's nose does not stop at the first body.
//   - cartographer adds a turret ability to the architect, whose ability already
//     is a turret. So it drops two.
var marks = map[string]func(Kit) Kit{
	"surgeon":      func(k Kit) Kit { k.Damage++; return k },
	"weaver":       func(k Kit) Kit { k.Shots++; return k },
	"monk":         func(k Kit) Kit { k.Regen += 2; return k },
	"gardener":     func(k Kit) Kit { k.MaxHP += 4; k.Regen++; return k },
	"bloodhound":   func(k Kit) Kit { k.Homing = true; k.Pierce++; return k },
	"exterminator": func(k Kit) Kit { k.Splash++; return k },
	"cartographer": func(k Kit) Kit { k.Special = AbilityTurret2; return k },
	"oracle":       func(k Kit) Kit { k.Cooldown = k.Cooldown * 7 / 10; return k },
	"bolt":         func(k Kit) Kit { k.Cadence = k.Cadence * 7 / 10; return k },
	"sniper":       func(k Kit) Kit { k.Pierce += 2; return k },
	"ox":           func(k Kit) Kit { k.MaxHP = k.MaxHP * 3 / 2; return k },
	"mole":         func(k Kit) Kit { k.Special = AbilityInvuln; return k },
	// The gremlin's "random damage spikes" had no field in the design's Kit at
	// all, which is the third way its tables collided: without Spike a gremlin
	// was a bare feral.
	"gremlin": func(k Kit) Kit { k.Spike = 25; return k },
	"kraken":  func(k Kit) Kit { k.Special = AbilityThree; return k },
}

// KitFor resolves a form and a level to a kit.
//
// Pure in its two arguments, which is what makes the forty-one-way distinctness
// test mean anything and what lets a boss wear a rival form's weapon without
// reading anybody's pet.json. An unknown form flies like the larva, the same
// fallback pet.Draw makes for an unknown sprite.
func KitFor(form string, level int) Kit {
	if level < 1 {
		level = 1
	}
	if top := len(pet.Levels); level > top {
		level = top
	}

	base, ok := families[anchor(form)]
	if !ok {
		base = families[pet.Root]
	}

	k := base
	if mark, title := markOf(form); mark != "" {
		k = marks[mark](k)
		k.Trait = mark
		if title {
			k = tierUp(k)
			k.Trait = form
		}
	}
	return scale(k, level)
}

// anchor is the family a form hangs off: the rung-3 trade, or the shallowest
// form there is when the lineage does not reach that far.
//
// Built on pet.Lineage rather than on pet's own tradeOf, which is unexported and
// would be the wrong answer anyway: it says "" for the root and the three rung-2
// forms, and all four of them have a family here.
func anchor(form string) string {
	for _, secret := range pet.Secrets {
		if form == secret {
			return form
		}
	}
	line := pet.Lineage(form)
	if len(line) == 0 {
		return pet.Root
	}
	i := 2
	if last := len(line) - 1; i > last {
		i = last
	}
	return line[i]
}

// markOf is the mark a form carries and whether the form is its title. A bare
// trade, the root, a rung-2 form and both secrets carry none.
func markOf(form string) (mark string, title bool) {
	if _, isMark := pet.Unlocks[form]; isMark {
		return form, false
	}
	if parent, ok := pet.Parent[form]; ok {
		if _, isMark := pet.Unlocks[parent]; isMark {
			return parent, true
		}
	}
	return "", false
}

// tierUp is a title: its mark one tier up.
//
// The design says "every number x1.5 and the ability's cooldown halved", which
// cannot be read literally either - cadence is a number where LOWER is better,
// so multiplying it would hand the title a worse gun than its mark. Numbers
// that want to be big go up by half; the two that want to be small come down.
func tierUp(k Kit) Kit {
	k.Damage = up(k.Damage)
	k.Shots = up(k.Shots)
	k.Pierce = up(k.Pierce)
	k.Splash = up(k.Splash)
	k.Spike = up(k.Spike)
	k.MaxHP = up(k.MaxHP)
	k.Regen = up(k.Regen)
	k.Cadence = down(k.Cadence)
	k.Cooldown = k.Cooldown / 2
	return k
}

// up is n and a half, rounded so that 1 -> 2, 2 -> 3, 3 -> 5. Since every kit
// has at least one damage, a title always differs from its mark.
func up(n int) int { return (n*3 + 1) / 2 }

// down is n divided by a half more, the other side of up.
func down(n int) int {
	if out := (n*2 + 2) / 3; out > 0 {
		return out
	}
	return 1
}

// scale is the level ladder: the same form, stronger as the pet is.
//
// Cadence is multiplied rather than decremented so that the ratios between
// families survive to level 6 instead of everyone piling onto the same floor.
func scale(k Kit, level int) Kit {
	k.Damage += level
	k.Cadence = k.Cadence * (12 - level) / 12
	if k.Cadence < minCadence {
		k.Cadence = minCadence
	}
	if level >= 4 {
		k.Shots++
	}
	if level >= 6 {
		k.Shots++
	}
	k.MaxHP += 2 * level

	// The magazine, which the level buys along with everything else: more rounds
	// and a quicker reload.
	//
	// It does NOT depend on the volley width, and that was a bug for about ten
	// minutes: a press spends one round whatever it fires, so tying the capacity
	// to Shots made a level-four form - where Shots goes up - come out with a
	// SMALLER magazine than it had at level three, which is a level that makes
	// you weaker.
	k.Cap = 8 + level

	// The reload is six shots' worth of cadence, so a fast gun reloads fast.
	//
	// Flat was wrong and measurably so: at a flat sixty-four ticks a bolt - four
	// ticks a shot - spent two thirds of its life standing still reloading while
	// a marathon barely noticed. Two thirds of the whole magazine was the second
	// try and it wobbled: integer division made the ratio drift a point either
	// way between levels, which showed up as a level that fires slower over a
	// magazine than the level below it. Six times the cadence has neither
	// problem - it falls with the cadence and it does not depend on the
	// capacity, so more rounds is always more rounds.
	k.Reload = clamp(6*k.Cadence, 40, 140)
	return k
}

// shape is the mechanical half of a Kit: everything that changes how it plays
// and nothing that only changes what it is called. Distinctness is measured on
// this and not on the whole struct, or a family name would be enough to pass a
// form off as different from one it flies exactly like.
type shape struct {
	Cadence, Damage, Shots, Pierce, Splash, Spike, Cooldown, MaxHP, Regen int
	Cap, Reload                                                           int
	Homing, Overload, Revive                                              bool
	Special                                                               string
}

func (k Kit) shape() shape {
	return shape{
		Cadence: k.Cadence, Damage: k.Damage, Shots: k.Shots, Pierce: k.Pierce,
		Splash: k.Splash, Spike: k.Spike, Cooldown: k.Cooldown,
		MaxHP: k.MaxHP, Regen: k.Regen, Cap: k.Cap, Reload: k.Reload,
		Homing: k.Homing, Overload: k.Overload, Revive: k.Revive, Special: k.Special,
	}
}
