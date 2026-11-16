package pet

import "github.com/kyros-software/claude-code-themes/internal/theme"

// The ramps for a light terminal.
//
// Same branches, same hues, the opposite physics. On black a creature's energy
// is LIGHT: "fresca" glows near-white and k.o. sinks into the dark footer. On
// white, energy is INK: "fresca" is the deepest tone of its hue and k.o. fades
// into the #f3f3f3 footer - at the bottom they all look equally washed out,
// which is the point the dark ramps make the other way round.
//
// Derived, not picked: each branch keeps the OKLCH hue of its dark "vibrante"
// (the phoenix leans to its own red, as it does in the dark), and the seven
// steps climb L 0.47 -> 0.82 while the chroma falls to a tenth. Measured on the
// footer, "fresca" is about 6:1, "vibrante" 5:1 and k.o. 1.6:1 - the dark k.o.
// is 1.9:1 against its own footer. The lit eye is L 0.36, about 10:1, so it
// still reads as the brightest thing in the face, here by being the darkest.
var lightRamps = map[string]Ramp{
	"spark": {
		Body: [7]theme.Colour{
			theme.Hex("#545b66"), // fresca
			theme.Hex("#626975"), // vibrante
			theme.Hex("#757b85"), // a gusto
			theme.Hex("#878c95"), // espesa
			theme.Hex("#9a9fa5"), // cansada
			theme.Hex("#aeb1b6"), // ahogada
			theme.Hex("#c3c4c5"), // k.o.
		},
		Eye: theme.Hex("#383d47"),
	},
	"pattern": {
		Body: [7]theme.Colour{
			theme.Hex("#036a5d"), // fresca
			theme.Hex("#087a6c"), // vibrante
			theme.Hex("#268c7d"), // a gusto
			theme.Hex("#529c8f"), // espesa
			theme.Hex("#74aba0"), // cansada
			theme.Hex("#93bbb2"), // ahogada
			theme.Hex("#bdc6c4"), // k.o.
		},
		Eye: theme.Hex("#00483f"),
	},
	"probe": {
		Body: [7]theme.Colour{
			theme.Hex("#225d96"), // fresca
			theme.Hex("#316ca6"), // vibrante
			theme.Hex("#4e7eb0"), // a gusto
			theme.Hex("#6890ba"), // espesa
			theme.Hex("#82a2c3"), // cansada
			theme.Hex("#9db4cd"), // ahogada
			theme.Hex("#bfc5cb"), // k.o.
		},
		Eye: theme.Hex("#053e6e"),
	},
	"ember": {
		Body: [7]theme.Colour{
			theme.Hex("#854a04"), // fresca
			theme.Hex("#965819"), // vibrante
			theme.Hex("#a36c3d"), // a gusto
			theme.Hex("#af815b"), // espesa
			theme.Hex("#bb9679"), // cansada
			theme.Hex("#c7ab95"), // ahogada
			theme.Hex("#cac3bd"), // k.o.
		},
		Eye: theme.Hex("#5c3101"),
	},
	"tidy": {
		Body: [7]theme.Colour{
			theme.Hex("#016d36"), // fresca
			theme.Hex("#047e3f"), // vibrante
			theme.Hex("#318f53"), // a gusto
			theme.Hex("#579e6c"), // espesa
			theme.Hex("#77ad85"), // cansada
			theme.Hex("#95bd9f"), // ahogada
			theme.Hex("#bdc7bf"), // k.o.
		},
		Eye: theme.Hex("#004a23"),
	},
	"architect": {
		Body: [7]theme.Colour{
			theme.Hex("#48529f"), // fresca
			theme.Hex("#5561af"), // vibrante
			theme.Hex("#6875b8"), // a gusto
			theme.Hex("#7c88c1"), // espesa
			theme.Hex("#929cc9"), // cansada
			theme.Hex("#a7afd1"), // ahogada
			theme.Hex("#c1c4cc"), // k.o.
		},
		Eye: theme.Hex("#2e3576"),
	},
	"marathon": {
		Body: [7]theme.Colour{
			theme.Hex("#715700"), // fresca
			theme.Hex("#826401"), // vibrante
			theme.Hex("#917732"), // a gusto
			theme.Hex("#a08a54"), // espesa
			theme.Hex("#ae9d73"), // cansada
			theme.Hex("#bdb092"), // ahogada
			theme.Hex("#c7c4bc"), // k.o.
		},
		Eye: theme.Hex("#4d3a01"),
	},
	"feral": {
		Body: [7]theme.Colour{
			theme.Hex("#96363b"), // fresca
			theme.Hex("#a64549"), // vibrante
			theme.Hex("#b25e5f"), // a gusto
			theme.Hex("#bd7675"), // espesa
			theme.Hex("#c78d8c"), // cansada
			theme.Hex("#d0a5a3"), // ahogada
			theme.Hex("#ccc1c1"), // k.o.
		},
		Eye: theme.Hex("#6d1a22"),
	},
	"phoenix": {
		Body: [7]theme.Colour{
			theme.Hex("#8b441c"), // fresca
			theme.Hex("#9b522b"), // vibrante
			theme.Hex("#a86848"), // a gusto
			theme.Hex("#b37e63"), // espesa
			theme.Hex("#bf937e"), // cansada
			theme.Hex("#caa99a"), // ahogada
			theme.Hex("#cac2be"), // k.o.
		},
		Eye: theme.Hex("#652800"),
	},
	"chimera": {
		Body: [7]theme.Colour{
			theme.Hex("#6c419c"), // fresca
			theme.Hex("#7a50ad"), // vibrante
			theme.Hex("#8966b6"), // a gusto
			theme.Hex("#987dbf"), // espesa
			theme.Hex("#a893c8"), // cansada
			theme.Hex("#b8a9d0"), // ahogada
			theme.Hex("#c6c2cc"), // k.o.
		},
		Eye: theme.Hex("#4c2574"),
	},
}

// darkRamps is the table Ramps starts as; Use swaps between the two.
var darkRamps = Ramps

// paint points Ramps and the state colours at the palette in use. Vitals holds
// its colours by value, so it has to be copied again whenever the palette
// changes - in a real run that is once, in init, before anything is drawn.
func paint() {
	if theme.Light() {
		Ramps = lightRamps
	} else {
		Ramps = darkRamps
	}
	cols := stateColours()
	for i := range Vitals {
		Vitals[i].Colour, Vitals[i].PaleEye, Vitals[i].DarkEye = cols[i][0], cols[i][1], cols[i][2]
	}
	KO = Vitals[len(Vitals)-1]
}

func init() {
	theme.OnChange(paint)
	paint()
}
