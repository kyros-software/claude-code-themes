package invaders

import "github.com/kyros-software/claude-code-themes/internal/pet"

// The player: a representation of the creature, not the creature.
//
// Up to here the cannon WAS the pet - pet.DrawCard, four rows of nine cells,
// crest and all. It is the best drawing in the repo and it was the wrong thing
// to fly: nine cells wide against a five-cell enemy is a target you cannot help
// hitting, it dwarfed everything else on the field, and it belongs to a painter
// that knows nothing about ships.
//
// So the creature is represented instead, in the line art the rest of the game
// is drawn in: three rows, five cells, a crest of antennae, a pair of eyes and a
// tail. What carries the identity is not the silhouette alone - it is three
// things at once, and each of them says something the player already knows:
//
//	the COLOUR is your form's own ramp, so a bughunter is the bughunter's green
//	the SILHOUETTE is your weapon's family, so the thirteen fly visibly apart
//	the EYES are your health, worn down the same way the creature's are
//
// It is smaller than what it replaces on purpose. The field is eighteen rows on
// the smallest terminal worth playing in, and the four the card took were four
// the enemies could not use.

// Hull is one family's ship: a crest, a body between two brackets and a tail.
//
// Five cells and three rows for every one of them. Held apart as pieces rather
// than as three finished rows because the middle row is assembled per frame -
// the eyes change with the health - and because a five-cell string with the
// eyes baked in is a string somebody edits to six cells one afternoon.
type Hull struct {
	Crest string // five cells
	Left  string // one
	Right string // one
	Tail  string // five
}

// Hulls is one per weapon family, and no two are alike: see
// TestNoTwoFamiliesFlyTheSameSilhouette. The families are the ones in kit.go,
// which are themselves the trade anchors off the pet's own tree, so the ship
// changes when the creature evolves into another branch and not before.
var Hulls = map[string]Hull{
	"single":   {Crest: "  ^  ", Left: "<", Right: ">", Tail: " /^\\ "},
	"steady":   {Crest: " -+- ", Left: "[", Right: "]", Tail: " /_\\ "},
	"seeker":   {Crest: " .^. ", Left: "<", Right: ">", Tail: " \\v/ "},
	"rapid":    {Crest: " ^^^ ", Left: "<", Right: ">", Tail: " /v\\ "},
	"twin":     {Crest: " ^ ^ ", Left: "[", Right: "]", Tail: " /^\\ "},
	"sweep":    {Crest: " --- ", Left: "<", Right: ">", Tail: " \\_/ "},
	"homing":   {Crest: " \\^/ ", Left: "<", Right: ">", Tail: " /^\\ "},
	"turret":   {Crest: " [^] ", Left: "|", Right: "|", Tail: " /|\\ "},
	"burst":    {Crest: " *^* ", Left: "<", Right: ">", Tail: " vvv "},
	"cannon":   {Crest: "  |  ", Left: "=", Right: "=", Tail: " /_\\ "},
	"overload": {Crest: " \\v/ ", Left: "{", Right: "}", Tail: " /^\\ "},
	"phoenix":  {Crest: " ^*^ ", Left: "<", Right: ">", Tail: " /*\\ "},
	"chimera":  {Crest: " ^-v ", Left: "{", Right: "]", Tail: " /v\\ "},
}

// eyes are the three cells in the middle of the hull, and they are the health.
//
// Three faces out of the pet's seven vital states rather than seven of them:
// at one cell an eye either is there, is half shut, or is out, and inventing
// four more would be four the player cannot tell apart in a moving frame.
var eyes = [3]string{"o o", "- -", "x x"}

// ShipArt is the three rows to draw, for a family at a state of health.
//
// A family nobody knows flies the larva's hull, which is the same fallback
// pet.Draw makes for a form nobody knows, and for the same reason: a run must
// never fail to draw because a table has a gap in it.
func ShipArt(family string, vital pet.Vital) [ShipRows]string {
	h, ok := Hulls[family]
	if !ok {
		h = Hulls["single"]
	}
	return [ShipRows]string{h.Crest, h.Left + eyesFor(vital) + h.Right, h.Tail}
}

// eyesFor splits the pet's seven ranks three ways: open, half shut, out. The
// last one is rank six, which is the state the creature is in when the run is
// over - so the ship dies with its eyes out, the way the creature lies down.
func eyesFor(vital pet.Vital) string {
	switch {
	case vital.Rank >= len(pet.Vitals)-1:
		return eyes[2]
	case vital.Rank >= 3:
		return eyes[1]
	default:
		return eyes[0]
	}
}
