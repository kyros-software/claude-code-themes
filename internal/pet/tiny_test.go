package pet

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Five cells, always, whatever the state and whatever the form. A row that
// measures anything else shears the row it is drawn into - which for the game
// this exists for means the floor of the field.
func TestTheTinyCreatureIsFiveCellsInEveryState(t *testing.T) {
	if len(Sprites) != 41 {
		t.Fatalf("the atlas is 41 forms, there are %d", len(Sprites))
	}
	for form := range Sprites {
		for _, v := range Vitals {
			for _, step := range []int{0, 1, 3, 7, 12} {
				for _, dim := range []bool{false, true} {
					rows := DrawTiny(form, v, step, dim)
					for i, row := range rows {
						if w := theme.Width(row); w != TinyWidth {
							t.Fatalf("%s, %s, step %d: row %d is %d cells, want %d: %q",
								form, v.Label, step, i, w, TinyWidth, theme.Strip(row))
						}
					}
				}
			}
		}
	}
}

// It has to stay a creature at this size, and a creature is its eyes. They are
// the one thing that says which of the seven states it is in - and at zero life
// they are what says it is over.
func TestTheTinyCreatureKeepsItsEyes(t *testing.T) {
	for form := range Sprites {
		for _, v := range Vitals {
			rows := DrawTiny(form, v, 0, false)
			plain := []rune(theme.Strip(rows[1]))
			if got, want := plain[1], v.Eyes[0]; got != want {
				t.Errorf("%s, %s: left eye is %q, want %q", form, v.Label, got, want)
			}
			if got, want := plain[3], v.Eyes[1]; got != want {
				t.Errorf("%s, %s: right eye is %q, want %q", form, v.Label, got, want)
			}
		}
	}
	// And the k.o. eyes are not the fresh ones, or dying looks like living.
	fresh := theme.Strip(DrawTiny("wasp", Vitals[0], 0, false)[1])
	down := theme.Strip(DrawTiny("wasp", KO, 0, false)[1])
	if fresh == down {
		t.Errorf("a k.o. creature is drawn exactly like a fresh one: %q", down)
	}
}

// Five cells cannot carry a whole form, but it must not collapse the atlas into
// a handful of shapes either. The crest's middle tells 31 of the 41 apart on its
// own; the rest are a mark and its title, or two forms of one branch, and the
// colour separates those.
//
// The first draft of this drew a shoulder either side of the eyes and nothing
// else, and gave FORTY of the forty-one the same five glyphs. That is what the
// number below is guarding against.
func TestTheTinyCreatureStillTellsMostFormsApart(t *testing.T) {
	seen := map[string]string{}
	for form := range Sprites {
		rows := DrawTiny(form, Vitals[0], 0, false)
		key := theme.Strip(rows[0]) + "|" + theme.Strip(rows[1])
		seen[key] = form
	}
	if len(seen) < 25 {
		t.Errorf("the 41 forms come out as %d shapes, which is not a creature any more", len(seen))
	}

	// And where two share a shape they must not also share a colour.
	byShape := map[string][]string{}
	for form := range Sprites {
		rows := DrawTiny(form, Vitals[0], 0, false)
		byShape[theme.Strip(rows[0])] = append(byShape[theme.Strip(rows[0])], form)
	}
	for shape, forms := range byShape {
		if len(forms) < 2 {
			continue
		}
		for i := 1; i < len(forms); i++ {
			a, b := RampOf(forms[0]), RampOf(forms[i])
			if a == b && Tier(forms[0]) == Tier(forms[i]) {
				t.Errorf("%s and %s share the mark %q, the same ramp and the same rung",
					forms[0], forms[i], shape)
			}
		}
	}
}

// A form nobody knows flies like the larva, the same fallback the other three
// painters make.
func TestATinyFormNobodyKnowsIsTheLarva(t *testing.T) {
	for _, form := range []string{"", "nonsense", "SPARK"} {
		if got, want := DrawTiny(form, Vitals[0], 0, false), DrawTiny(Root, Vitals[0], 0, false); got != want {
			t.Errorf("DrawTiny(%q) = %q, want the larva's %q", form, got, want)
		}
	}
}
