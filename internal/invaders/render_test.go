package invaders

import (
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// states is a spread of the shapes a frame can be in, so the width and layout
// guards are not all looking at an empty field.
func states(t *testing.T, f Field) []Game {
	t.Helper()
	out := []Game{}
	for _, c := range []struct {
		form  string
		level int
		wave  int
	}{
		{"spark", 1, 1}, {"wasp", 6, 5}, {"marathon", 4, 12},
		{"chimera", 5, 25}, {"nonsense-form", 3, 40}, {"leviathan", 6, 175},
	} {
		g := NewGame(f, c.form, c.level, Save{Wave: c.wave, Seed: 0x5EED})
		for _, ticks := range []int{0, 1, 90, 400} {
			out = append(out, drive(g, Fire, ticks))
		}
	}
	return out
}

// Nothing may be wider than the terminal, in cells, with the escapes stripped.
// One line too long wraps, which pushes the field down a row and turns every
// frame after it into a smear.
func TestNoRenderedLineIsWiderThanTheTerminal(t *testing.T) {
	for _, cols := range []int{MinCols, 72, 80, 116, 200} {
		f := aField(t, cols, 24)
		for _, g := range states(t, f) {
			for i, line := range Render(g, cols) {
				if w := theme.Width(line); w > cols {
					t.Fatalf("%d cols, wave %d: row %d is %d cells: %q",
						cols, g.Wave.N, i, w, theme.Strip(line))
				}
			}
		}
	}
}

// Exactly as many rows as the terminal has, always.
func TestEveryFrameIsExactlyTheRowsTheTerminalHas(t *testing.T) {
	for _, rows := range []int{MinRows, 24, 40} {
		f := aField(t, 80, rows)
		for _, g := range states(t, f) {
			if got := len(Render(g, 80)); got != rows {
				t.Fatalf("%d rows of terminal drew %d lines", rows, got)
			}
		}
	}
}

// A painted line may never be cut inside an escape sequence.
//
// theme.Truncate counts the bytes of an escape as visible width and cuts
// wherever it lands, so a Spanish HUD in a 76-column terminal once printed a
// literal "[38" where the life bar should have been - three cells wide, so every
// test asserting "no wider than the terminal" was satisfied.
//
// What it looks for is a bracket followed by a DIGIT, not a bare bracket. The
// bare-bracket version was the first draft and it went red the moment the fleet
// arrived: two of the ten ships are drawn with brackets, and so is the turret
// family's hull. "[o-o-o]" is a coraza; "[38;2" is a bug.
func TestNoLineIsEverCutInsideAnEscapeSequence(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, cols := range []int{MinCols, 62, 70, 76, 80, 116} {
			f := aField(t, cols, MinRows)
			for _, g := range states(t, f) {
				for i, line := range Render(g, cols) {
					plain := theme.Strip(line)
					if strings.Contains(plain, "\033") || cutEscape(plain) {
						t.Fatalf("%s, %d cols, row %d: an escape leaked: %q", lang, cols, i, plain)
					}
				}
			}
		}
	}
	i18n.Use("")
}

// cutEscape is the tail of a chopped escape sequence: a bracket with a digit, a
// semicolon or a question mark after it.
func cutEscape(plain string) bool {
	for i := 0; i+1 < len(plain); i++ {
		if plain[i] != '[' {
			continue
		}
		switch c := plain[i+1]; {
		case c >= '0' && c <= '9', c == ';', c == '?':
			return true
		}
	}
	return false
}

// The HUD gives up its least important parts rather than its shape.
func TestANarrowHudDropsWholePartsAndKeepsTheWaveAndTheLife(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		f := aField(t, MinCols, MinRows)
		g := NewGame(f, "exterminator", 5, Save{Wave: 5, Seed: 1})

		wide, narrow := hud(g, 200), hud(g, MinCols)
		if theme.Width(narrow) > MinCols {
			t.Errorf("%s: the narrow hud is %d cells", lang, theme.Width(narrow))
		}
		if !strings.Contains(theme.Strip(narrow), "♥") {
			t.Errorf("%s: the narrow hud dropped the life: %q", lang, theme.Strip(narrow))
		}
		if theme.Width(wide) <= theme.Width(narrow) {
			t.Errorf("%s: the hud did not grow when given room", lang)
		}
	}
	i18n.Use("")
}

// The ship is the family's silhouette in the form's own colour: a representation
// of the creature, drawn in the same line art as the fleet. See ship.go.
func TestTheShipIsTheFamilysHullInTheFormsOwnColour(t *testing.T) {
	f := aField(t, 80, 24)
	for _, form := range []string{"spark", "wasp", "phoenix", "marathon"} {
		g := NewGame(f, form, 4, Save{Wave: 1, Seed: 1})
		frame := strings.Join(Render(g, 80), "\n")
		plain := theme.Strip(frame)

		for i, row := range ShipArt(g.Kit.Family, g.Vital()) {
			if !strings.Contains(plain, strings.TrimSpace(row)) {
				t.Errorf("%s: row %d of the %s hull is not in the frame: %q",
					form, i, g.Kit.Family, row)
			}
		}
		ramp := pet.RampOf(form)
		if !strings.Contains(frame, theme.Fg(ramp.Body[0])) {
			t.Errorf("%s: the ship is not painted in the form's own colour", form)
		}
	}
}

// Two forms in different families draw different ships, which is the whole point
// of there being thirteen hulls.
func TestTwoFormsFromDifferentBranchesLookDifferent(t *testing.T) {
	f := aField(t, 80, 24)
	seen := map[string]string{}
	for _, form := range []string{"spark", "pattern", "bughunter", "architect", "marathon", "feral"} {
		g := NewGame(f, form, 4, Save{Wave: 1, Seed: 1})
		art := ShipArt(g.Kit.Family, g.Vital())
		key := strings.Join(art[:], "|")
		if other, dup := seen[key]; dup {
			t.Errorf("%s flies the same ship as %s", form, other)
		}
		seen[key] = form
	}
}

// A dying ship has its eyes out, which is how the creature's own seven states
// survive into the representation.
func TestADyingShipHasItsEyesOut(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "bughunter", 4, Save{Wave: 1, Seed: 1})
	g.HP = 0
	if plain := theme.Strip(strings.Join(Render(g, 80), "\n")); !strings.Contains(plain, "x x") {
		t.Error("a dead ship still has its eyes open")
	}
	g.HP = g.Kit.MaxHP
	if plain := theme.Strip(strings.Join(Render(g, 80), "\n")); !strings.Contains(plain, "o o") {
		t.Error("a ship at full life does not have its eyes open")
	}
}

// The fleet is drawn with its own art, in the wave's stage tone.
func TestTheFleetIsDrawnWithItsOwnArt(t *testing.T) {
	f := aField(t, 80, 24)
	for of, c := range Fleet {
		g := NewGame(f, "spark", 1, Save{Wave: 1, Seed: 1})
		g = oneAlien(g, of, 20, 3)
		plain := theme.Strip(strings.Join(Render(g, 80), "\n"))
		for i, row := range c.Rows {
			if !strings.Contains(plain, strings.TrimSpace(row)) {
				t.Errorf("%s: row %d is not in the frame: %q", c.Name, i, row)
			}
		}
		frame := strings.Join(Render(g, 80), "\n")
		pal := Stages[g.Wave.Stage-1]
		if !strings.Contains(frame, theme.Fg(pal.Tones[0])) {
			t.Errorf("%s is not painted in the stage's tone", c.Name)
		}
	}
}

// The sky is behind everything and never on top of it: a star drawn over the
// hull is a hole in the ship.
func TestTheSkyIsBehindTheShip(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "spark", 1, Save{Wave: 1, Seed: 7})
	if len(g.Stars) == 0 {
		t.Fatal("there is no sky")
	}
	// A star put exactly on the hull must not show through it.
	g.Ship = 20
	g.Stars = []Star{{X: 22, Y: float64(g.Field.ShipRow() + 1), V: 0.14}}
	rows := Render(g, 80)
	line := theme.Strip(rows[HUDRows+g.Field.ShipRow()+1])
	if line[22] == '.' || line[22] == '*' {
		t.Errorf("a star is showing through the hull: %q", line)
	}
}

// A rock and a health kit are drawn, and are not the same glyph as each other.
func TestTheRocksAndTheKitsAreTellableApart(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "spark", 1, Save{Wave: 1, Seed: 1})
	g.Stones = []Stone{{X: 10, Y: 4, HP: Rock.HP, MaxHP: Rock.HP}}
	g.Drops = []Drop{{X: 40, Y: 6}}
	plain := theme.Strip(strings.Join(Render(g, 80), "\n"))
	if !strings.Contains(plain, strings.TrimSpace(Rock.Rows[1])) {
		t.Error("the rock is not in the frame")
	}
	if !strings.Contains(plain, "✚") {
		t.Error("the health kit is not in the frame")
	}
}

// The magazine is in the HUD, because it is the number you have to know before
// you press anything - and it says so while it is reloading.
func TestTheHudSaysHowManyRoundsAreLeft(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "marathon", 4, Save{Wave: 1, Seed: 1})
	g.Ammo = 3
	if plain := theme.Strip(hud(g, 200)); !strings.Contains(plain, "3/") {
		t.Errorf("the hud does not say the rounds left: %q", plain)
	}
	g.Loading = 10
	if plain := theme.Strip(hud(g, 200)); !strings.Contains(plain, i18n.G().Reloading) {
		t.Errorf("a reloading gun does not say so: %q", plain)
	}
	g.Loading = 0
	g.Kits = 2
	if plain := theme.Strip(hud(g, 200)); !strings.Contains(plain, "✚2") {
		t.Errorf("the hud does not say what is in the hold: %q", plain)
	}
}

// A boss is one big sprite off the canvas, drawn where it stands.
func TestABossIsDrawnAsTheSpriteItIs(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "spark", 5, Save{Wave: 5, Seed: 3})
	if !g.Boss.Alive {
		t.Fatal("wave 5 has no boss")
	}
	b := Bosses[g.Boss.Of]
	plain := theme.Strip(strings.Join(Render(g, 80), "\n"))
	for name, row := range map[string]string{"top": b.Top, "upper": b.Upper, "lower": b.Lower} {
		if !strings.Contains(plain, strings.TrimSpace(row)) {
			t.Errorf("the %s of the %s is not in the frame", name, b.Name)
		}
	}
}

// Every banner the tick can set has to turn into words, in both languages.
func TestEveryBannerTheTickCanSetTurnsIntoWords(t *testing.T) {
	f := aField(t, 80, 24)
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, id := range []string{
			BannerCleared, BannerBoss, BannerBossOff, BannerRevived,
			BannerLanded, BannerOver, BannerClaude, BannerPaused,
			BannerChoose, BannerKit,
		} {
			g := NewGame(f, "spark", 3, Save{Wave: 5, Seed: 1})
			g.Banner = id
			if got := Banner(g); strings.TrimSpace(got) == "" {
				t.Errorf("%s: the banner %q says nothing", lang, id)
			}
			if strings.Contains(Banner(g), "%!") {
				t.Errorf("%s: the banner %q is a broken format: %q", lang, id, Banner(g))
			}
		}
		g := NewGame(f, "spark", 3, Save{Wave: 1, Seed: 1})
		g.Banner = ""
		if got := Banner(g); got != "" {
			t.Errorf("%s: no banner still said %q", lang, got)
		}
	}
	i18n.Use("")
}

// The help row has to name the keys that exist, in both languages. It is the
// only instructions there are.
func TestTheHelpRowNamesTheKeysThatExist(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		help := i18n.G().Help
		for _, want := range []string{"←", "→", "x", "r", "e", "p", "q"} {
			if !strings.Contains(help, want) {
				t.Errorf("%s: the help row does not mention %q: %q", lang, want, help)
			}
		}
		// The tight one may drop the brake and the long names, but not a key
		// that does something nothing else does.
		for _, want := range []string{"←", "→", "x", "r", "e", "p", "q"} {
			if !strings.Contains(i18n.G().Tight, want) {
				t.Errorf("%s: the tight help row does not mention %q: %q",
					lang, want, i18n.G().Tight)
			}
		}
	}
	i18n.Use("")
}

// And it is chosen by width rather than cut: a key row that ends mid-word has
// stopped being a key row. In Spanish the full one is ninety columns.
func TestTheHelpRowShrinksInsteadOfBeingCut(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		w := i18n.G()
		if got := helpRow(w, 200); got != w.Help {
			t.Errorf("%s: with room to spare it printed the tight row", lang)
		}
		if got := helpRow(w, MinCols); theme.Width(got) > MinCols {
			t.Errorf("%s: at %d columns it printed %d: %q", lang, MinCols, theme.Width(got), got)
		}
		if theme.Width(w.Tight) > MinCols {
			t.Errorf("%s: the tight row is %d columns and the floor is %d",
				lang, theme.Width(w.Tight), MinCols)
		}
	}
	i18n.Use("")
}

// The refusal has to say both pairs of numbers.
func TestTheTooSmallRefusalSaysBothNumbers(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		got := TooSmall(40, 10)
		for _, want := range []string{"60", "18", "40", "10"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: %q does not say %s", lang, got, want)
			}
		}
	}
	i18n.Use("")
}

// Colour comes from internal/theme, which is the only place in the repo that
// emits an escape.
func TestNothingIsPaintedWithARawEscapeThatThemeDoesNotOwn(t *testing.T) {
	raw := readSource(t, "render.go")
	if strings.Contains(raw, "\\033[3") || strings.Contains(raw, "\\x1b[3") {
		t.Error("render.go writes a colour escape of its own")
	}
}
