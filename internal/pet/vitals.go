package pet

import "github.com/kyros-software/claude-code-themes/internal/theme"

// Vital is one of the seven states of the here-and-now layer. One number from
// 0 to 100 picks it: how full THIS session's context window is - see
// ContextLoad. The state chooses eyes, feet and colour; it never touches the
// silhouette, which belongs to the progress layer.
type Vital struct {
	// Rank is the state's position, 0 for "fresh" to 6 for "k.o.". It is the
	// index into the evolution's colour ramp - see ramps.go - so the state
	// picks the step and the form picks the hue.
	Rank    int
	Cap     float64
	Label   string
	Colour  theme.Colour
	PaleEye theme.Colour
	DarkEye theme.Colour
	Eyes    [2]rune
	HeadUp  bool
	Walks   bool
	Sparkle bool
}

// Vitals in order. The first whose Cap the usage does not exceed wins.
var Vitals = []Vital{
	{0, 22, "fresh", theme.Ident, theme.PaleGreen, theme.DarkGreen, [2]rune{'>', '<'}, true, true, true},
	{1, 45, "lively", theme.Ident, theme.PaleGreen, theme.DarkGreen, [2]rune{'>', '<'}, true, true, false},
	{2, 63, "easy", theme.Path, theme.PaleTeal, theme.DarkTeal, [2]rune{'o', 'o'}, true, true, false},
	{3, 78, "sluggish", theme.Quota, theme.PaleBlue, theme.DarkBlue, [2]rune{'▬', '▬'}, true, false, false},
	{4, 89, "tired", theme.Quota, theme.PaleBlue, theme.DarkBlue, [2]rune{'_', '_'}, false, false, false},
	{5, 99.999, "drowning", theme.Drowned, theme.PaleIndigo, theme.DarkIndigo, [2]rune{'x', 'x'}, false, false, false},
	{6, 100, "k.o.", theme.Grey, theme.PaleGrey, theme.DarkGrey, [2]rune{'x', 'x'}, false, false, false},
}

// KO is the last state, and the only one with a door of its own.
var KO = Vitals[len(Vitals)-1]

// Intact are the states where the evolution keeps its own eyes. From "easy"
// downwards the state takes over, which is what lets you read the tiredness at
// a glance without reading the label.
func (v Vital) Intact() bool { return v.Label == "fresh" || v.Label == "lively" }

// StateFor returns the state matching a 0..100 usage figure.
//
// It used to carry a second argument and a special door - context alone at 100
// went straight to k.o. - because the figure it was handed was a WEIGHTED MEAN,
// and a mean of three numbers cannot reach 100 unless all three do. With ctx,
// 5h and 7d at 100, 90 and 90 the mean landed on 95 and the k.o. sprite never
// showed. What it is handed now reaches 100 on its own, so the door has
// nothing left to do.
func StateFor(usage float64) Vital {
	if usage != usage || usage < 0 { // NaN or below the floor
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	for _, v := range Vitals {
		if usage <= v.Cap {
			return v
		}
	}
	return KO
}

// ContextLoad is how full the window is, and it is the ONLY thing the state
// reads. A missing figure is a zero, which is what keeps an old CLI - it sends
// no percentage - from painting a creature nobody measured.
//
// This number is the session and nothing else, which is the whole point. The
// state is the here-and-now of THIS session: a full window is a Claude that
// answers slower and thicker, and that is a thing you feel while you work, in
// this terminal, in this conversation. The 5h and 7d quotas are not that. They
// belong to the ACCOUNT: every session open at once reads the same figure, so
// letting them in made two windows - one at 6% context and one at 64% - sit
// there both saying "cansada" off a 5h quota at 81. Two creatures with one
// mood, and neither of them was describing its own session.
//
// This is the third answer to the same question and the reasons for the other
// two still hold in their own place. A 50/30/20 weighted mean DILUTED: with the
// context full at 100 and the quotas at 20 and 10 it returned 58 and the pet
// sat in turquoise with no room left to work in. The tightest of the three
// necks fixed that and broke this. So the quotas keep the reading they always
// deserved - band 1 prints them as numbers, painted off this same ladder, so a
// 5h at 95 shows up in drowning indigo - and they no longer speak for a
// session they know nothing about.
//
// The Vitals thresholds are untouched by any of it and always were the right
// ones: 22/45/63/78/89 is the first version's quadratic comfort curve,
// vida = 100*(1-(neck/100)^2), solved for the neck at its 95/80/60/40/20 marks.
// The curve has survived three inputs intact.
func ContextLoad(pc *float64) float64 {
	if pc == nil || *pc != *pc { // absent, or NaN
		return 0
	}
	if *pc < 0 {
		return 0
	}
	if *pc > 100 {
		return 100
	}
	return *pc
}
