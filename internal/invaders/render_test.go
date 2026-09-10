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
// test asserting "no wider than the terminal" was satisfied. Nothing legitimate
// in a frame contains a bracket.
func TestNoLineIsEverCutInsideAnEscapeSequence(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, cols := range []int{MinCols, 62, 70, 76, 80, 116} {
			f := aField(t, cols, MinRows)
			for _, g := range states(t, f) {
				for i, line := range Render(g, cols) {
					if plain := theme.Strip(line); strings.ContainsAny(plain, "[\033") {
						t.Fatalf("%s, %d cols, row %d: an escape leaked: %q", lang, cols, i, plain)
					}
				}
			}
		}
	}
	i18n.Use("")
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

// The creature at the bottom is the pet, drawn by pet.DrawCompact with its own
// ramp - not a glyph that happens to look like one.
func TestTheCreatureIsTheFormsOwnSpriteAndItsOwnRamp(t *testing.T) {
	f := aField(t, 80, 24)
	for _, form := range []string{"spark", "wasp", "phoenix", "marathon"} {
		g := NewGame(f, form, 4, Save{Wave: 1, Seed: 1})
		frame := strings.Join(Render(g, 80), "\n")
		for i, row := range pet.DrawTiny(form, g.Vital(), g.Frame/8, false) {
			if !strings.Contains(frame, row) {
				t.Errorf("%s: tiny row %d is not in the frame", form, i)
			}
		}
	}
}

// A dying creature lies down: the same seven states the statusline uses, so the
// player reads its health off the creature rather than off a bar.
func TestADyingCreatureLiesDown(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "bughunter", 4, Save{Wave: 1, Seed: 1})
	g.HP = 0
	frame := strings.Join(Render(g, 80), "\n")
	down := pet.DrawTiny("bughunter", pet.KO, g.Frame/8, false)
	for i, row := range down {
		if !strings.Contains(frame, row) {
			t.Errorf("row %d of the k.o. creature is not in the frame", i)
		}
	}

	g.HP = g.Kit.MaxHP
	if up := strings.Join(Render(g, 80), "\n"); strings.Contains(up, down[len(down)-1]) {
		t.Error("a creature at full life has its eyes shut")
	}
}

// The block is drawn with the arcade's own sprites, in the stage's colour.
func TestTheBlockIsDrawnWithTheArcadeSprites(t *testing.T) {
	f := aField(t, 80, 24)
	g := NewGame(f, "spark", 1, Save{Wave: 1, Seed: 1})
	frame := strings.Join(Render(g, 80), "\n")
	plain := theme.Strip(frame)

	for _, s := range Troops {
		want := false
		for _, i := range g.Wave.Species {
			want = want || Troops[i].Name == s.Name
		}
		if got := strings.Contains(plain, s.Glyph); got != want {
			t.Errorf("the %s is on screen = %v, want %v", s.Name, got, want)
		}
	}

	pal := Stages[g.Wave.Stage-1]
	if !strings.Contains(frame, theme.Fg(pal.Tones[0])) {
		t.Error("the block is not painted in the stage's tone")
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
		for _, want := range []string{"←", "→", "x", "p", "q"} {
			if !strings.Contains(help, want) {
				t.Errorf("%s: the help row does not mention %q: %q", lang, want, help)
			}
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
