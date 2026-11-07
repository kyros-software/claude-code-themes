package pet

import "testing"

// Ahead is every shape still in front of one. The panel colours a counter by
// whether what it opens is in here: a counter feeding a branch the pet walked
// past years ago is counted and leads nowhere.
func TestAheadIsEverythingStillInFront(t *testing.T) {
	ahead := Ahead("bughunter")
	for _, want := range []string{"bloodhound", "exterminator", "wolf", "wasp"} {
		if !ahead[want] {
			t.Errorf("%s should be ahead of a bughunter", want)
		}
	}
	for _, notWant := range []string{"bughunter", "probe", "spark", "gremlin", "surgeon"} {
		if ahead[notWant] {
			t.Errorf("%s should not be ahead of a bughunter", notWant)
		}
	}
	if len(Ahead("wolf")) != 0 {
		t.Error("a title has nothing ahead of it")
	}
	// From the larva, everything: the root sees the whole tree below it.
	if got, want := len(Ahead(Root)), countForms(); got != want {
		t.Errorf("the larva sees %d shapes ahead, want the other %d", got, want)
	}
}

func TestGoalOfFindsTheNearestShapeACounterOpens(t *testing.T) {
	s := New()
	s.Counters = map[string]int{
		"repro_before_fix": 33,  // bloodhound asks 10, wolf asks 30
		"test_streak":      1,   // exterminator asks 15
		"bypass_turns":     206, // gremlin, on a branch behind a bughunter
		"widest_commit":    7,   // weaver, likewise
	}

	t.Run("the mark comes before its title", func(t *testing.T) {
		g := GoalOf(s, "bughunter", "repro_before_fix")
		if g.Form != "bloodhound" {
			t.Errorf("nearest is %q, want bloodhound", g.Form)
		}
		if g.Threshold != 10 || g.Done != 33 {
			t.Errorf("got %d/%d, want 33/10", g.Done, g.Threshold)
		}
		if !g.Reached() {
			t.Error("33 of 10 is not reported as reached")
		}
	})

	t.Run("nothing, once the mark is worn", func(t *testing.T) {
		// v3: the title comes with the XP, so a worn mark's habit leads nowhere.
		if g := GoalOf(s, "bloodhound", "repro_before_fix"); g.Leads() {
			t.Errorf("a worn mark's habit claims to lead to %q", g.Form)
		}
	})

	t.Run("on its way", func(t *testing.T) {
		g := GoalOf(s, "bughunter", "test_streak")
		if !g.Leads() || g.Reached() {
			t.Errorf("1 of 15 should lead somewhere and not be reached: %+v", g)
		}
	})

	t.Run("a counter whose shapes are all behind", func(t *testing.T) {
		for _, counter := range []string{"bypass_turns", "widest_commit"} {
			g := GoalOf(s, "bughunter", counter)
			if g.Leads() {
				t.Errorf("%s claims to lead to %q from a bughunter", counter, g.Form)
			}
			// It is still counted: this stage's number survives, only the goal is gone.
			if g.Done != s.Stage(counter) {
				t.Errorf("%s lost its count: %d", counter, g.Done)
			}
		}
	})

	t.Run("a counter nothing asks for", func(t *testing.T) {
		if g := GoalOf(s, "bughunter", "ctx_low"); g.Leads() {
			t.Errorf("ctx_low decides a fork, it does not unlock: %+v", g)
		}
	})

	t.Run("a negative count reads as zero", func(t *testing.T) {
		s := New()
		s.Counters = map[string]int{"repro_before_fix": -5}
		if g := GoalOf(s, "bughunter", "repro_before_fix"); g.Done != 0 {
			t.Errorf("got %d, want 0", g.Done)
		}
	})
}

// Siblings is the level 5 fork, and only that: a shape whose children are not
// marks has no race to report.
func TestSiblingsOnlyReportsAMarkFork(t *testing.T) {
	s := New()
	s.Counters = map[string]int{"repro_before_fix": 33, "test_streak": 1}

	sibs := Siblings(s, "bughunter")
	if len(sibs) != 2 {
		t.Fatalf("got %d siblings, want 2", len(sibs))
	}
	byForm := map[string]Sibling{}
	for _, sib := range sibs {
		byForm[sib.Form] = sib
	}
	if got := byForm["bloodhound"]; got.Done != 33 || got.Threshold != 10 || !got.Reached() {
		t.Errorf("bloodhound: %+v", got)
	}
	if got := byForm["exterminator"]; got.Done != 1 || got.Threshold != 15 || got.Reached() {
		t.Errorf("exterminator: %+v", got)
	}

	for _, form := range []string{"spark", "probe", "bloodhound", "wolf", "nonsense"} {
		if sibs := Siblings(s, form); len(sibs) != 0 {
			t.Errorf("%s has no mark fork but reported %d", form, len(sibs))
		}
	}
}

// countForms is every shape in the tree except the root itself.
func countForms() int {
	seen := map[string]bool{}
	for parent, children := range Tree {
		if parent != Root {
			seen[parent] = true
		}
		for _, c := range children {
			seen[c] = true
		}
	}
	return len(seen)
}
