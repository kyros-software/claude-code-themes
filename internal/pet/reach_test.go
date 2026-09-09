package pet

import (
	"math"
	"os"
	"strconv"
	"testing"
)

// Reachability. Not "does the tree contain the form" - the tree obviously does
// - but "can a pet that has already been played for months still GET there".
//
// The two are not the same question, and the difference is where the tree lost
// eight of its forty-one forms. Every counter that decides a fork only ever
// goes up, so any rule that reads "the first sibling to cross its line wins"
// hands that branch to whichever habit you happened to do first and never gives
// it back. The forms stayed in the map and stopped being reachable, which is
// the worst shape a bug can have: nothing is missing, nothing errors, the pet
// just quietly cannot become four of the things it advertises.
//
// So these tests start from a VETERAN - a pet with every counter already far
// past every threshold, the state most hostile to a late change of direction -
// and require that each of the forty-one is still reachable from there by
// raising counters only. Never by lowering one, because playing cannot lower
// one.

// xpFor is the XP that opens a level, read off Levels rather than written
// down. Every test that wants "a pet at level 5" used to spell 900 out, so
// widening the ladder meant chasing the number through thirty-odd literals -
// and any one of them missed is a test that quietly asserts nothing.
func xpFor(level int) int {
	best := 0
	for _, l := range Levels {
		if l.Level <= level && l.XP > best {
			best = l.XP
		}
	}
	return best
}

// veteran is a pet that has done everything, a lot. Under the old rule this
// state could only ever be seven of the fourteen marks.
func veteran() *State {
	s := &State{XP: xpFor(6), Counters: map[string]int{}}
	for _, c := range BranchBy {
		s.Counters[c] = 500
	}
	for _, u := range Unlocks {
		s.Counters[u.Counter] = u.Threshold * 10
	}
	return s
}

// tierXP is the XP a form's own tier sits at: a spark is level 1 and a title is
// level 6, so asking whether a spark is "reachable" at xpFor(6) XP is asking the
// wrong question.
func tierXP(form string) int {
	switch {
	case form == Root:
		return 0
	case isSecret(form):
		return xpFor(5)
	case isTitle(form):
		return xpFor(6)
	}
	if _, mark := Unlocks[form]; mark {
		return xpFor(5)
	}
	depth := 0
	for f, ok := Parent[form]; ok; f, ok = Parent[f] {
		depth++
	}
	if depth == 1 {
		return 60
	}
	return 180
}

func isSecret(form string) bool {
	for _, s := range Secrets {
		if form == s {
			return true
		}
	}
	return false
}

func isTitle(form string) bool {
	mark, ok := Parent[form]
	return ok && Titles[mark] == form
}

// play raises the counters a veteran would have to raise to end up as form,
// and returns the state it leaves behind. It only ever assigns a value HIGHER
// than the one already there - playCheck enforces that - because that is the
// only move a person actually has.
func play(t *testing.T, form string) *State {
	t.Helper()
	s := veteran()
	s.XP = tierXP(form)

	if isSecret(form) {
		s.Secret = Secret(form)
		return s
	}

	path := []string{form}
	for p, ok := Parent[path[0]]; ok; p, ok = Parent[path[0]] {
		path = append([]string{p}, path...)
	}

	for _, node := range path {
		switch {
		// A fork decided by a counter race: out-work every sibling, in days
		// of the habit rather than in raw counts. See outrank.
		case BranchBy[node] != "":
			raise(t, s, BranchBy[node], outrank(s, node))

		// A title: the same habit as its mark, at twice the bar. Getting the
		// title means also winning the mark's fork on the way past, and both
		// read the same counter, so one raise settles both.
		case isTitle(node):
			mark := Parent[node]
			u, ok := TitleUnlock(mark)
			if !ok {
				t.Fatalf("%s: el titulo no tiene umbral", node)
			}
			raise(t, s, u.Counter, needed(s, mark, u.Threshold))

		// A mark: be the ripest of its siblings, and clear your own bar.
		default:
			u, ok := Unlocks[node]
			if !ok {
				continue
			}
			raise(t, s, u.Counter, needed(s, node, u.Threshold))
		}
	}
	return s
}

// needed is the value that makes mark the ripest of its siblings while also
// clearing floor - the least a person would have to do, rounded up.
func needed(s *State, mark string, floor int) int {
	u := Unlocks[mark]
	top := 0.0
	for _, sib := range Tree[Parent[mark]] {
		if sib == mark {
			continue
		}
		su, ok := Unlocks[sib]
		if !ok || su.Threshold <= 0 {
			continue
		}
		if r := float64(s.Counters[su.Counter]) / float64(su.Threshold); r > top {
			top = r
		}
	}
	want := int(math.Floor(top*float64(u.Threshold))) + 1
	if want < floor {
		want = floor
	}
	return want
}

// raise is the whole point: a counter may go up and may not go down.
func raise(t *testing.T, s *State, counter string, to int) {
	t.Helper()
	if to < s.Counters[counter] {
		t.Fatalf("%s tendria que BAJAR de %d a %d, y jugando no se baja",
			counter, s.Counters[counter], to)
	}
	s.Counters[counter] = to
}

// outrank is the value that wins node's fork: enough of node's own counter to
// pass the ripest sibling once every counter is divided by its scale.
//
// It used to be "one more than the highest sibling", which was the same
// arithmetic topBranch did and stopped being true when topBranch started
// dividing. A helper that models the old rule tests the old rule.
//
// Never below what the counter already holds, because playing cannot lower one
// and raise would refuse it - and a value already above every sibling is
// already a winner.
func outrank(s *State, node string) int {
	parent := Parent[node]
	top := 0.0
	for _, sib := range Tree[parent] {
		if sib == node {
			continue
		}
		if share := branchShare(s, sib); share > top {
			top = share
		}
	}
	// A fork with somebody standing on it costs the margin on top, which is
	// what topBranch asks of a challenger. Without this every test that steers
	// a pet across a defended fork would be steering it nowhere.
	if held, ok := s.Branch[parent]; ok && held != node {
		if share := branchShare(s, held) + BranchMargin; share > top {
			top = share
		}
	}
	scale := BranchScale[BranchBy[node]]
	if scale <= 0 {
		scale = 1
	}
	want := int(math.Floor(top*float64(scale))) + 1
	if have := s.Counters[BranchBy[node]]; have > want {
		want = have
	}
	return want
}

// steer sets the counter races along the way so the walk from the root
// actually arrives at node. Without it every synthetic state drifts to
// whatever sibling happens to be first in the map, which says nothing about
// the fork under test.
func steer(s *State, node string) {
	for f, ok := node, true; ok; f, ok = Parent[f] {
		if c := BranchBy[f]; c != "" {
			s.Counters[c] = outrank(s, f)
		}
	}
}

// Every fork has to have a scale, or topBranch silently divides that one
// branch by one and the race is raw again for exactly the sibling nobody
// thought about. This is the guard for a fork added later.
func TestEveryForkCounterHasAScale(t *testing.T) {
	for form, counter := range BranchBy {
		if BranchScale[counter] <= 0 {
			t.Errorf("%s se bifurca por %q y ese contador no tiene escala",
				form, counter)
		}
	}
}

// --- the defended fork ------------------------------------------------------

// habitDays is a counter value expressed in what BranchScale says a day of
// that habit is, which is the unit the margin is written in.
func habitDays(counter string, n float64) int {
	return int(n * float64(BranchScale[counter]))
}

// A fork already taken is not handed over to whoever creeps a nose ahead. That
// nose was the whole complaint: methodical 4.60 against inquisitive 4.38, two
// suites one way and three commits the other, and the pet changed sprite twice
// in an afternoon over a fork nobody had really won.
func TestADefendedForkResistsANarrowLead(t *testing.T) {
	s := &State{XP: xpFor(2), Branch: map[string]string{Root: "pattern"},
		Counters: map[string]int{
			"methodical":  habitDays("methodical", 4),
			"inquisitive": habitDays("inquisitive", 4.875), // ahead, by less than a day
		}}
	if got := topBranch(s, Root); got != "pattern" {
		t.Errorf("una ventaja de 0,875 días se llevó la rama: %s", got)
	}
}

// And it IS handed over once the lead is a day of the habit, which is what
// makes the margin a delay and not a lock.
func TestADefendedForkYieldsToAClearLead(t *testing.T) {
	s := &State{XP: xpFor(2), Branch: map[string]string{Root: "pattern"},
		Counters: map[string]int{
			"methodical":  habitDays("methodical", 4),
			"inquisitive": habitDays("inquisitive", 5), // exactly the margin
		}}
	if got := topBranch(s, Root); got != "probe" {
		t.Errorf("un día entero de ventaja no bastó: %s", got)
	}
}

// The margin must not amputate the tree, which is the failure this whole file
// exists to catch: a defended fork is slower to take, never impossible. Every
// side of every counter fork, against every possible defender.
func TestADefendedForkCanStillBeTaken(t *testing.T) {
	for parent, kids := range Tree {
		if BranchBy[kids[0]] == "" {
			continue // a mark fork: ripestMark decides those, undefended
		}
		for _, held := range kids {
			for _, want := range kids {
				s := veteran()
				s.Branch = map[string]string{parent: held}
				steer(s, want)
				if got := topBranch(s, parent); got != want {
					t.Errorf("%s: con %s defendiendo, %s es inalcanzable (sale %s)",
						parent, held, want, got)
				}
			}
		}
	}
}

// Only the forks the pet has actually crossed. Writing the level 3 fork while
// the pet is still level 2 would hand it a defender chosen a level early, off
// counters that have not decided anything yet.
func TestRememberBranchOnlyWritesForksTheP3tHasCrossed(t *testing.T) {
	s := &State{XP: xpFor(2), Counters: map[string]int{"methodical": 50, "diffs": 50}}
	RememberBranch(s)
	if s.Branch[Root] != "pattern" {
		t.Errorf("la bifurcación del nivel 2 no se anotó: %v", s.Branch)
	}
	if got, ok := s.Branch["pattern"]; ok {
		t.Errorf("anotó la del nivel 3 a nivel 2: %s", got)
	}

	s.XP = xpFor(3)
	RememberBranch(s)
	if s.Branch["pattern"] != "refactor" {
		t.Errorf("a nivel 3 no se anotó el oficio: %v", s.Branch)
	}
}

// It records what topBranch already decided, so running it again changes
// nothing - otherwise every save would nudge the fork it was meant to hold.
func TestRememberBranchIsIdempotent(t *testing.T) {
	s := &State{XP: xpFor(3), Counters: map[string]int{
		"methodical": habitDays("methodical", 4), "inquisitive": habitDays("inquisitive", 4.5),
		"diffs": 30, "ctx_low": 3}}
	RememberBranch(s)
	first := map[string]string{}
	for k, v := range s.Branch {
		first[k] = v
	}
	for range 5 {
		RememberBranch(s)
	}
	for k, v := range first {
		if s.Branch[k] != v {
			t.Errorf("%s se movió de %s a %s sin que cambiara un contador", k, v, s.Branch[k])
		}
	}
}

func TestEveryFormIsReachableFromAVeteran(t *testing.T) {
	if len(Sprites) != 41 {
		t.Fatalf("el atlas son 41 formas, hay %d", len(Sprites))
	}
	for form := range Sprites {
		s := play(t, form)
		got, level := CurrentForm(s)
		if got != form {
			t.Errorf("%s inalcanzable: con sus contadores por delante sale %s (nivel %d)",
				form, got, level)
		}
	}
}

// The pair that started this: a bughunter who has already reproduced ten bugs
// before fixing them is a bloodhound, and used to be a bloodhound for the rest
// of its life - test_streak could run to a thousand and exterminator would
// never come. Both directions have to stay open from the same state.
func TestASiblingIsNotLockedOutByTheOneEarnedFirst(t *testing.T) {
	for parent, kids := range Tree {
		if len(kids) < 2 {
			continue
		}
		if _, isMarkFork := Unlocks[kids[0]]; !isMarkFork {
			continue
		}
		for _, want := range kids {
			s := &State{XP: xpFor(5), Counters: map[string]int{}}
			// Every sibling already earned, the first one by a mile: the shape
			// the old rule froze.
			for i, kid := range kids {
				u := Unlocks[kid]
				s.Counters[u.Counter] = u.Threshold
				if i == 0 {
					s.Counters[u.Counter] = u.Threshold * 50
				}
			}
			steer(s, parent)
			u := Unlocks[want]
			s.Counters[u.Counter] = u.Threshold * 100
			if got, _ := CurrentForm(s); got != want {
				t.Errorf("%s: %s bloqueado por %s, sale %s", parent, want, kids[0], got)
			}
		}
	}
}

// The bar in band 4 names the mark the pet is closest to. It has to be the mark
// the walk would actually hand over, or the bar is pointing somewhere the pet
// cannot go - which is exactly what it did while the two used different rules.
func TestTheBarNamesTheMarkTheWalkWouldGive(t *testing.T) {
	for parent, kids := range Tree {
		if _, isMarkFork := Unlocks[kids[0]]; !isMarkFork {
			continue
		}
		for _, want := range kids {
			s := &State{XP: 400, Counters: map[string]int{}}
			for _, kid := range kids {
				u := Unlocks[kid]
				s.Counters[u.Counter] = u.Threshold / 3
			}
			u := Unlocks[want]
			s.Counters[u.Counter] = u.Threshold * 4
			steer(s, parent)

			mark, ok := NextMark(s, parent)
			if !ok {
				t.Fatalf("%s: la barra no apunta a nada", parent)
			}
			if mark.Form != want {
				t.Errorf("%s: la barra apunta a %s, se esperaba %s", parent, mark.Form, want)
			}
			// Same state, enough XP: the walk must agree with the bar.
			s.XP = xpFor(5)
			if got, _ := CurrentForm(s); got != mark.Form {
				t.Errorf("%s: la barra dice %s y la caminata da %s", parent, mark.Form, got)
			}
		}
	}
}

// A bar does not overflow, however far past the line the habit has run.
func TestTheBarNeverOverflows(t *testing.T) {
	s := veteran()
	for parent := range Tree {
		if _, isMarkFork := Unlocks[Tree[parent][0]]; !isMarkFork {
			continue
		}
		mark, ok := NextMark(s, parent)
		if !ok {
			continue
		}
		if mark.Done > mark.Threshold {
			t.Errorf("%s: la barra va %d/%d, se sale", mark.Form, mark.Done, mark.Threshold)
		}
		if mark.Share() > 1 {
			t.Errorf("%s: share %.2f", mark.Form, mark.Share())
		}
	}
}

// --- and the way back -------------------------------------------------------
//
// A shape only ever moves sideways or up. It is recomputed from the counters
// on every refresh and two of the habits - the clean test streak and the clean
// diff streak - go back to zero when the context blows, so what a pet IS can
// change from one line to the next. What it cannot do is go back DOWN the
// tree: an `exterminator` who loses the streak may become a `bloodhound`, the
// same rung by the other habit, but never a plain `bughunter` again.
//
// Without that floor the fall was dramatic and silent: a level 6 `wasp` came
// back as a level 3 `bughunter`, still labelled level 6, because the level is
// XP and XP does not fall. State.FormSeen is what holds the rung.

// streakMarks are the marks whose habit can actually go DOWN. Only the two
// clean streaks can: every other counter counts days, sessions or records, and
// those only accumulate.
func streakMarks() map[string]bool {
	drops := map[string]bool{}
	for _, counters := range streaksBrokenBy {
		for _, c := range counters {
			drops[c] = true
		}
	}
	out := map[string]bool{}
	for mark, u := range Unlocks {
		if drops[u.Counter] {
			out[mark] = true
			out[Titles[mark]] = true
		}
	}
	return out
}

func TestOnlyTwoMarksRideAHabitThatCanFall(t *testing.T) {
	// Naming them is the point: this is the blast radius of the whole idea and
	// it must not grow by accident. A new entry in streaksBrokenBy makes
	// another branch able to move, and that is a design decision, not a detail.
	want := map[string]bool{
		"exterminator": true, "wasp": true, // test_streak
		"surgeon": true, "scalpel": true, // diff_streak
	}
	got := streakMarks()
	for form := range want {
		if !got[form] {
			t.Errorf("%s deberia montar sobre una racha y no lo hace", form)
		}
	}
	for form := range got {
		if !want[form] {
			t.Errorf("%s monta sobre una racha sin decirlo", form)
		}
	}
}

// Tier is what the floor is measured in, so it has to be right for all 41
// before anything built on it means a thing.
func TestEveryFormSitsOnTheRungItIsWornAt(t *testing.T) {
	for form := range Sprites {
		want := 3
		switch {
		case form == Root:
			want = 1
		case titleForms[form]:
			want = 6
		case isSecret(form):
			want = 5
		default:
			if _, mark := Unlocks[form]; mark {
				want = 5
			} else if _, ok := Parent[Parent[form]]; !ok {
				want = 2
			}
		}
		if got := Tier(form); got != want {
			t.Errorf("%s esta en el escalon %d, se esperaba %d", form, got, want)
		}
	}
	// An unknown name holds nothing back - a hand-edited pet.json must not be
	// able to freeze the tree by naming something that does not exist.
	for _, junk := range []string{"", "cazabugs", "../../etc/passwd", "WASP"} {
		if got := Tier(junk); got != 0 {
			t.Errorf("Tier(%q) = %d, se esperaba 0", junk, got)
		}
	}
}

// The whole point, stated once: the streak falls all the way to zero and the
// shape stays on its rung.
func TestABrokenStreakNeverTakesTheShapeDownTheTree(t *testing.T) {
	s := &State{XP: xpFor(6), Counters: map[string]int{"inquisitive": 20, "tests": 20}}
	s.Counters["test_streak"] = TitleAsks["wasp"]
	form, _ := CurrentForm(s)
	if form != "wasp" {
		t.Fatalf("con la racha entera sale %s, se esperaba wasp", form)
	}
	RememberForm(s, form)

	for _, streak := range []int{TitleAsks["wasp"] - 1, 15, 14, 1, 0} {
		s.Counters["test_streak"] = streak
		got, level := CurrentForm(s)
		if Tier(got) < Tier("wasp") {
			t.Errorf("racha %d: cayo a %s, escalon %d", streak, got, Tier(got))
		}
		if level != 6 {
			t.Errorf("racha %d: el nivel bajo a %d, y el nivel es XP", streak, level)
		}
	}
}

// Sideways is still allowed, and it is the whole reason the floor is a rung
// and not the form itself: losing the test streak with the other habit earned
// is a real change and says something true about the week.
func TestTheShapeStillMovesSidewaysAlongItsRung(t *testing.T) {
	s := &State{XP: xpFor(5), Counters: map[string]int{
		"inquisitive": 20, "tests": 20,
		"test_streak":      20, // exterminator, ripest
		"repro_before_fix": 10, // bloodhound, earned
	}}
	form, _ := CurrentForm(s)
	if form != "exterminator" {
		t.Fatalf("salio %s, se esperaba exterminator", form)
	}
	RememberForm(s, form)

	s.Counters["test_streak"] = 0
	got, _ := CurrentForm(s)
	if got != "bloodhound" {
		t.Errorf("al romperse la racha sale %s, se esperaba bloodhound", got)
	}
	if Tier(got) != Tier(form) {
		t.Errorf("el movimiento lateral cambio de escalon: %d -> %d", Tier(form), Tier(got))
	}
}

// Up is allowed from anywhere on the rung, floor or no floor.
func TestTheFloorNeverBlocksTheWayUp(t *testing.T) {
	s := &State{XP: xpFor(5), Counters: map[string]int{
		"inquisitive": 20, "tests": 20, "test_streak": 20,
	}}
	form, _ := CurrentForm(s)
	RememberForm(s, form) // exterminator, rung 5

	s.XP = xpFor(6)
	s.Counters["test_streak"] = TitleAsks["wasp"]
	if got, _ := CurrentForm(s); got != "wasp" {
		t.Errorf("con el titulo ganado sale %s, se esperaba wasp", got)
	}
}

// The floor is a rung, not a memory of one particular shape: standing on 5
// must not pin the pet to the exact mark it first got there with.
func TestTheFloorRemembersTheRungAndNotTheShape(t *testing.T) {
	s := &State{Counters: map[string]int{}}
	RememberForm(s, "exterminator")
	RememberForm(s, "bloodhound") // same rung: the newer one is what is held
	if s.FormSeen != "bloodhound" {
		t.Errorf("FormSeen = %q, se esperaba bloodhound", s.FormSeen)
	}
	RememberForm(s, "bughunter") // lower rung: ignored
	if s.FormSeen != "bloodhound" {
		t.Errorf("un escalon mas bajo pisoteo el suelo: %q", s.FormSeen)
	}
	RememberForm(s, "wasp") // higher rung: taken
	if s.FormSeen != "wasp" {
		t.Errorf("FormSeen = %q, se esperaba wasp", s.FormSeen)
	}
}

// An older pet.json has no form_seen at all, and that has to be the same thing
// as no floor - the pet keeps whatever it already is and the rung starts there.
func TestAFileWithNoRungRecordedStartsFromWhereverItIs(t *testing.T) {
	s := &State{XP: xpFor(5), Counters: map[string]int{
		"inquisitive": 20, "tests": 20, "repro_before_fix": 10,
	}}
	if s.FormSeen != "" {
		t.Fatal("el estado de partida ya traia un escalon")
	}
	form, _ := CurrentForm(s)
	if form != "bloodhound" {
		t.Fatalf("salio %s, se esperaba bloodhound", form)
	}
	RememberForm(s, form)
	if s.FormSeen != "bloodhound" {
		t.Errorf("el primer refresco no anoto el escalon: %q", s.FormSeen)
	}
}

// And the twelve marks that ride a counter which only ever grows never needed
// the floor in the first place. Belt and braces: they must not move either.
func TestTheTwelveCumulativeMarksNeverMoveAtAll(t *testing.T) {
	for mark := range Unlocks {
		if streakMarks()[mark] {
			continue
		}
		s := play(t, mark)
		before, _ := CurrentForm(s)
		if before != mark {
			t.Fatalf("%s no se alcanzo para empezar, salio %s", mark, before)
		}
		RememberForm(s, before)
		for _, counters := range streaksBrokenBy {
			for _, c := range counters {
				s.Counters[c] = 0
			}
		}
		if after, _ := CurrentForm(s); after != mark {
			t.Errorf("%s se movio al romperse una racha y paso a %s", mark, after)
		}
	}
}

// The floor is worth nothing if it does not survive the file. Load reads
// pet.json field by field out of a map rather than unmarshalling the struct,
// so a new field with a json tag on it is NOT automatically read back - which
// is how form_seen was written, saved, and silently lost on the next refresh
// for as long as the tests only ever built a State by hand.
func TestTheRungSurvivesTheRoundTrip(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := New()
	s.XP = xpFor(5)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": 20}
	form, _ := CurrentForm(s)
	RememberForm(s, form)
	Save(s, path)

	back := Load(path)
	if back.FormSeen != form {
		t.Fatalf("tras ida y vuelta FormSeen = %q, se esperaba %q", back.FormSeen, form)
	}
	// And it does its job on the far side: the streak is gone and the shape
	// stays on its rung.
	back.Counters["test_streak"] = 0
	if got, _ := CurrentForm(back); Tier(got) < Tier(form) {
		t.Errorf("tras cargar del fichero cayo a %s (escalon %d)", got, Tier(got))
	}
}

// The forks go through a stricter gate than the rung: the pair has to name a
// real choice - a fork the tree has, and one of that fork's own children - or
// it defends nothing and is not worth keeping.
func TestABranchThatNamesNoRealChoiceIsDropped(t *testing.T) {
	for _, tc := range []struct {
		stored, wantKey, wantValue string
	}{
		{`{"spark":"pattern"}`, "spark", "pattern"}, // a real fork
		{`{"pattern":"tidy"}`, "pattern", "tidy"},   // and the one below it
		{`{"spark":"godzilla"}`, "", ""},            // not a child of spark
		{`{"spark":"tidy"}`, "", ""},                // a form, wrong fork
		{`{"godzilla":"pattern"}`, "", ""},          // not a fork
		{`{"../../etc/passwd":"pattern"}`, "", ""},  // not anything
		{`{"bughunter":"bloodhound"}`, "", ""},      // a MARK fork: ripestMark owns it
		{`{"chispa":"pauta"}`, "spark", "pattern"},  // v1 Spanish, both halves
	} {
		path := t.TempDir() + "/pet.json"
		if err := os.WriteFile(path,
			[]byte(`{"xp":`+strconv.Itoa(xpFor(3))+`,"branch":`+tc.stored+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got := Load(path).Branch
		if tc.wantKey == "" {
			if len(got) != 0 {
				t.Errorf("%s se guardó como %v, y no nombra ninguna elección", tc.stored, got)
			}
			continue
		}
		if got[tc.wantKey] != tc.wantValue {
			t.Errorf("%s se cargó como %v, se esperaba %s=%s",
				tc.stored, got, tc.wantKey, tc.wantValue)
		}
	}
}

// Clone exists because a shallow copy of a State shares its maps. Branch is
// the sixth one, and the panel builds a what-if pet on every /pet.
func TestCloneDoesNotShareTheBranch(t *testing.T) {
	s := New()
	s.Branch = map[string]string{Root: "pattern"}
	twin := s.Clone()
	twin.Branch[Root] = "ember"
	if s.Branch[Root] != "pattern" {
		t.Errorf("escribir en la copia movió la rama del original: %s", s.Branch[Root])
	}
}

// A v1 file names its forms in Spanish, and junk in the field is dropped
// rather than stored.
func TestTheRungIsTranslatedAndSanitisedOnTheWayIn(t *testing.T) {
	for _, tc := range []struct{ stored, want string }{
		{"exterminador", "exterminator"}, // v1 Spanish
		{"exterminator", "exterminator"}, // v2
		{"", ""},
		{"cazabugs", "bughunter"}, // v1 Spanish again
		{"../../etc/passwd", ""},  // not a form
		{"WASP", ""},              // not a form either
	} {
		path := t.TempDir() + "/pet.json"
		if err := os.WriteFile(path,
			[]byte(`{"xp":`+strconv.Itoa(xpFor(5))+`,"form_seen":`+strconv.Quote(tc.stored)+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := Load(path).FormSeen; got != tc.want {
			t.Errorf("form_seen %q se cargo como %q, se esperaba %q", tc.stored, got, tc.want)
		}
	}
}

// CheckSecrets asks which BRANCH the pet is on - `feral` or `marathon`, two
// rung 3 shapes - and it used to ask CurrentForm, which answers a different
// question: what the pet LOOKS like, floor and all. The floor can lift a shape
// above rung 3, so the two disagreed in principle.
//
// In practice they never did: every mark under those two branches rides a
// counter that only grows, so there was nothing for the floor to lift. That is
// a coincidence, not a design, and one new streak away from ending. It asks
// pet.walk now, which is the question it meant, and this proves the floor
// cannot reach it however high the rung goes.
func TestTheFloorCannotBlockThePhoenix(t *testing.T) {
	for _, branch := range []string{"feral", "marathon"} {
		s := New()
		s.XP = xpFor(5)
		s.Counters = map[string]int{BranchBy["ember"]: 99, BranchBy[branch]: 99}
		if got, _ := walk(s); got != branch {
			t.Fatalf("el camino a %s da %s", branch, got)
		}
		// Stand the pet on the highest rung there is, on a completely
		// different branch, which is the worst the floor can do.
		s.FormSeen = "wasp"
		if got, _ := CurrentForm(s); got != "wasp" {
			t.Fatalf("el suelo no esta puesto: sale %s", got)
		}
		s.Hunger, s.HungerPeak = 0, HungerMax
		CheckSecrets(s)
		if s.Secret != "phoenix" {
			t.Errorf("%s: el suelo bloqueo el fenix, secret = %q", branch, s.Secret)
		}
	}
}

// And the phoenix still comes out at the end of the path it is supposed to.
func TestThePhoenixIsStillReachableWithAFloorInPlace(t *testing.T) {
	s := New()
	s.XP = 400 // level 4: a trade, no mark yet
	s.Counters = map[string]int{"impulsive": 50, "ctx_maxed": 50}
	form, _ := CurrentForm(s)
	if form != "feral" {
		t.Fatalf("el camino de partida da %s, se esperaba feral", form)
	}
	RememberForm(s, form)

	s.Hunger, s.HungerPeak = 0, HungerMax
	CheckSecrets(s)
	if s.Secret != "phoenix" {
		t.Fatalf("el fenix no se concedio: secret = %q", s.Secret)
	}
	s.XP = xpFor(5)
	if got, _ := CurrentForm(s); got != "phoenix" {
		t.Errorf("con el fenix ganado sale %s", got)
	}
}

// The most surprising thing the rule implies, written down so it is not
// mistaken for a bug later.
//
// The temperament fork is decided by a counter race and it can flip at any
// time: commit for a week and `methodical` overtakes `inquisitive`. Before the
// pet wears a mark that flip changes the shape, which is the point of the whole
// tree. After it wears one it does NOT - the walk down the new branch lands on
// a trade, rung 3, and the floor will not go down a rung to meet it.
//
// So from the first mark onwards a change of temperament is invisible until it
// is EARNED on the new branch: get the new branch's mark and the shape moves
// sideways to it. That is the rule doing exactly what it says, and the
// alternative - dropping to a trade you already outgrew - is the thing it
// exists to prevent.
func TestChangingTemperamentDoesNotUndoAMarkUntilTheNewBranchEarnsOne(t *testing.T) {
	s := &State{XP: xpFor(5), Counters: map[string]int{
		"inquisitive": 20, "tests": 20, "test_streak": 20,
	}}
	form, _ := CurrentForm(s)
	if form != "exterminator" {
		t.Fatalf("el punto de partida es %s, se esperaba exterminator", form)
	}
	RememberForm(s, form)

	// The temperament flips outright: methodical now leads by miles.
	s.Counters["methodical"], s.Counters["diffs"] = 99, 99
	got, _ := CurrentForm(s)
	if got != "exterminator" {
		t.Errorf("el cambio de temperamento devolvio %s; la marca debe aguantar", got)
	}

	// And it moves the moment the new branch has a mark of its own.
	s.Counters["diff_streak"] = 40
	got, _ = CurrentForm(s)
	if got != "surgeon" {
		t.Errorf("con surgeon ganado sale %s", got)
	}
	if Tier(got) != 5 {
		t.Errorf("el movimiento cambio de escalon: %d", Tier(got))
	}
}

// A title used to be the end of the sideways movement: rung 6 beats rung 5, so
// a pet that had earned one stopped following its own branch until the new
// branch grew a title too - a different habit, three times over. Now the floor
// asks why the walk came out lower, and a change of branch is not a fall.
func TestATitleMovesToAnEarnedMarkOnAnotherBranch(t *testing.T) {
	s := &State{XP: xpFor(6), Counters: map[string]int{
		"inquisitive": 20, "tests": 20,
		"test_streak": TitleAsks["wasp"], // exterminator, and the title behind it
	}}
	form, _ := CurrentForm(s)
	if form != "wasp" {
		t.Fatalf("el punto de partida es %s, se esperaba wasp", form)
	}
	RememberForm(s, form)

	// The temperament flips and the new trade already has a mark paid for.
	s.Counters["methodical"], s.Counters["diffs"] = 99, 99
	s.Counters["diff_streak"] = Unlocks["surgeon"].Threshold
	got, _ := CurrentForm(s)
	if got != "surgeon" {
		t.Errorf("con surgeon ganado en la rama nueva sale %s", got)
	}
	if tradeOf(got) == tradeOf(form) {
		t.Errorf("%s y %s cuelgan del mismo oficio: el caso no prueba nada", got, form)
	}
}

// The other half of the same rule, and the one the floor was built for: inside
// ONE trade a title still does not fall to its own sibling mark. `wasp` losing
// the streak is a habit falling, not a pet moving.
func TestATitleDoesNotFallToTheOtherMarkOfItsOwnTrade(t *testing.T) {
	s := &State{XP: xpFor(6), Counters: map[string]int{
		"inquisitive": 20, "tests": 20,
		"test_streak":      TitleAsks["wasp"],
		"repro_before_fix": Unlocks["bloodhound"].Threshold,
	}}
	form, _ := CurrentForm(s)
	if form != "wasp" {
		t.Fatalf("el punto de partida es %s, se esperaba wasp", form)
	}
	RememberForm(s, form)

	s.Counters["test_streak"] = 0
	if got, _ := CurrentForm(s); got != "wasp" {
		t.Errorf("al romperse la racha cayo a %s; dentro del oficio el suelo aguanta", got)
	}
}

// Clone shares nothing. `copy := *s` looks like this and is not: five fields
// are maps and slices, and a write through one is a write through both.
func TestACloneSharesNothingWithItsOriginal(t *testing.T) {
	s := New()
	s.XP = 400
	s.Counters["tests"] = 7
	s.Meals["commit"] = 111
	s.DayMarks["mon"] = "x"
	s.Log = append(s.Log, LogEntry{Event: "tests", XP: 15})
	s.Said = append(s.Said, "hola")

	c := s.Clone()
	c.XP = xpFor(6)
	c.Counters["tests"] = 999
	c.Counters["new"] = 1
	c.Meals["commit"] = 222
	c.DayMarks["mon"] = "y"
	c.Log[0].XP = 999
	c.Said[0] = "adios"

	if s.XP != 400 {
		t.Errorf("XP del original = %d", s.XP)
	}
	if s.Counters["tests"] != 7 {
		t.Errorf("el clon escribio en los contadores del original: tests = %d", s.Counters["tests"])
	}
	if _, added := s.Counters["new"]; added {
		t.Error("el clon anadio un contador al original")
	}
	if s.Meals["commit"] != 111 {
		t.Errorf("comidas del original = %d", s.Meals["commit"])
	}
	if s.DayMarks["mon"] != "x" {
		t.Errorf("marcas del original = %q", s.DayMarks["mon"])
	}
	if s.Log[0].XP != 15 {
		t.Errorf("el log del original = %d", s.Log[0].XP)
	}
	if s.Said[0] != "hola" {
		t.Errorf("lo dicho por el original = %q", s.Said[0])
	}
}

// A clone of a newborn, and of a state loaded from an empty file, must not
// panic on nil containers.
func TestCloningAnEmptyStateIsSafe(t *testing.T) {
	for _, s := range []*State{{}, New(), Load(t.TempDir() + "/missing.json")} {
		got := s.Clone()
		got.Counters["x"] = 1
		got.Meals["y"] = 1
		got.DayMarks["z"] = "1"
		got.Log = append(got.Log, LogEntry{})
		got.Said = append(got.Said, "")
	}
}

// The shape and the level are two different facts and are allowed to disagree.
//
// Before the floor they moved together: XP fell, the level fell, the shape fell
// with it. Now the shape holds and the level still falls, so "avispa nivel 5"
// is a thing you can see - a title beside a level that could not have earned
// one. It is not an incoherence to tidy away: clamping the level up to the
// rung would cancel the hunger penalty that
// TestStarvingCanCostALevelButNeverKills exists to protect.
func TestTheShapeHoldsWhileTheLevelIsStillAllowedToFall(t *testing.T) {
	s := New()
	s.XP = xpFor(6)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": TitleAsks["wasp"]}
	form, level := CurrentForm(s)
	if form != "wasp" || level != 6 {
		t.Fatalf("el punto de partida es %s nivel %d", form, level)
	}
	RememberForm(s, form)

	// One blown context: 15 XP and the clean streak.
	s.XP -= 15
	s.Counters["test_streak"] = 0
	form, level = CurrentForm(s)
	if form != "wasp" {
		t.Errorf("la forma cayo a %s, y una forma no baja de escalon", form)
	}
	if level != 5 {
		t.Errorf("el nivel es %d; la XP cayo por debajo de xpFor(6) y el nivel la sigue", level)
	}
}

// The fourteen titles come off the canvas "Cómo llegar a cada forma", which
// gives a factor per title rather than one multiplier for all of them. Before
// that they were a uniform x2 invented here, because the atlas carried the
// titles and no condition for any of them.
//
// Two entries deliberately disagree with the canvas, and this is where the
// disagreement is written down rather than left to be rediscovered.
func TestTheTitlesAreTheCanvasNumbers(t *testing.T) {
	canvas := map[string]int{
		"scalpel": 50, "loom": 25, "abbot": 15, "forest": 7,
		"wolf": 30, "wasp": 50, "atlas": 50, "sphinx": 20,
		"storm": 30, "falcon": 25, "mammoth": 10, "worm": 20,
		"devil": 100, "leviathan": 10,
	}
	// atlas: the canvas asks for "5 planes de 10 tareas", a COUNT of big plans.
	// longest_plan is the largest single plan ever closed, so 50 there would ask
	// for one plan of fifty tasks - a different quantity, not a harder one.
	// devil: bypass_turns moves ~30 a day, so the canvas's 100 is a title that
	// arrives already earned.
	ours := map[string]int{"atlas": 20, "devil": 200}

	if len(TitleAsks) != len(canvas) {
		t.Fatalf("hay %d titulos y el lienzo trae %d", len(TitleAsks), len(canvas))
	}
	for title, want := range canvas {
		if mine, differs := ours[title]; differs {
			want = mine
		}
		if got := TitleAsks[title]; got != want {
			t.Errorf("%s pide %d, se esperaba %d", title, got, want)
		}
	}
}

// Whatever the numbers are, the chain can never invert: a title always asks
// strictly more of the habit than the mark under it.
func TestATitleAlwaysAsksMoreThanItsMark(t *testing.T) {
	for mark, title := range Titles {
		base, ok := Unlocks[mark]
		if !ok {
			t.Fatalf("%s no es una marca", mark)
		}
		u, ok := TitleUnlock(mark)
		if !ok {
			t.Fatalf("%s no tiene umbral", title)
		}
		if u.Counter != base.Counter {
			t.Errorf("%s lee %s y su marca %s", title, u.Counter, base.Counter)
		}
		if u.Threshold <= base.Threshold {
			t.Errorf("%s pide %d y %s ya pedia %d", title, u.Threshold, mark, base.Threshold)
		}
	}
}

// And every title has an entry: a missing one used to fall back to a
// multiplication and would now silently make the title unreachable.
func TestEveryTitleHasAThreshold(t *testing.T) {
	for mark, title := range Titles {
		if _, ok := TitleAsks[title]; !ok {
			t.Errorf("%s (detras de %s) no tiene umbral", title, mark)
		}
		if _, ok := TitleUnlock(mark); !ok {
			t.Errorf("TitleUnlock(%s) no devuelve nada", mark)
		}
	}
	for title := range TitleAsks {
		if !titleForms[title] {
			t.Errorf("TitleAsks trae %q, que no es un titulo", title)
		}
	}
}
