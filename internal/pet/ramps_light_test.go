package pet

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

func useLight(t *testing.T) {
	t.Helper()
	theme.Use(true)
	t.Cleanup(func() { theme.Use(false) })
}

// Same ten branches in both palettes: a form with no light ramp would fall
// back to the larva's grey in a white terminal and lose its colour, which is
// what tells the forms apart.
func TestLasRampasClarasCubrenLasMismasRamas(t *testing.T) {
	for name := range darkRamps {
		if _, ok := lightRamps[name]; !ok {
			t.Errorf("la rama %q no tiene rampa clara", name)
		}
	}
	if len(lightRamps) != len(darkRamps) {
		t.Errorf("%d rampas claras para %d oscuras", len(lightRamps), len(darkRamps))
	}
}

// The physics of the light ramps: energy is ink. Each step reads less than the
// one before it on the footer, "fresca" and "vibrante" hold the 3:1 a graphic
// needs, k.o. fades nearly into the background, and the lit eye stands out.
func TestEnClaroLaMascotaSeDestineHaciaElFondo(t *testing.T) {
	useLight(t)
	for name, r := range Ramps {
		prev := 99.0
		for i, c := range r.Body {
			got := theme.ContrastRatio(c, theme.Bg)
			if got >= prev {
				t.Errorf("%s paso %d: %.2f, no se destiñe respecto al anterior (%.2f)", name, i, got, prev)
			}
			prev = got
		}
		for i := 0; i < 2; i++ {
			if got := theme.ContrastRatio(r.Body[i], theme.Bg); got < 3 {
				t.Errorf("%s paso %d: %.2f sobre el pie, por debajo de 3", name, i, got)
			}
		}
		if got := theme.ContrastRatio(r.Body[6], theme.Bg); got > 2 {
			t.Errorf("%s k.o.: %.2f, no parece muerta", name, got)
		}
		if got := theme.ContrastRatio(r.Eye, theme.Bg); got < 4.5 {
			t.Errorf("%s ojo: %.2f sobre el pie", name, got)
		}
	}
}

// The model pill is painted the creature's colour, with On picking the ink.
// Whatever the step, one of the two inks has to read on it.
func TestLaPastillaSeLeeEnTodosLosPasos(t *testing.T) {
	useLight(t)
	for name, r := range Ramps {
		for i, c := range r.Body {
			best := theme.ContrastRatio(theme.Black8, c)
			if w := theme.ContrastRatio(theme.Paper, c); w > best {
				best = w
			}
			if best < 4.5 {
				t.Errorf("%s paso %d: la mejor tinta da %.2f", name, i, best)
			}
		}
	}
}

// Use repaints what pet copied: the state colours and the ramp table.
func TestCambiarDePaletaRepintaLosEstados(t *testing.T) {
	useLight(t)
	if Vitals[0].Colour != theme.Ident || KO.Colour != theme.Grey {
		t.Errorf("los estados no se repintaron al pasar a claro")
	}
	if RampOf("pattern").Body[1] != lightRamps["pattern"].Body[1] {
		t.Errorf("RampOf sigue dando la rampa oscura")
	}
	theme.Use(false)
	if Vitals[0].Colour != theme.Ident || RampOf("pattern").Body[1] != darkRamps["pattern"].Body[1] {
		t.Errorf("al volver a oscuro no se repintó")
	}
}
