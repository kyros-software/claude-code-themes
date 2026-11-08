package pet

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"
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
	if len(back.Album) == 0 || back.Album[0] != (AlbumEntry{"pattern", 3, 42}) {
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

// The title is worn for TitleStay days of use, counting the day it arrives,
// and the first meal after that is the new egg's.
func TestATitleIsWornThenTheEggHatches(t *testing.T) {
	s := New()
	s.XP = xpFor(6)
	Advance(s, t0.Unix())
	if !AtTitle(s) || s.TitleDays != 1 {
		t.Fatalf("at title %v, title_days %d", AtTitle(s), s.TitleDays)
	}
	s.LastDay = Today(t0)
	for d := 1; d < TitleStay; d++ {
		Feed(s, "commit", "", t0.Add(day*time.Duration(d)))
		if s.Generation > 1 {
			t.Fatalf("hatched on day %d of %d", d+1, TitleStay)
		}
	}
	Feed(s, "commit", "", t0.Add(day*time.Duration(TitleStay)))
	if s.Generation != 2 {
		t.Fatalf("generation %d after %d days at the title", s.Generation, TitleStay)
	}
	if form, level := CurrentForm(s); form != Root || level != 1 {
		t.Errorf("the egg is %s nivel %d", form, level)
	}
	if s.XP != Foods["commit"].XP {
		t.Errorf("xp %d: the meal that hatched it should be the egg's first", s.XP)
	}
}

// What carries over is the record; what resets is the shape.
func TestHatchKeepsTheRecordAndResetsTheShape(t *testing.T) {
	s := New()
	s.Generation = 1
	s.Counters["methodical"] = 500
	s.Streak, s.BestStreak, s.Hunger = 9, 12, 3
	s.Secret = "chimera"
	s.XP = xpFor(6)
	Advance(s, 1)
	album := len(s.Album)
	Hatch(s, 2)
	if s.Generation != 2 || s.XP != 0 || len(s.Branch) != 0 || s.Secret != "" || s.TitleDays != 0 {
		t.Errorf("shape not reset: gen %d xp %d branch %v secret %q", s.Generation, s.XP, s.Branch, s.Secret)
	}
	if s.Counters["methodical"] != 500 || s.Streak != 9 || s.BestStreak != 12 || s.Hunger != 3 {
		t.Errorf("record lost: %v streak %d/%d hunger %d", s.Counters, s.Streak, s.BestStreak, s.Hunger)
	}
	if len(s.Album) != album+1 || s.Album[len(s.Album)-1] != (AlbumEntry{Root, 2, 2}) {
		t.Errorf("album %v", s.Album)
	}
	if s.Stage("methodical") != 0 {
		t.Errorf("the new life starts with stage methodical %d", s.Stage("methodical"))
	}
}

func TestRescaleV2KeepsTheLevelAndThePlaceInIt(t *testing.T) {
	for _, c := range []struct{ old, want int }{
		{0, 0},
		{30, 100},          // halfway through v2 level 1
		{60, 200},          // v2 level 2 opens
		{400, 700},         // v2 level 4 opens
		{1200, 850},        // halfway through v2 level 4
		{2000, 1000},       // v2 level 5
		{4500, 1500},       // v2 level 6
		{4560, 1560},       // the v2 ceiling onto the v3 one
		{99999, XPCeiling}, // a hand-edited file stays under the ceiling
	} {
		if got := rescaleV2(c.old); got != c.want {
			t.Errorf("rescaleV2(%d) = %d, want %d", c.old, got, c.want)
		}
	}
}

// The measured pet: v2 at the ceiling wearing scalpel. It becomes generation 1
// at the v3 ceiling, still scalpel, with its lineage in the album and the
// title's days starting.
func TestAV2PetBecomesTheFirstGeneration(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	if err := os.WriteFile(path, []byte(`{"xp":4560,"form_seen":"scalpel",`+
		`"counters":{"methodical":1036,"diffs":971,"ctx_low":649},`+
		`"branch":{"spark":"pattern","pattern":"tidy"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Load(path)
	if s.Generation != 1 || s.XP != 1560 {
		t.Fatalf("generation %d, xp %d", s.Generation, s.XP)
	}
	if form, level := CurrentForm(s); form != "scalpel" || level != 6 {
		t.Errorf("migrated as %s nivel %d, want scalpel nivel 6", form, level)
	}
	if len(s.Album) != 5 || s.Album[4].Form != "scalpel" || s.Album[4].Gen != 1 {
		t.Errorf("album %v", s.Album)
	}
	if s.TitleDays != 1 || s.GenSeen != 1 {
		t.Errorf("title_days %d, gen_seen %d", s.TitleDays, s.GenSeen)
	}
	// The next life starts from the stage, not from methodical 1036.
	if s.Stage("methodical") != 0 {
		t.Errorf("stage methodical %d after migrating", s.Stage("methodical"))
	}
	// And it survives a save: Load does not migrate a v3 file twice.
	Save(s, path)
	if back := Load(path); back.XP != 1560 || len(back.Album) != 5 {
		t.Errorf("after a round trip xp %d album %v", back.XP, back.Album)
	}
}

// A v2 pet whose level is past its recorded shape decides the missing fork
// once, on the lifetime counters - what v2 would have shown - and the stage
// starts after it.
func TestAV2PetBehindItsLevelDecidesOnceOnTheLifetime(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	if err := os.WriteFile(path, []byte(`{"xp":500,"form_seen":"probe",`+
		`"counters":{"inquisitive":90,"tests":90,"plans":2}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Load(path)
	Save(s, path)
	s = Load(path)
	if form, _ := CurrentForm(s); form != "bughunter" {
		t.Errorf("migrated as %s, want bughunter on the lifetime tests", form)
	}
	if s.Stage("tests") != 0 {
		t.Errorf("the stage did not start after the fork: tests %d", s.Stage("tests"))
	}
}

// What a v2 binary does to a v3 file: keeps day_marks whole, drops the fields
// it does not know. Measured on 8-nov-2026 with 2.2.0 after the update.
func savedByV2(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"generation", "gen_seen", "album", "stage_base",
		"stage_max", "title_days", "repo_runs"} {
		delete(doc, k)
	}
	out, _ := json.Marshal(doc)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

// The pet of 8-nov-2026: migrated by 3.0.0, then saved by a 2.2.0 session that
// was still open. Its XP is already v3's and must not be rescaled again.
func TestAV3PetSavedByAV2BinaryIsNotMigratedTwice(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	if err := os.WriteFile(path, []byte(`{"xp":4560,"form_seen":"scalpel",`+
		`"counters":{"methodical":1036,"diffs":971,"ctx_low":649}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Load(path)
	Save(s, path)
	savedByV2(t, path)

	back := Load(path)
	if back.XP != 1560 || back.Generation != 1 || back.GenSeen != 1 {
		t.Fatalf("xp %d generation %d gen_seen %d, want 1560 1 1",
			back.XP, back.Generation, back.GenSeen)
	}
	if form, level := CurrentForm(back); form != "scalpel" || level != 6 {
		t.Errorf("came back as %s nivel %d, want scalpel nivel 6", form, level)
	}
	if len(back.Album) != 5 || back.TitleDays != 1 {
		t.Errorf("album %v title_days %d", back.Album, back.TitleDays)
	}
}

// The mark carries the generation, so a later life is not taken for the first.
func TestTheGenerationMarkKeepsTheLife(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := Load(path)
	s.Generation, s.GenSeen, s.XP = 3, 3, 250
	Save(s, path)
	savedByV2(t, path)

	back := Load(path)
	if back.Generation != 3 || back.XP != 250 || back.GenSeen != 3 {
		t.Errorf("generation %d xp %d gen_seen %d, want 3 250 3",
			back.Generation, back.XP, back.GenSeen)
	}
}

// Negative control: without the mark a file is a v2 pet, and is rescaled.
func TestWithoutTheMarkAV2PetIsStillRescaled(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	if err := os.WriteFile(path, []byte(`{"xp":1560,"day_marks":{"docs_days":"2026-11-07"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if s := Load(path); s.XP == 1560 || s.Generation != 1 {
		t.Errorf("xp %d generation %d: a v2 pet was not rescaled", s.XP, s.Generation)
	}
}
