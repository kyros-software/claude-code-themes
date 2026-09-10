package invaders

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// levels is the ladder, read off pet.Levels rather than written down.
func levels() int { return len(pet.Levels) }

// A form that plays like another is a form nobody has a reason to want, and the
// whole premise of the game is that the creature you happen to have changes how
// it plays. So: forty-one forms, six levels, no two alike.
//
// Four collisions had to be broken to get here, and three of them are in the
// design's own tables - transcribe them literally and this test fails on the
// spot:
//
//   - bloodhound adds homing to a bughunter that already homes, so it also
//     takes a point of pierce.
//   - cartographer adds a turret ability to an architect whose ability already
//     is a turret, so it drops two.
//   - the gremlin's "random damage spikes" had no field in the Kit at all, so
//     without Spike a gremlin was a bare feral.
//
// The fourth is not in the tables, it is in the ladder: subtracting from cadence
// per level piles every fast family onto the same floor, and a bolt at level 6
// came out identical to a bare sprinter. Cadence is multiplied instead.
func TestEveryOneOfTheFortyOneFormsFliesDifferently(t *testing.T) {
	if len(pet.Sprites) != 41 {
		t.Fatalf("the atlas is 41 forms, there are %d", len(pet.Sprites))
	}
	for level := 1; level <= levels(); level++ {
		seen := map[shape]string{}
		for form := range pet.Sprites {
			k := KitFor(form, level)
			if other, dup := seen[k.shape()]; dup {
				t.Errorf("level %d: %s flies exactly like %s: %+v",
					level, form, other, k.shape())
				continue
			}
			seen[k.shape()] = form
		}
		if len(seen) != len(pet.Sprites) {
			t.Errorf("level %d: %d distinct kits for %d forms", level, len(seen), len(pet.Sprites))
		}
	}
}

// A level is a reward. Nothing about a form may get worse when the pet grows,
// and something has to get better - a level that changes nothing is a level the
// player cannot feel.
func TestNoLevelEverMakesAFormWeaker(t *testing.T) {
	for form := range pet.Sprites {
		for level := 2; level <= levels(); level++ {
			was, now := KitFor(form, level-1), KitFor(form, level)

			for _, c := range []struct {
				name     string
				was, now int
			}{
				{"damage", was.Damage, now.Damage},
				{"shots", was.Shots, now.Shots},
				{"pierce", was.Pierce, now.Pierce},
				{"splash", was.Splash, now.Splash},
				{"spike", was.Spike, now.Spike},
				{"max hp", was.MaxHP, now.MaxHP},
				{"regen", was.Regen, now.Regen},
				{"magazine", was.Cap, now.Cap},
			} {
				if c.now < c.was {
					t.Errorf("%s at level %d lost %s: %d, was %d",
						form, level, c.name, c.now, c.was)
				}
			}
			if now.Cadence > was.Cadence {
				t.Errorf("%s at level %d fires slower: cadence %d, was %d",
					form, level, now.Cadence, was.Cadence)
			}
			// NOT the reload on its own: a bigger magazine takes longer to
			// fill, and that is not a worse gun. What may never get worse is the
			// SUSTAINED rate - rounds per tick counting the reload - which is
			// the number the player actually feels.
			if sustained(now) < sustained(was) {
				t.Errorf("%s at level %d fires slower over a magazine: %g, was %g",
					form, level, sustained(now), sustained(was))
			}
			if now.Cooldown > was.Cooldown {
				t.Errorf("%s at level %d waits longer for its ability: %d, was %d",
					form, level, now.Cooldown, was.Cooldown)
			}
			if now.Homing != was.Homing || now.Overload != was.Overload || now.Special != was.Special {
				t.Errorf("%s at level %d changed what it is, not how strong it is", form, level)
			}
			if was.shape() == now.shape() {
				t.Errorf("%s at level %d is the same kit as at %d: the level buys nothing",
					form, level, level-1)
			}
		}
	}
}

// A title is its mark one tier up, and "up" has to mean up in both directions:
// the numbers that want to be big grow and the two that want to be small shrink.
// Reading "every number x1.5" literally would hand a title a slower gun than its
// mark, which is the opposite of a tier.
func TestATitleIsItsMarkOneTierUp(t *testing.T) {
	for mark, title := range pet.Titles {
		for level := 1; level <= levels(); level++ {
			m, ti := KitFor(mark, level), KitFor(title, level)

			if ti.Damage < m.Damage || ti.Shots < m.Shots || ti.Pierce < m.Pierce ||
				ti.MaxHP < m.MaxHP || ti.Regen < m.Regen || ti.Splash < m.Splash {
				t.Errorf("%s is weaker than %s at level %d:\n%+v\n%+v",
					title, mark, level, ti, m)
			}
			if ti.Cadence > m.Cadence {
				t.Errorf("%s fires slower than %s at level %d: %d vs %d",
					title, mark, level, ti.Cadence, m.Cadence)
			}
			if ti.Cooldown > m.Cooldown {
				t.Errorf("%s waits longer than %s at level %d: %d vs %d",
					title, mark, level, ti.Cooldown, m.Cooldown)
			}
			if ti.shape() == m.shape() {
				t.Errorf("%s flies exactly like %s at level %d", title, mark, level)
			}
		}
	}
}

// The family anchor is the rung-3 trade where there is one, and the form itself
// where the lineage does not reach that far. pet.tradeOf answers this question
// for the tree's own purposes and answers "" for the root, the three rung-2
// forms and both secrets - all six of which need a weapon here.
func TestTheFamilyAnchorIsTheTradeWhereThereIsOne(t *testing.T) {
	for _, c := range []struct{ form, want string }{
		{"spark", "spark"},
		{"pattern", "pattern"},
		{"probe", "probe"},
		{"ember", "ember"},
		{"refactor", "refactor"},
		{"surgeon", "refactor"},
		{"scalpel", "refactor"},
		{"kraken", "feral"},
		{"leviathan", "feral"},
		{"phoenix", "phoenix"},
		{"chimera", "chimera"},
	} {
		if got := anchor(c.form); got != c.want {
			t.Errorf("anchor(%q) = %q, want %q", c.form, got, c.want)
		}
	}
}

// Every mark on the tree has to do something, or two of its trade's children
// fly the same way. Driven off pet.Unlocks so a fifteenth mark cannot be added
// to the tree without one here.
func TestEveryMarkOnTheTreeModifiesSomething(t *testing.T) {
	for mark := range pet.Unlocks {
		change, ok := marks[mark]
		if !ok {
			t.Errorf("the mark %q has no modifier", mark)
			continue
		}
		base := families[anchor(mark)]
		if change(base).shape() == base.shape() {
			t.Errorf("the mark %q leaves its family exactly as it was", mark)
		}
	}
	for mark := range marks {
		if _, ok := pet.Unlocks[mark]; !ok {
			t.Errorf("%q has a modifier and is not a mark on the tree", mark)
		}
	}
}

// A hand-edited pet.json, or a form renamed in some future atlas, must not be a
// crash or a ship with no gun. Same fallback pet.Draw makes for a sprite it does
// not know.
func TestAFormNobodyKnowsFliesLikeTheLarva(t *testing.T) {
	for _, form := range []string{"", "nonsense", "SPARK", "../../etc/passwd"} {
		got := KitFor(form, 3)
		if want := KitFor(pet.Root, 3); got != want {
			t.Errorf("KitFor(%q) = %+v, want the larva's %+v", form, got, want)
		}
	}
}

// A level off the end of the ladder is clamped rather than indexed, because the
// level comes from LevelFor and a corrupt XP is somebody else's bug to survive.
func TestALevelOffTheLadderIsClamped(t *testing.T) {
	for _, level := range []int{-5, 0} {
		if got, want := KitFor("wasp", level), KitFor("wasp", 1); got != want {
			t.Errorf("level %d gave %+v, want level 1's", level, got)
		}
	}
	for _, level := range []int{7, 99} {
		if got, want := KitFor("wasp", level), KitFor("wasp", levels()); got != want {
			t.Errorf("level %d gave %+v, want the top of the ladder", level, got)
		}
	}
}

// Every family the forty-one forms can produce needs a name in both languages,
// or the HUD prints a blank where the weapon goes in exactly one of them.
func TestEveryFamilyHasANameInBothLanguages(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		names := i18n.G().Families
		for form := range pet.Sprites {
			family := KitFor(form, 1).Family
			if names[family] == "" {
				t.Errorf("%s: the family %q of %s has no name", lang, family, form)
			}
		}
	}
	i18n.Use("")
}

// Every ability a kit can name has to be one the game knows how to run. A typo
// in a mark's modifier would otherwise be a space bar that silently does
// nothing for one form out of forty-one.
func TestEveryAbilityAKitAsksForIsOneThatExists(t *testing.T) {
	known := map[string]bool{
		AbilityVolley: true, AbilitySweep: true, AbilityTurret: true,
		AbilityTurret2: true, AbilityInvuln: true, AbilityThree: true,
		AbilityBlast: true, AbilityChimera: true,
	}
	for form := range pet.Sprites {
		for level := 1; level <= levels(); level++ {
			if s := KitFor(form, level).Special; !known[s] {
				t.Errorf("%s at level %d asks for the ability %q", form, level, s)
			}
		}
	}
}

// One line, and it is the one the distinctness test rests on: the day somebody
// gives Kit a slice field, this stops compiling instead of quietly turning
// map[shape]string into a panic.
func TestAKitIsUsableAsAMapKey(t *testing.T) {
	var _ = map[Kit]bool{}
	var _ = map[shape]string{}
}

// The phoenix is the only form that revives, and it is the ability that says so.
// Nothing else may claim it, or the once-per-run rule has two owners.
func TestOnlyThePhoenixCarriesTheRevival(t *testing.T) {
	for form := range pet.Sprites {
		revives := KitFor(form, 5).Revive
		if revives != (form == "phoenix") {
			t.Errorf("%s revives = %v", form, revives)
		}
	}
}

// The magazine is part of the kit, so it has to be sane for all forty-one at
// every level: a gun with no rounds cannot fire and a reload of zero is a gun
// that never has to stop.
func TestEveryFormHasAMagazineItCanEmptyAndFill(t *testing.T) {
	for form := range pet.Sprites {
		for level := 1; level <= levels(); level++ {
			k := KitFor(form, level)
			if k.Cap < 4 {
				t.Errorf("%s at level %d carries %d rounds", form, level, k.Cap)
			}
			if k.Reload < 20 {
				t.Errorf("%s at level %d reloads in %d ticks, which is no cost at all",
					form, level, k.Reload)
			}
			// A magazine has to outlast the cadence, or the gun spends its life
			// reloading and the cadence stops meaning anything.
			if k.Cap*k.Cadence < k.Reload {
				t.Errorf("%s at level %d empties in %d ticks and reloads in %d",
					form, level, k.Cap*k.Cadence, k.Reload)
			}
		}
	}
}

// sustained is rounds per tick over a whole magazine and the reload after it.
func sustained(k Kit) float64 {
	return float64(k.Cap) / float64(k.Cap*k.Cadence+k.Reload)
}
