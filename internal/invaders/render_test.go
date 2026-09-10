package invaders

import (
	"os"
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
		{"chimera", 5, 25}, {"crawler-nonsense", 3, 40}, {"leviathan", 6, 200},
	} {
		g := NewGame(f, c.form, c.level, Save{Wave: c.wave, Seed: 0x5EED})
		for _, ticks := range []int{0, 1, 90, 400} {
			out = append(out, drive(g, Fire, ticks))
		}
	}
	return out
}

// Nothing may be wider than the terminal, in cells, with the escapes stripped.
// One line too long wraps, which pushes the whole field down a row and turns
// every subsequent frame into a smear.
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

// Exactly as many rows as the terminal has, always. One short and the shell
// prompt shows through the bottom of the field; one long and it scrolls.
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

// The HUD has to survive the narrowest terminal the game accepts, in both
// languages, without eating the field.
func TestTheHudFitsInTheNarrowestTerminal(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		f := aField(t, MinCols, MinRows)
		for _, g := range states(t, f) {
			line := hud(g, MinCols)
			if w := theme.Width(line); w > MinCols {
				t.Errorf("%s: the hud is %d cells of %d: %q", lang, w, MinCols, theme.Strip(line))
			}
			if strings.TrimSpace(theme.Strip(line)) == "" {
				t.Errorf("%s: the hud is blank", lang)
			}
		}
	}
	i18n.Use("")
}

// The ship is the creature, drawn by pet.Draw with its own ramp - not a glyph
// that happens to look like one. The painted rows have to come through verbatim,
// or the sprite is being re-implemented here and internal/pet's own guarantees
// stop applying.
func TestTheShipIsTheFormsOwnSpriteAndItsOwnRamp(t *testing.T) {
	f := aField(t, 80, 24)
	for _, form := range []string{"spark", "wasp", "phoenix", "marathon"} {
		g := NewGame(f, form, 4, Save{Wave: 1, Seed: 1})
		g.Ship = 3
		frame := strings.Join(Render(g, 80), "\n")
		for i, row := range pet.Draw(form, g.Vital(), g.Frame/8, false) {
			if !strings.Contains(frame, row) {
				t.Errorf("%s: sprite row %d is not in the frame", form, i)
			}
		}
	}
}

// A dying creature lies down. It is the same seven states the statusline uses,
// so the player reads the ship's health off the ship rather than off a bar.
func TestADyingShipLiesDown(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "bughunter", 4, Save{Wave: 1, Seed: 1})
	g.HP = 0
	frame := strings.Join(Render(g, 80), "\n")
	if !strings.Contains(frame, pet.Sprites["bughunter"].KO) {
		t.Error("a ship at zero hp is not lying down")
	}
	g.HP = g.Kit.MaxHP
	if frame := strings.Join(Render(g, 80), "\n"); strings.Contains(frame, pet.Sprites["bughunter"].KO) {
		t.Error("a ship at full hp is lying down")
	}
}

// A rival is drawn as the creature it is, by the same painter, and it wears down
// through the same states. This is the whole reason forty-one bosses cost no new
// drawing code.
func TestABossIsDrawnAsTheCreatureItIs(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "spark", 5, Save{Wave: 5, Seed: 3})
	if len(g.Enemies) != 1 || !g.Enemies[0].Boss() {
		t.Fatalf("wave 5 has no rival: %+v", g.Enemies)
	}
	boss := g.Enemies[0]

	frame := strings.Join(Render(g, 80), "\n")
	for i, row := range pet.Draw(boss.Rival, boss.Vital(), g.Frame/8, false) {
		if !strings.Contains(frame, row) {
			t.Errorf("the rival's sprite row %d is not in the frame", i)
		}
	}

	g.Enemies[0].HP = 0
	if frame := strings.Join(Render(g, 80), "\n"); !strings.Contains(frame, pet.Sprites[boss.Rival].KO) {
		t.Error("a rival at zero hp is not lying down")
	}
}

// Two creatures on screen at once must not overwrite each other, and the frame
// must still be exactly the right width. A nine-cell painted block spliced into
// the middle of a row is the one place the assembly can lose count.
func TestARivalAndTheShipBothFitOnTheirRows(t *testing.T) {
	f := aField(t, MinCols, MinRows)
	g := NewGame(f, "wasp", 6, Save{Wave: 5, Seed: 9})
	g.Ship = 2
	g.Enemies[0].Row = 2

	for _, line := range Render(g, MinCols) {
		if w := theme.Width(line); w > MinCols {
			t.Fatalf("a row with two creatures on it is %d cells: %q", w, theme.Strip(line))
		}
	}
	frame := strings.Join(Render(g, MinCols), "\n")
	if !strings.Contains(frame, pet.Draw("wasp", g.Vital(), g.Frame/8, false)[0]) {
		t.Error("the ship lost its crest to the rival")
	}
}

// The refusal has to say both pairs of numbers. "Too small" without them is a
// message that makes the player guess.
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

// Every banner the tick can set has to turn into words, in both languages. A
// banner id that falls through prints nothing at all, which reads as the game
// having frozen.
func TestEveryBannerTheTickCanSetTurnsIntoWords(t *testing.T) {
	f := aField(t, 80, 24)
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, id := range []string{
			BannerCleared, BannerBoss, BannerBossOff,
			BannerRevived, BannerOver, BannerClaude, BannerPaused,
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

// Colour comes from internal/theme, which is the only place in the repo that
// emits an escape. The frame's own cursor and screen-mode escapes live in
// run.go, not here.
func TestNothingIsPaintedWithARawEscapeThatThemeDoesNotOwn(t *testing.T) {
	raw := mustRead(t, "render.go")
	if strings.Contains(raw, "\\033[3") || strings.Contains(raw, "\\x1b[3") {
		t.Error("render.go writes a colour escape of its own")
	}
}

// Thirty-five kinds have to be tellable apart on screen, or composing them was
// pointless: the glyph says the body and the colour says the trait.
func TestEveryOneOfTheThirtyFiveEnemiesIsTellableFromTheOthers(t *testing.T) {
	theme.SetTruecolor(true)
	defer theme.SetTruecolor(false)

	seen := map[string]string{}
	for b := Body(0); b < bodyCount; b++ {
		for tr := Trait(0); tr < traitCount; tr++ {
			e := Enemy{Body: b, Trait: tr}
			painted := theme.Fg(traitInk[tr]) + e.Glyph() + theme.Reset
			name := bodies[b].ID + "/" + traitIDs[tr]
			if other, dup := seen[painted]; dup {
				t.Errorf("%s is drawn exactly like %s", name, other)
			}
			seen[painted] = name
		}
	}
	if len(seen) != 35 {
		t.Errorf("%d distinguishable kinds, want 35", len(seen))
	}
}

func mustRead(t *testing.T, name string) string {
	t.Helper()
	raw, err := readFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// readFile is here rather than inline so the source-scanning guards do not have
// to import os into a test file that is otherwise about pictures.
func readFile(name string) (string, error) {
	raw, err := os.ReadFile(name)
	return string(raw), err
}

// A painted line may never be cut inside an escape sequence.
//
// This is the guard that was missing. theme.Truncate counts the bytes of an
// escape as visible width and cuts wherever it lands, so a Spanish HUD in a
// 76-column terminal printed a literal "[38" where the life bar should have
// been - and the width test passed, because the fragment is only three cells
// wide. Nothing legitimate in a frame contains a bracket.
func TestNoLineIsEverCutInsideAnEscapeSequence(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, cols := range []int{MinCols, 62, 70, 76, 80, 116} {
			f := aField(t, cols, MinRows)
			for _, g := range states(t, f) {
				for i, line := range Render(g, cols) {
					plain := theme.Strip(line)
					if strings.ContainsAny(plain, "[\033") {
						t.Fatalf("%s, %d cols, row %d: an escape leaked through: %q",
							lang, cols, i, plain)
					}
				}
			}
		}
	}
	i18n.Use("")
}

// The HUD gives up its least important parts rather than its shape. At sixty
// columns in Spanish it cannot hold everything, and what it must never do is
// end mid-word or mid-colour.
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
