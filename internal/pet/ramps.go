package pet

import "github.com/kyros-software/claude-code-themes/internal/theme"

// The colour ramps, from the design canvas "Atlas de Formas y Estados".
//
// Colour belongs to the EVOLUTION, not to the state. Each branch owns a hue and
// holds it across all seven states; the state only picks which step of that ramp
// gets drawn. So a cazabugs is blue whether it is fresh or drowning, and a
// maraton is amber either way - which is what tells 41 silhouettes apart in the
// three rows the statusline has room for, where the crest does not fit. The
// canvas: "la marca de arriba y el numero de patas identifican la forma y no
// cambian nunca ... y baja el color por su rampa".
//
// Ten ramps for 41 forms - a mark and a title inherit their trade's - seven
// steps each, in Vitals order. The last step is nearly grey for all ten: at the
// bottom they all look equally dead, which is the point.

// Ramp is one branch's colours: seven body steps indexed by the state's rank,
// and the bright tone its eyes carry while they are still lit.
type Ramp struct {
	Body [7]theme.Colour
	Eye  theme.Colour
}

// Lit is how the eyes are painted at a given rank. They carry the branch's own
// bright tone down to "tired" and then go out - not by changing glyph, which is
// the state's other signal, but by sinking into the body's own ramp two steps
// back. The canvas draws exactly that: from "ahogada" the eyes take Body[2],
// and at k.o. Body[3].
func (r Ramp) Lit(rank int) theme.Colour {
	switch {
	case rank >= 6:
		return r.Body[3]
	case rank == 5:
		return r.Body[2]
	}
	return r.Eye
}

// Ramps, by the name of the form that heads the branch.
var Ramps = map[string]Ramp{
	"spark": {
		Body: [7]theme.Colour{
			theme.Hex("#c3ccd6"), // fresca
			theme.Hex("#8891a0"), // vibrante
			theme.Hex("#75808e"), // a gusto
			theme.Hex("#5f6a78"), // espesa
			theme.Hex("#4b5563"), // cansada
			theme.Hex("#3d4652"), // ahogada
			theme.Hex("#39404a"), // k.o.
		},
		Eye: theme.Hex("#e8eef5"),
	},
	"pattern": {
		Body: [7]theme.Colour{
			theme.Hex("#7ff0dd"), // fresca
			theme.Hex("#4dd6c1"), // vibrante
			theme.Hex("#3bb8a6"), // a gusto
			theme.Hex("#2f9489"), // espesa
			theme.Hex("#27736e"), // cansada
			theme.Hex("#1f5a5c"), // ahogada
			theme.Hex("#3d4a4c"), // k.o.
		},
		Eye: theme.Hex("#d6fffa"),
	},
	"probe": {
		Body: [7]theme.Colour{
			theme.Hex("#a5d4ff"), // fresca
			theme.Hex("#6fb6ff"), // vibrante
			theme.Hex("#5495e0"), // a gusto
			theme.Hex("#4276bd"), // espesa
			theme.Hex("#345c96"), // cansada
			theme.Hex("#2a4875"), // ahogada
			theme.Hex("#414a55"), // k.o.
		},
		Eye: theme.Hex("#d5e9ff"),
	},
	"ember": {
		Body: [7]theme.Colour{
			theme.Hex("#ffc48f"), // fresca
			theme.Hex("#f2a35e"), // vibrante
			theme.Hex("#d1854a"), // a gusto
			theme.Hex("#a96a3b"), // espesa
			theme.Hex("#85522f"), // cansada
			theme.Hex("#6b4126"), // ahogada
			theme.Hex("#4d4437"), // k.o.
		},
		Eye: theme.Hex("#ffe3c9"),
	},
	"tidy": {
		Body: [7]theme.Colour{
			theme.Hex("#8ff5ae"), // fresca
			theme.Hex("#57e389"), // vibrante
			theme.Hex("#45bf71"), // a gusto
			theme.Hex("#38995c"), // espesa
			theme.Hex("#2d7549"), // cansada
			theme.Hex("#245c3d"), // ahogada
			theme.Hex("#3f4a44"), // k.o.
		},
		Eye: theme.Hex("#d8ffe9"),
	},
	"architect": {
		Body: [7]theme.Colour{
			theme.Hex("#b8c2ff"), // fresca
			theme.Hex("#8b9cff"), // vibrante
			theme.Hex("#6f7fe0"), // a gusto
			theme.Hex("#5a67bd"), // espesa
			theme.Hex("#474f96"), // cansada
			theme.Hex("#3a4075"), // ahogada
			theme.Hex("#444656"), // k.o.
		},
		Eye: theme.Hex("#e2e6ff"),
	},
	"marathon": {
		Body: [7]theme.Colour{
			theme.Hex("#ffdf95"), // fresca
			theme.Hex("#e8c46a"), // vibrante
			theme.Hex("#c7a453"), // a gusto
			theme.Hex("#a28442"), // espesa
			theme.Hex("#7d6634"), // cansada
			theme.Hex("#64512a"), // ahogada
			theme.Hex("#4a4638"), // k.o.
		},
		Eye: theme.Hex("#fff4d6"),
	},
	"feral": {
		Body: [7]theme.Colour{
			theme.Hex("#ff9fa2"), // fresca
			theme.Hex("#f2777a"), // vibrante
			theme.Hex("#d15d61"), // a gusto
			theme.Hex("#a94a4e"), // espesa
			theme.Hex("#85393d"), // cansada
			theme.Hex("#6b2d31"), // ahogada
			theme.Hex("#4d3f41"), // k.o.
		},
		Eye: theme.Hex("#ffd9da"),
	},
	"phoenix": {
		Body: [7]theme.Colour{
			theme.Hex("#ffd9a0"), // fresca
			theme.Hex("#f2a35e"), // vibrante
			theme.Hex("#e07b45"), // a gusto
			theme.Hex("#c25a35"), // espesa
			theme.Hex("#96422a"), // cansada
			theme.Hex("#73301f"), // ahogada
			theme.Hex("#4d3a33"), // k.o.
		},
		Eye: theme.Hex("#fff4d6"),
	},
	"chimera": {
		Body: [7]theme.Colour{
			theme.Hex("#d6c2ff"), // fresca
			theme.Hex("#b07cf0"), // vibrante
			theme.Hex("#9264cc"), // a gusto
			theme.Hex("#7650a8"), // espesa
			theme.Hex("#5c3d85"), // cansada
			theme.Hex("#48306b"), // ahogada
			theme.Hex("#413a52"), // k.o.
		},
		Eye: theme.Hex("#efe6ff"),
	},
}

// FormRamp maps each of the 41 forms to its branch's ramp.
var FormRamp = map[string]string{
	"spark":        "spark",
	"pattern":      "pattern",
	"probe":        "probe",
	"ember":        "ember",
	"refactor":     "pattern",
	"tidy":         "tidy",
	"bughunter":    "probe",
	"architect":    "architect",
	"sprinter":     "ember",
	"marathon":     "marathon",
	"feral":        "feral",
	"surgeon":      "pattern",
	"weaver":       "pattern",
	"monk":         "tidy",
	"gardener":     "tidy",
	"bloodhound":   "probe",
	"exterminator": "probe",
	"cartographer": "architect",
	"oracle":       "architect",
	"bolt":         "ember",
	"sniper":       "ember",
	"ox":           "marathon",
	"mole":         "marathon",
	"gremlin":      "feral",
	"kraken":       "feral",
	"scalpel":      "pattern",
	"loom":         "pattern",
	"abbot":        "tidy",
	"forest":       "tidy",
	"wolf":         "probe",
	"wasp":         "probe",
	"atlas":        "architect",
	"sphinx":       "architect",
	"storm":        "ember",
	"falcon":       "ember",
	"mammoth":      "marathon",
	"worm":         "marathon",
	"devil":        "feral",
	"leviathan":    "feral",
	"phoenix":      "phoenix",
	"chimera":      "chimera",
}

// RampOf is the ramp a form draws itself in. An unknown form falls back to the
// larva's grey, which is what something with no colour of its own looks like.
func RampOf(form string) Ramp {
	if r, ok := Ramps[FormRamp[form]]; ok {
		return r
	}
	return Ramps[Root]
}
