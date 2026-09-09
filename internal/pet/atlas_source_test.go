package pet

import (
	"encoding/json"
	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"os"
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The atlas, as the source it claims to be.
//
// sprites.go and ramps.go were generated from the design canvas rather than
// typed, and until now that was a claim in a comment: the file they came from
// lived in a scratch directory under /tmp belonging to a session that had
// already ended. One tmpreaper away, "generated, not typed" would have been
// unprovable and 41 silhouettes x 7 states would have become 287 magic strings.
//
// testdata/ATLAS.json is that file, and these turn the claim into a check.
// There is an older TestSpritesMatchTheCanvasRowByRow that does the same
// against the canvas HTML, but it needs CANVAS= pointing at a file outside the
// repo and skips without it. This one always runs.

type atlasForm struct {
	Name   string              `json:"name"`
	Parent string              `json:"parent"`
	Note   string              `json:"note"`
	Base   string              `json:"base"`
	Ramp   []string            `json:"ramp"`
	States map[string][]string `json:"states"`
}

func loadAtlas(t *testing.T) []atlasForm {
	t.Helper()
	raw, err := os.ReadFile("testdata/ATLAS.json")
	if err != nil {
		t.Fatalf("no se pudo leer el atlas: %v", err)
	}
	var forms []atlasForm
	if err := json.Unmarshal(raw, &forms); err != nil {
		t.Fatalf("el atlas no es JSON valido: %v", err)
	}
	if len(forms) != 41 {
		t.Fatalf("el atlas trae %d formas, se esperaban 41", len(forms))
	}
	return forms
}

// idOf inverts Name: the atlas writes the canvas's Spanish, pet.json stores the
// English ids.
func idOf(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for id := range Sprites {
		out[Name(id)] = id
	}
	return out
}

func TestTheAtlasNamesTheSameFortyOneFormsTheCodeDoes(t *testing.T) {
	ids := idOf(t)
	seen := map[string]bool{}
	for _, f := range loadAtlas(t) {
		id, ok := ids[f.Name]
		if !ok {
			t.Errorf("el atlas trae %q y el codigo no lo conoce", f.Name)
			continue
		}
		seen[id] = true
	}
	for id := range Sprites {
		if !seen[id] {
			t.Errorf("%s esta en el codigo y no en el atlas", id)
		}
	}
}

// Every colour in ramps.go came off the atlas, so every colour has to still be
// there. This is the whole of the "colour belongs to the branch" scheme: ten
// ramps, seven steps, checked against the file they were read from.
func TestEveryRampStepMatchesTheAtlas(t *testing.T) {
	ids := idOf(t)
	for _, f := range loadAtlas(t) {
		id := ids[f.Name]
		if id == "" {
			continue
		}
		ramp := RampOf(id)
		if len(f.Ramp) != len(ramp.Body) {
			t.Errorf("%s: el atlas da %d colores y la rampa tiene %d",
				f.Name, len(f.Ramp), len(ramp.Body))
			continue
		}
		for step, want := range f.Ramp {
			if got := ramp.Body[step]; got != theme.Hex(want) {
				t.Errorf("%s peldano %d: el codigo pinta #%02x%02x%02x y el atlas dice %s",
					f.Name, step, got.R, got.G, got.B, want)
			}
		}
	}
}

// And the silhouettes, row by row, all 287 of them: the promise the atlas's own
// header makes - "287 variantes, sin dos iguales" - checked against the engine
// that draws them rather than against itself.
//
// The feet alternate: a walking state draws one half of the step or the other
// depending on the second, and the atlas caught whichever phase it caught. Both
// are the same sprite, so both are accepted; every other row must be exact.
func TestEverySpriteIsTheOneTheAtlasDrew(t *testing.T) {
	ids := idOf(t)
	checked := 0
	for _, f := range loadAtlas(t) {
		id := ids[f.Name]
		if id == "" {
			continue
		}
		for _, v := range Vitals {
			want, ok := f.States[NameIn(i18n.ES, v.Label)]
			if !ok {
				t.Errorf("%s: el atlas no trae el estado %q", f.Name, NameIn(i18n.ES, v.Label))
				continue
			}
			checked++
			if !drawsAs(id, v, want) {
				t.Errorf("%s/%s no coincide con el atlas:\n  atlas  %q\n  codigo %q",
					f.Name, NameIn(i18n.ES, v.Label), strings.Join(want, "|"),
					strings.Join(drawn(id, v, 0), "|"))
			}
		}
	}
	if checked != 287 {
		t.Errorf("se compararon %d variantes, el atlas promete 287", checked)
	}
}

func drawn(id string, v Vital, step int) []string {
	rows := Draw(id, v, step, false)
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = theme.Strip(row)
	}
	return out
}

func drawsAs(id string, v Vital, want []string) bool {
	for _, step := range []int{0, 1} {
		if strings.Join(drawn(id, v, step), "|") == strings.Join(want, "|") {
			return true
		}
	}
	return false
}

// The atlas also carries the tree, and it must be the tree the code walks.
func TestTheAtlasParentsAreTheCodesTree(t *testing.T) {
	ids := idOf(t)
	for _, f := range loadAtlas(t) {
		id := ids[f.Name]
		if id == "" || f.Parent == "—" {
			continue
		}
		want := ids[f.Parent]
		if want == "" {
			// The two secrets hang off no branch the tree knows.
			if isSecret(id) {
				continue
			}
			t.Errorf("%s: el atlas lo cuelga de %q, que no es una forma", f.Name, f.Parent)
			continue
		}
		if got := Parent[id]; got != want {
			t.Errorf("%s: el atlas lo cuelga de %s y el codigo de %s", id, want, got)
		}
	}
}
