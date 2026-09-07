package pet

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The canvas "Atlas de Formas y Estados" ends its header with a promise:
// "41 x 7 = 287 variantes de diez rampas, sin dos iguales". That is a testable
// claim, and it is the whole reason colour moved from the state to the form -
// before it, 27 forms came out as 20 distinct silhouettes at "easy" and 14 at
// k.o., so a cazabugs, a kraken and a francotirador were the same creature.
func TestNoTwoVariantsAreAlike(t *testing.T) {
	seen := map[string]string{}
	for form := range Sprites {
		for _, v := range Vitals {
			key := ""
			for _, row := range Draw(form, v, 0, false) {
				key += row
			}
			if other, clash := seen[key]; clash {
				t.Errorf("%s and %s draw the same thing at %q", other, form, v.Label)
			}
			seen[key] = form
		}
	}
	if want := len(Sprites) * len(Vitals); len(seen) != want {
		t.Errorf("%d distinct variants, want %d", len(seen), want)
	}
}

func TestTheAtlasIsAllThere(t *testing.T) {
	if len(Sprites) != 41 {
		t.Errorf("%d sprites, want the atlas's 41", len(Sprites))
	}
	if len(Ramps) != 10 {
		t.Errorf("%d ramps, want 10", len(Ramps))
	}
	for form := range Sprites {
		if _, ok := FormRamp[form]; !ok {
			t.Errorf("%s has no ramp", form)
		}
	}
	for form, ramp := range FormRamp {
		if _, ok := Sprites[form]; !ok {
			t.Errorf("FormRamp has %q, which is not a form", form)
		}
		if _, ok := Ramps[ramp]; !ok {
			t.Errorf("%s points at ramp %q, which does not exist", form, ramp)
		}
	}
}

func TestEveryRowIsNineCellsInEveryState(t *testing.T) {
	// One cell of drift and the whole footer goes crooked.
	for form := range Sprites {
		for _, v := range Vitals {
			for i, row := range Draw(form, v, 0, false) {
				if got := theme.Width(row); got != SpriteWidth {
					t.Errorf("%s %s row %d = %d cells, want %d", form, v.Label, i, got, SpriteWidth)
				}
			}
			for i, row := range DrawCompact(form, v, 0, false) {
				if got := theme.Width(row); got != SpriteWidth {
					t.Errorf("%s %s compact row %d = %d cells", form, v.Label, i, got)
				}
			}
		}
	}
}

func TestTheCrestAndTheFeetSurviveEveryState(t *testing.T) {
	// "La marca de arriba y el numero de patas identifican la forma y no
	// cambian nunca" - including the k.o., which lies the feet down without
	// losing one: "lo tumba en k.o. sin perder la cuenta de patas".
	count := func(row string) int {
		n := 0
		for _, r := range row {
			if r != ' ' {
				n++
			}
		}
		return n
	}
	for form, sprite := range Sprites {
		want := count(sprite.Feet)
		for _, row := range []string{sprite.Step, sprite.Still, sprite.KO} {
			if got := count(row); got != want {
				t.Errorf("%s: %q has %d feet, %q has %d", form, sprite.Feet, want, row, got)
			}
		}
		for _, v := range Vitals {
			if got := theme.Strip(Draw(form, v, 0, false)[0]); got != sprite.Crest {
				t.Errorf("%s lost its crest at %q: %q", form, v.Label, got)
			}
		}
	}
}

func TestATitleNeverAsksLessThanItsMark(t *testing.T) {
	// The chain is mark then title, so a title asking for less than its mark
	// is a form nothing can ever reach: you would have to un-earn the mark to
	// qualify. A flat three-times factor put storm at 7 against bolt's 10 and
	// did exactly that.
	for mark, title := range Titles {
		base, ok := Unlocks[mark]
		if !ok {
			t.Errorf("%s has no unlock of its own", mark)
			continue
		}
		u, ok := TitleUnlock(mark)
		if !ok {
			t.Errorf("%s opens nothing", mark)
			continue
		}
		if u.Counter != base.Counter {
			t.Errorf("%s asks for %q, its mark asks for %q", title, u.Counter, base.Counter)
		}
		if u.Threshold <= base.Threshold {
			t.Errorf("%s asks %d, but %s already asked %d - it can never be reached",
				title, u.Threshold, mark, base.Threshold)
		}
	}
}
