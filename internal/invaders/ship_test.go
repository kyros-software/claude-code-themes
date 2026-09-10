package invaders

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// Every family flies something, and the something is the right size. A family
// with no hull would fly the larva's, which is a form quietly pretending to be
// another one.
func TestEveryWeaponFamilyHasAHullOfItsOwn(t *testing.T) {
	for anchor, kit := range families {
		if _, ok := Hulls[kit.Family]; !ok {
			t.Errorf("the %s family (off %s) has no ship drawn for it", kit.Family, anchor)
		}
	}
	if len(Hulls) != len(families) {
		t.Errorf("%d hulls for %d families: one of them is drawn and flown by nobody",
			len(Hulls), len(families))
	}
}

func TestEveryHullIsFiveCellsAndThreeRows(t *testing.T) {
	for family, h := range Hulls {
		for _, row := range []string{h.Crest, h.Tail} {
			if got := utf8.RuneCountInString(row); got != ShipCols {
				t.Errorf("%s has a row of %d cells and the ship is %d: %q",
					family, got, ShipCols, row)
			}
		}
		for _, side := range []string{h.Left, h.Right} {
			if utf8.RuneCountInString(side) != 1 {
				t.Errorf("%s has a bracket of %q, which is not one cell", family, side)
			}
		}
		art := ShipArt(family, pet.Vitals[0])
		if len(art) != ShipRows {
			t.Errorf("%s draws %d rows", family, len(art))
		}
		for i, row := range art {
			if got := utf8.RuneCountInString(row); got != ShipCols {
				t.Errorf("%s row %d is %d cells: %q", family, i, got, row)
			}
		}
	}
}

// Thirteen families, thirteen silhouettes. The colour says which creature you
// are and the shape says what it shoots with, so two families that draw the same
// ship are two guns the player cannot tell apart at a glance.
func TestNoTwoFamiliesFlyTheSameSilhouette(t *testing.T) {
	seen := map[string]string{}
	for family := range Hulls {
		art := ShipArt(family, pet.Vitals[0])
		key := strings.Join(art[:], "\n")
		if other, dup := seen[key]; dup {
			t.Errorf("%s and %s fly the same ship", family, other)
		}
		seen[key] = family
	}
}

// The eyes are the health, which is the one piece of the creature's own
// behaviour the representation keeps: it droops as it is worn and its eyes go
// out when the run is over.
func TestTheEyesGoOutAsTheShipIsWornDown(t *testing.T) {
	fresh := ShipArt("homing", pet.Vitals[0])[1]
	tired := ShipArt("homing", pet.Vitals[4])[1]
	ko := ShipArt("homing", pet.Vitals[len(pet.Vitals)-1])[1]

	if !strings.Contains(fresh, "o o") {
		t.Errorf("a fresh ship shows %q", fresh)
	}
	if !strings.Contains(tired, "- -") {
		t.Errorf("a worn ship shows %q", tired)
	}
	if !strings.Contains(ko, "x x") {
		t.Errorf("a dead ship shows %q", ko)
	}
	if fresh == tired || tired == ko {
		t.Error("the three states are not three different faces")
	}
}

// A family nobody has heard of still draws something, the way pet.Draw falls
// back to the larva: a run must never fail to paint because a table has a gap.
func TestAFamilyNobodyKnowsFliesTheLarvasHull(t *testing.T) {
	got := ShipArt("no-such-family", pet.Vitals[0])
	want := ShipArt("single", pet.Vitals[0])
	if got != want {
		t.Errorf("an unknown family drew %q, want the larva's %q", got, want)
	}
}

// The hitbox is the middle row and the three middle cells of the other two: the
// tips of the antennae do not kill you.
func TestTheAntennaeAreNotPartOfTheHitbox(t *testing.T) {
	g := aGame(t, "bughunter", 4)
	g.Ship = 20
	row := float64(g.Field.ShipRow())

	for _, c := range []struct {
		name string
		x, y float64
		want bool
	}{
		{"the middle of the hull", 22, row + 1, true},
		{"the left bracket", 20, row + 1, true},
		{"the right bracket", 24, row + 1, true},
		{"the tip of the left antenna", 20, row, false},
		{"the tip of the right antenna", 24, row, false},
		{"the middle of the crest", 22, row, true},
		{"the corner of the tail", 24, row + 2, false},
		{"just left of the ship", 19, row + 1, false},
		{"a row above the ship", 22, row - 1, false},
	} {
		if got := g.hitsShip(c.x, c.y); got != c.want {
			t.Errorf("%s at %g,%g: hit=%v, want %v", c.name, c.x, c.y, got, c.want)
		}
	}
}
