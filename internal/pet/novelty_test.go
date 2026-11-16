package pet

import "testing"

// The album weighs on the forks: a form worn in an earlier life has to be
// pushed harder to be worn again. See SeenWeight.

// secondLife is a pet in generation gen whose earlier lives each walked the
// given path, standing at a fresh stage with no habits yet.
func secondLife(gen int, path ...string) *State {
	s := New()
	s.Generation = gen
	for g := 1; g < gen; g++ {
		for _, form := range path {
			s.Album = append(s.Album, AlbumEntry{Form: form, Gen: g})
		}
	}
	s.NewStage()
	return s
}

func TestTheSameHabitsDoNotWalkTheSamePathForEver(t *testing.T) {
	// Two days of commits against a day and a half of green suites: pauta
	// without an album, and pauta in the first life. In the second it has been
	// worn, and the closer habit gets its turn.
	s := secondLife(1)
	s.Bump("methodical", 20)
	s.Bump("inquisitive", 15)
	if got := pick(s, Root); got != "pattern" {
		t.Fatalf("first life went to %s, want pattern", got)
	}
	s = secondLife(2, "spark", "pattern", "refactor", "weaver", "loom")
	s.Bump("methodical", 20)
	s.Bump("inquisitive", 15)
	if got := pick(s, Root); got != "probe" {
		t.Errorf("second life went to %s again, want probe", got)
	}
}

func TestAStrongHabitStillDecides(t *testing.T) {
	// The weight is a handicap, not a ban: ten days against one is still pauta.
	s := secondLife(2, "spark", "pattern", "refactor", "weaver", "loom")
	s.Bump("methodical", 100)
	s.Bump("inquisitive", 10)
	if got := pick(s, Root); got != "pattern" {
		t.Errorf("went to %s, want pattern: 50 weighed days against 1", got)
	}
}

func TestEveryRepeatCostsHalfAgain(t *testing.T) {
	// The numbers of the pet this was written for, taken as one stage:
	// methodical 524, inquisitive 83. Pauta holds for three lives and then the
	// sonda gets in: 52.4 -> 26.2 -> 13.1 -> 6.55 against 8.3.
	for gen, want := range map[int]string{1: "pattern", 2: "pattern", 3: "pattern", 4: "probe"} {
		s := secondLife(gen, "spark", "pattern")
		s.Bump("methodical", 524)
		s.Bump("inquisitive", 83)
		if got := pick(s, Root); got != want {
			t.Errorf("generation %d went to %s, want %s", gen, got, want)
		}
	}
}

func TestAStageWithNoHabitWalksTheTree(t *testing.T) {
	// The route by default used to be the first child, every life.
	s := secondLife(2, "spark", "pattern", "refactor", "surgeon", "scalpel")
	if got := pick(s, Root); got != "probe" {
		t.Errorf("no habit, pauta already worn: went to %s, want probe", got)
	}
	if got := pick(s, "refactor"); got != "weaver" {
		t.Errorf("no habit, cirujano already worn: went to %s, want weaver", got)
	}
	// And with nothing worn it is still the design's first.
	if got := pick(secondLife(1), Root); got != "pattern" {
		t.Errorf("no habit, no album: went to %s, want pattern", got)
	}
}

func TestThisLifesFormsAreNotHeldAgainstIt(t *testing.T) {
	s := secondLife(1)
	s.Album = append(s.Album, AlbumEntry{Form: "pattern", Gen: 1})
	if w := novelty(s, "pattern"); w != 1 {
		t.Errorf("a form of the current life weighs %.2f, want 1", w)
	}
}

func TestTheBarPointsWhereTheWalkGoes(t *testing.T) {
	// NextMark and pick read the same race, album included: the bar must not
	// promise the mark the walk is about to refuse.
	s := secondLife(2, "spark", "pattern", "refactor", "weaver", "loom")
	for _, mark := range Tree["refactor"] {
		u := Unlocks[mark]
		if maxCounters[u.Counter] {
			s.RecordMax(u.Counter, s.Counters[u.Counter]+u.Threshold)
		} else {
			s.Bump(u.Counter, u.Threshold)
		}
	}
	next, ok := NextMark(s, "refactor")
	if !ok || next.Form != pick(s, "refactor") {
		t.Errorf("the bar says %q and the walk %q", next.Form, pick(s, "refactor"))
	}
	if next.Form != "surgeon" {
		t.Errorf("both habits paid in full and tejedor already worn: %s, want surgeon", next.Form)
	}
}
