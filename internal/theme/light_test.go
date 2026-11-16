package theme

import (
	"os"
	"path/filepath"
	"testing"
)

// useLight switches to the light palette for one test and puts the dark one
// back afterwards, whatever happens in between.
func useLight(t *testing.T) {
	t.Helper()
	Use(true)
	t.Cleanup(func() { Use(false) })
}

// Every colour that carries text has to read on the light footer. The dark
// pastels are the failure this palette exists for: #4dd6c1 on white is 1.79.
func TestElClaroSeLeeSobreElPie(t *testing.T) {
	useLight(t)
	for name, c := range map[string]Colour{
		"Path": Path, "Ident": Ident, "Link": Link, "Number": Number, "Mode": Mode,
		"Bad": Bad, "Emph": Emph, "Dim": Dim, "Dir": Dir, "Text": Text,
		"Quota": Quota, "Drowned": Drowned, "Grey": Grey,
	} {
		if got := ContrastRatio(c, Bg); got < 4.5 {
			t.Errorf("%s %02x%02x%02x sobre el pie: %.2f, por debajo de 4.5", name, c.R, c.G, c.B, got)
		}
	}
}

// The footer is the light terminal's own background, #f3f3f3, with the
// #e5e5e5 hairline: not a band of a different white under the conversation.
func TestElPieEsLaBaseDelTerminal(t *testing.T) {
	useLight(t)
	if Bg != Hex("#f3f3f3") || Border != Hex("#e5e5e5") {
		t.Errorf("pie %v, línea %v: tienen que ser #f3f3f3 y #e5e5e5", Bg, Border)
	}
}

// Switching back must leave the dark palette exactly as it was: every reader
// of the dark one would otherwise inherit a half-swapped table.
func TestVolverAlOscuroEsExacto(t *testing.T) {
	before := make([]Colour, len(twins))
	for i, tw := range twins {
		before[i] = *tw.at
	}
	Use(true)
	Use(false)
	for i, tw := range twins {
		if *tw.at != before[i] {
			t.Errorf("variable %d: %v tras ida y vuelta, era %v", i, *tw.at, before[i])
		}
	}
}

// On has to have a light ink to offer in the light palette too. With Emph as
// that ink it compared black against black and picked black on dark pills.
func TestOnEnClaroOfreceTintaBlanca(t *testing.T) {
	useLight(t)
	deep := Hex("#036a5d") // the "fresca" step of the teal ramp
	if On(deep) == Black {
		t.Errorf("sobre %v eligió negro (%.2f) y el blanco da %.2f",
			deep, ContrastRatio(Black8, deep), ContrastRatio(Paper, deep))
	}
	if On(Bg) != Black {
		t.Errorf("sobre el pie claro tenía que elegir negro")
	}
}

func TestLaPaletaSigueAlTemaDeClaudeCode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		themes   map[string]string
		want     bool
	}{
		{"sin settings", "", nil, false},
		{"sin tema: el de serie es oscuro", `{"model":"x"}`, nil, false},
		{"integrado oscuro", `{"theme":"dark"}`, nil, false},
		{"integrado claro", `{"theme":"light"}`, nil, true},
		{"integrado claro daltónico", `{"theme":"light-daltonized"}`, nil, true},
		{"propio con base clara", `{"theme":"custom:terminal-light"}`,
			map[string]string{"terminal-light": `{"name":"Terminal Light","base":"light"}`}, true},
		{"propio con base oscura manda la base, no el nombre", `{"theme":"custom:not-light"}`,
			map[string]string{"not-light": `{"base":"dark"}`}, false},
		{"propio sin fichero: queda el nombre", `{"theme":"custom:paper-light"}`, nil, true},
		{"propio sin fichero ni pista", `{"theme":"custom:terminal"}`, nil, false},
		{"un slug no sale de themes/", `{"theme":"custom:../evil"}`,
			map[string]string{"../evil": `{"base":"light"}`}, false},
		{"settings rotos", `{"theme":`, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.settings != "" {
				os.WriteFile(filepath.Join(dir, "settings.json"), []byte(tc.settings), 0o600)
			}
			os.MkdirAll(filepath.Join(dir, "themes"), 0o700)
			for slug, body := range tc.themes {
				os.WriteFile(filepath.Join(dir, "themes", slug+".json"), []byte(body), 0o600)
			}
			if got := LightTheme(dir); got != tc.want {
				t.Errorf("LightTheme = %v, se esperaba %v", got, tc.want)
			}
		})
	}
}

// STATUSLINE_THEME wins over whatever Claude Code is set to, both ways.
func TestStatuslineThemeFuerzaElModo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme":"light"}`), 0o600)
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	for env, want := range map[string]bool{"": true, "dark": false, "DARK": false, "light": true, "otra": true} {
		t.Setenv("STATUSLINE_THEME", env)
		if got := DetectLight(); got != want {
			t.Errorf("STATUSLINE_THEME=%q: %v, se esperaba %v", env, got, want)
		}
	}
}
