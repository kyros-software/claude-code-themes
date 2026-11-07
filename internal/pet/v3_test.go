package pet

import (
	"os"
	"strconv"
	"testing"
)

// v3: a fork is decided once, when the pet crosses into it, by the habits of
// the stage that just ended. See docs/design/v3-generaciones.md.
//
// The question v2's reach_test.go asked is asked again here, harder: not "can
// a pet with every counter past every line still get there", but "can it get
// there in its NEXT life by working that way" - from a veteran whose lifetime
// counters would have settled every fork in v2 before the first commit.

// xpFor is the XP that opens a level, read off Levels.
func xpFor(level int) int {
	best := 0
	for _, l := range Levels {
		if l.Level <= level && l.XP > best {
			best = l.XP
		}
	}
	return best
}

// veteran is a pet that has done everything, a lot, and starts a fresh stage.
// Under v2 its lifetime counters decided every fork before it lifted a finger.
func veteran() *State {
	s := New()
	for _, c := range BranchBy {
		s.Counters[c] = 1000
	}
	for _, u := range Unlocks {
		s.Counters[u.Counter] = u.Threshold * 50
	}
	s.NewStage()
	return s
}

// lean plays one stage of a habit: enough of the counter that decides form to
// win its fork against siblings that do nothing this stage.
func lean(s *State, form string) {
	if c := BranchBy[form]; c != "" {
		s.Bump(c, 3*scaleOf(c))
		return
	}
	if u, ok := Unlocks[form]; ok {
		if maxCounters[u.Counter] {
			s.RecordMax(u.Counter, s.Counters[u.Counter]+u.Threshold)
		} else {
			s.Bump(u.Counter, u.Threshold)
		}
	}
}

// grow plays a pet down the lineage of target: at every fork on the way it
// leans into the child it wants for a stage, then crosses the level.
func grow(s *State, target string) {
	path := Lineage(target)
	for i := 1; i < len(path); i++ {
		lean(s, path[i])
		s.XP = xpFor(levelOfTier(Tier(path[i])))
		Advance(s, 0)
	}
}

func levelOfTier(tier int) int {
	if tier == 3 {
		return 3
	}
	return tier
}

func TestEveryFormIsReachableInTheNextLife(t *testing.T) {
	for form := range Sprites {
		if isSecret(form) {
			continue
		}
		s := veteran()
		grow(s, form)
		if got, _ := CurrentForm(s); got != form {
			t.Errorf("jugando hacia %s se llega a %s (camino %v)", form, got, Lineage(got))
		}
	}
}

// The measured case: methodical at 1036 for life, a stage spent testing.
func TestAForkIsDecidedByTheStageNotTheLifetime(t *testing.T) {
	s := New()
	s.Counters["methodical"] = 1036
	s.Counters["diffs"] = 971
	s.NewStage()
	s.Bump("inquisitive", 20)
	s.XP = xpFor(2)
	Advance(s, 0)
	if got, _ := CurrentForm(s); got != "probe" {
		t.Errorf("una etapa de tests sale %s, se esperaba probe", got)
	}
}

// Once decided, a fork does not move, whatever the counters do next. That is
// what v2's defended branch was approximating, and v3 gets it for nothing.
func TestADecidedForkNeverMoves(t *testing.T) {
	s := New()
	s.Bump("methodical", 30)
	s.XP = xpFor(2)
	Advance(s, 0)
	s.Bump("inquisitive", 5000)
	s.Bump("impulsive", 5000)
	if got, _ := CurrentForm(s); got != "pattern" {
		t.Errorf("la rama decidida se movio a %s", got)
	}
}

// A stage with no habit at all takes the route by default, the first child,
// and the pet is never stuck on a rung.
func TestAStageWithNoHabitTakesTheDefaultRoute(t *testing.T) {
	s := New()
	s.XP = xpFor(6)
	Advance(s, 0)
	got, _ := CurrentForm(s)
	if got != "scalpel" {
		t.Errorf("sin habitos sale %s, se esperaba la ruta por defecto hasta scalpel", got)
	}
	if want := []string{"spark", "pattern", "refactor", "surgeon", "scalpel"}; strconv.Quote(joined(Lineage(got))) != strconv.Quote(joined(want)) {
		t.Errorf("camino %v", Lineage(got))
	}
}

func joined(xs []string) string {
	out := ""
	for _, x := range xs {
		out += x + "/"
	}
	return out
}

// Crossing a level starts a stage: the habit that decided this fork does not
// carry over to the next one.
func TestAdvanceStartsANewStage(t *testing.T) {
	s := New()
	s.Bump("inquisitive", 40)
	s.Bump("tests", 40)
	s.XP = xpFor(2)
	Advance(s, 0)
	if got := s.Stage("tests"); got != 0 {
		t.Fatalf("tras cruzar, tests de etapa = %d, se esperaba 0", got)
	}
	// The next fork is the stage's: plans this time, so architect.
	s.Bump("plans", 3*scaleOf("plans"))
	s.XP = xpFor(3)
	Advance(s, 0)
	if got, _ := CurrentForm(s); got != "architect" {
		t.Errorf("la etapa de planes sale %s, se esperaba architect", got)
	}
}

// Every form reached is booked in the album, once per life.
func TestAdvanceBooksTheAlbum(t *testing.T) {
	s := New()
	s.Generation = 1
	s.XP = xpFor(3)
	reached := Advance(s, 123)
	if len(reached) != 2 || len(s.Album) != 2 {
		t.Fatalf("alcanzadas %v, album %v", reached, s.Album)
	}
	if s.Album[0].Form != "pattern" || s.Album[0].Gen != 1 || s.Album[0].At != 123 {
		t.Errorf("primera entrada %+v", s.Album[0])
	}
	Advance(s, 456)
	s.Collect("pattern", 789)
	if len(s.Album) != 2 {
		t.Errorf("el album repitio una forma de la misma vida: %v", s.Album)
	}
}

func TestStageOfAStreakThatBrokeIsWhatItHolds(t *testing.T) {
	s := New()
	s.Counters["test_streak"] = 40
	s.NewStage()
	s.Counters["test_streak"] = 0 // a red
	s.Bump("test_streak", 3)
	if got := s.Stage("test_streak"); got != 3 {
		t.Errorf("stage = %d, se esperaba 3", got)
	}
}

func TestStageOfAMaxCounterIsThisStagesMax(t *testing.T) {
	s := New()
	s.RecordMax("widest_commit", 241)
	s.NewStage()
	s.RecordMax("widest_commit", 12)
	if got := s.Stage("widest_commit"); got != 12 {
		t.Errorf("stage = %d, se esperaba 12", got)
	}
	if s.Counters["widest_commit"] != 241 {
		t.Errorf("el maximo de vida cambio a %d", s.Counters["widest_commit"])
	}
}

// A run that carried on from before the stage counts only its days in it.
func TestStageOfARunIsItsDaysThisStage(t *testing.T) {
	s := New()
	s.Counters["same_repo_days"] = 40
	s.NewStage()
	s.Counters["same_repo_days"] = 43
	if got := s.Stage("same_repo_days"); got != 3 {
		t.Errorf("stage = %d, se esperaba 3", got)
	}
	s.Counters["same_repo_days"] = 2 // broke and restarted
	if got := s.Stage("same_repo_days"); got != 2 {
		t.Errorf("stage = %d, se esperaba 2", got)
	}
}

// Crossing two levels at once decides both forks on the stage that ended.
func TestAJumpDecidesEveryForkOnTheSameStage(t *testing.T) {
	s := New()
	s.Bump("inquisitive", 20)
	s.Bump("tests", 20)
	s.Bump("test_streak", 20)
	s.XP = xpFor(6)
	Advance(s, 0)
	if got, _ := CurrentForm(s); got != "wasp" {
		t.Errorf("un salto hasta el nivel 6 sale %s, se esperaba wasp", got)
	}
}

// The shape and the level are two different facts. The path only grows, so a
// level lost to hunger or a blown context leaves the shape where it was.
func TestTheShapeHoldsWhileTheLevelIsStillAllowedToFall(t *testing.T) {
	s := New()
	s.Bump("inquisitive", 20)
	s.Bump("tests", 20)
	s.Bump("test_streak", 50)
	s.XP = xpFor(6)
	Advance(s, 0)
	form, level := CurrentForm(s)
	if form != "wasp" || level != 6 {
		t.Fatalf("el punto de partida es %s nivel %d", form, level)
	}
	s.XP -= 15
	s.Counters["test_streak"] = 0
	form, level = CurrentForm(s)
	if form != "wasp" {
		t.Errorf("la forma cayo a %s", form)
	}
	if level != 5 {
		t.Errorf("el nivel es %d; la XP cayo y el nivel la sigue", level)
	}
}

// The panel builds a what-if pet; asking must not write the path.
func TestCurrentFormDecidesPendingForksWithoutWriting(t *testing.T) {
	s := New()
	s.Bump("inquisitive", 20)
	s.XP = xpFor(3)
	if got, _ := CurrentForm(s); Tier(got) != 3 {
		t.Fatalf("a nivel 3 sale %s", got)
	}
	if len(s.Branch) != 0 || len(s.Album) != 0 {
		t.Errorf("preguntar escribio: branch %v, album %v", s.Branch, s.Album)
	}
}

// Every fork counter has a scale, or that fork's race is raw again.
func TestEveryForkCounterHasAScale(t *testing.T) {
	for form, counter := range BranchBy {
		if BranchScale[counter] <= 0 {
			t.Errorf("%s se bifurca por %q y ese contador no tiene escala", form, counter)
		}
	}
}

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
	for _, junk := range []string{"", "cazabugs", "../../etc/passwd", "WASP"} {
		if got := Tier(junk); got != 0 {
			t.Errorf("Tier(%q) = %d, se esperaba 0", junk, got)
		}
	}
}

// The path goes through a gate on the way in: each link must be a real edge of
// the tree. In v3 the mark and title links are part of the path too.
func TestABranchThatNamesNoRealChoiceIsDropped(t *testing.T) {
	for _, tc := range []struct {
		stored, wantKey, wantValue string
	}{
		{`{"spark":"pattern"}`, "spark", "pattern"},
		{`{"pattern":"tidy"}`, "pattern", "tidy"},
		{`{"bughunter":"bloodhound"}`, "bughunter", "bloodhound"}, // a mark: part of the path now
		{`{"bloodhound":"wolf"}`, "bloodhound", "wolf"},           // and its title
		{`{"spark":"godzilla"}`, "", ""},
		{`{"spark":"tidy"}`, "", ""},
		{`{"godzilla":"pattern"}`, "", ""},
		{`{"../../etc/passwd":"pattern"}`, "", ""},
		{`{"chispa":"pauta"}`, "spark", "pattern"},
	} {
		path := t.TempDir() + "/pet.json"
		if err := os.WriteFile(path,
			[]byte(`{"generation":1,"xp":`+strconv.Itoa(xpFor(3))+`,"branch":`+tc.stored+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got := Load(path).Branch
		if tc.wantKey == "" {
			if len(got) != 0 {
				t.Errorf("%s se guardo como %v", tc.stored, got)
			}
			continue
		}
		if got[tc.wantKey] != tc.wantValue {
			t.Errorf("%s se cargo como %v, se esperaba %s=%s", tc.stored, got, tc.wantKey, tc.wantValue)
		}
	}
}

func TestTheRungIsTranslatedAndSanitisedOnTheWayIn(t *testing.T) {
	for _, tc := range []struct{ stored, want string }{
		{"exterminador", "exterminator"},
		{"exterminator", "exterminator"},
		{"", ""},
		{"cazabugs", "bughunter"},
		{"../../etc/passwd", ""},
		{"WASP", ""},
	} {
		path := t.TempDir() + "/pet.json"
		if err := os.WriteFile(path,
			[]byte(`{"generation":1,"xp":`+strconv.Itoa(xpFor(5))+`,"form_seen":`+strconv.Quote(tc.stored)+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := Load(path).FormSeen; got != tc.want {
			t.Errorf("form_seen %q se cargo como %q, se esperaba %q", tc.stored, got, tc.want)
		}
	}
}

func TestTheV3FieldsSurviveTheRoundTrip(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := New()
	s.Generation = 3
	s.TitleDays = 2
	s.Counters["tests"] = 9
	s.NewStage()
	s.RecordMax("widest_commit", 7)
	s.Collect("pattern", 42)
	Save(s, path)
	back := Load(path)
	if back.Generation != 3 || back.TitleDays != 2 {
		t.Errorf("generation %d, title_days %d", back.Generation, back.TitleDays)
	}
	if back.StageBase["tests"] != 9 || back.StageMax["widest_commit"] != 7 {
		t.Errorf("stage_base %v, stage_max %v", back.StageBase, back.StageMax)
	}
	if len(back.Album) != 1 || back.Album[0] != (AlbumEntry{"pattern", 3, 42}) {
		t.Errorf("album %v", back.Album)
	}
}

func TestAnAlbumEntryThatIsNotAFormIsDropped(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	if err := os.WriteFile(path, []byte(`{"generation":1,"album":[`+
		`{"form":"wasp","gen":1,"at":1},{"form":"godzilla","gen":1,"at":2},{"gen":1}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Load(path).Album; len(got) != 1 || got[0].Form != "wasp" {
		t.Errorf("album %v", got)
	}
}

func TestACloneSharesNothingWithItsOriginal(t *testing.T) {
	s := New()
	s.XP = 400
	s.Counters["tests"] = 7
	s.Meals["commit"] = 111
	s.DayMarks["mon"] = "x"
	s.Log = append(s.Log, LogEntry{Event: "tests", XP: 15})
	s.Said = append(s.Said, "hola")
	s.Branch = map[string]string{Root: "pattern"}
	s.StageBase = map[string]int{"tests": 1}
	s.StageMax = map[string]int{"widest_commit": 1}
	s.Album = []AlbumEntry{{"pattern", 1, 1}}

	c := s.Clone()
	c.Counters["tests"] = 999
	c.Meals["commit"] = 222
	c.DayMarks["mon"] = "y"
	c.Log[0].XP = 999
	c.Said[0] = "adios"
	c.Branch[Root] = "ember"
	c.StageBase["tests"] = 9
	c.StageMax["widest_commit"] = 9
	c.Album[0].Form = "ember"

	if s.Counters["tests"] != 7 || s.Meals["commit"] != 111 || s.DayMarks["mon"] != "x" ||
		s.Log[0].XP != 15 || s.Said[0] != "hola" || s.Branch[Root] != "pattern" ||
		s.StageBase["tests"] != 1 || s.StageMax["widest_commit"] != 1 || s.Album[0].Form != "pattern" {
		t.Errorf("el clon escribio en el original: %+v", s)
	}
}

func TestCloningAnEmptyStateIsSafe(t *testing.T) {
	for _, s := range []*State{{}, New(), Load(t.TempDir() + "/missing.json")} {
		got := s.Clone()
		got.Counters["x"] = 1
		got.Meals["y"] = 1
		got.DayMarks["z"] = "1"
		got.StageBase["a"] = 1
		got.StageMax["b"] = 1
		got.Log = append(got.Log, LogEntry{})
		got.Said = append(got.Said, "")
		got.Album = append(got.Album, AlbumEntry{})
	}
}

func isSecret(form string) bool {
	for _, s := range Secrets {
		if form == s {
			return true
		}
	}
	return false
}
