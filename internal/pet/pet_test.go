package pet

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const day = 24 * time.Hour

// t0 is a fixed Wednesday, so the streaks are testable.
var t0 = time.Unix(1788000000, 0)

// --- sprites and rendering -------------------------------------------------

func TestEverySpriteRowIsExactlyTheCardWidth(t *testing.T) {
	for name, s := range Sprites {
		for i, row := range []string{s.Crest, s.Upper, s.Face, s.Lower, s.Feet} {
			if n := len([]rune(row)); n != SpriteWidth {
				t.Errorf("%s row %d is %d runes, want %d", name, i, n, SpriteWidth)
			}
		}
	}
}

func TestTheEyeColumnsLandOnBlanksInTheFace(t *testing.T) {
	for name, s := range Sprites {
		face := []rune(s.Face)
		for _, col := range s.EyeCols {
			if col < 0 || col >= SpriteWidth {
				t.Fatalf("%s eye column %d out of range", name, col)
			}
			if face[col] != ' ' {
				t.Errorf("%s eye column %d sits on %q, not a hole", name, col, face[col])
			}
		}
	}
}

func TestEveryStateDrawsFiveRowsOfTheCardWidth(t *testing.T) {
	for name := range Sprites {
		for _, v := range Vitals {
			for step := 0; step < 14; step++ {
				rows := Draw(name, v, step, false)
				for i, row := range rows {
					if n := len([]rune(strip(row))); n != SpriteWidth {
						t.Fatalf("%s/%s step %d row %d: %d runes", name, v.Label, step, i, n)
					}
				}
			}
		}
	}
}

// The compact row is artboard 09's option 9b: the torso, the face, and a third
// row that is the lower contour squeezed to three cells with the form's own
// feet on either side.
//
// It used to draw a FIXED "▘▝" pair there, the same two glyphs for all 27
// forms. The atlas makes the foot count part of the form's identity, so the
// compact row carries the form's own - which is what lets a six-legged telar
// read differently from a two-legged monje in three rows.
func TestTheCompactRowCarriesTheFormsOwnFeet(t *testing.T) {
	t.Setenv("STATUSLINE_PET_WALK", "1")
	for _, form := range []string{"spark", "bughunter", "loom", "leviathan"} {
		sprite := Sprites[form]
		for _, c := range []struct {
			what string
			v    Vital
			step int
			want string
		}{
			{"paso A", Vitals[0], 0, sprite.Feet},
			{"paso B", Vitals[0], 1, sprite.Step},
			{"quieta", Vitals[3], 0, sprite.Still},
			{"k.o.", KO, 0, sprite.KO},
		} {
			row := []rune(strip(DrawCompact(form, c.v, c.step, false)[2]))
			feet := []rune(c.want)
			// the two cells either side of the three-cell body
			for _, at := range [][2]int{{1, 1}, {2, 2}, {6, 6}, {7, 7}} {
				if row[at[0]] != feet[at[1]] {
					t.Errorf("%s %s: row %q does not carry %q at %d",
						form, c.what, string(row), string(feet), at[0])
				}
			}
		}
	}
}

func TestTheCompactBodyIsTheLowerContourSqueezed(t *testing.T) {
	for form, sprite := range Sprites {
		row := []rune(strip(DrawCompact(form, Vitals[0], 0, false)[2]))
		if got, want := string(row[3:6]), squeeze(sprite.Lower); got != want {
			t.Errorf("%s: body %q, want %q", form, got, want)
		}
	}
}

func TestCompactIsAlwaysThreeRowsOfTheCardWidth(t *testing.T) {
	for name := range Sprites {
		for _, v := range Vitals {
			for step := 0; step < 14; step++ {
				rows := DrawCompact(name, v, step, false)
				for i, row := range rows {
					if n := len([]rune(strip(row))); n != SpriteWidth {
						t.Fatalf("%s/%s step %d row %d: %d runes |%s|",
							name, v.Label, step, i, n, strip(row))
					}
				}
			}
		}
	}
}

func TestAnUnknownFormFallsBackToTheRoot(t *testing.T) {
	if Draw("godzilla", Vitals[0], 0, false) != Draw(Root, Vitals[0], 0, false) {
		t.Error("an unknown form did not fall back to the root")
	}
}

func TestTheEyesBelongToTheStateForEveryForm(t *testing.T) {
	// They used to be half the form's identity - OwnEyes, kept while "intact"
	// and surrendered from "easy" down. The atlas draws all 41 forms with the
	// same pair at each state, and hands identity to the crest, the feet and
	// the ramp instead.
	want := map[string]string{"fresh": "> <", "easy": "o o", "tired": "_ _", "k.o.": "x x"}
	for _, v := range Vitals {
		glyphs, ok := want[v.Label]
		if !ok {
			continue
		}
		for _, form := range []string{"spark", "probe", "bughunter", "leviathan", "chimera"} {
			face := strip(Draw(form, v, 0, false)[2])
			if !contains(face, glyphs) {
				t.Errorf("%s at %q = %q, want the state's %q", form, v.Label, face, glyphs)
			}
		}
	}
}

func TestHungerOnlyChangesTheColourNotTheGlyph(t *testing.T) {
	fed := Draw("spark", Vitals[0], 0, false)
	hungry := Draw("spark", Vitals[0], 0, true)
	if strip(fed[2]) != strip(hungry[2]) {
		t.Error("hunger changed the glyphs")
	}
	if fed[2] == hungry[2] {
		t.Error("hunger did not change the colour")
	}
}

func TestTheKOLiesDownWithoutLosingAFoot(t *testing.T) {
	// It used to sweep the feet row away entirely. The atlas lays them down
	// instead - "lo tumba en k.o. sin perder la cuenta de patas" - because the
	// foot count is half of what names the form.
	rows := Draw("marathon", KO, 0, false)
	feet := strip(rows[4])
	if trimSpace(feet) == "" {
		t.Error("the k.o. swept the feet away")
	}
	if feet != Sprites["marathon"].KO {
		t.Errorf("k.o. feet = %q, want %q", feet, Sprites["marathon"].KO)
	}
}

// --- vitality --------------------------------------------------------------

func TestStateForWalksTheThresholds(t *testing.T) {
	want := map[float64]string{
		0: "fresh", 22: "fresh", 23: "lively", 45: "lively", 63: "easy",
		78: "sluggish", 89: "tired", 99.9: "drowning", 100: "k.o.",
	}
	for usage, label := range want {
		if got := StateFor(usage).Label; got != label {
			t.Errorf("StateFor(%v) = %s, want %s", usage, got, label)
		}
	}
}

func TestTheThresholdsAreTheFirstVersionsComfortCurve(t *testing.T) {
	// The caps are not arbitrary and never were: they are
	// vida = 100*(1-(neck/100)^2) - the quadratic the very first statusline.sh
	// drew its face from - solved for the neck at the marks that version used,
	// 95/80/60/40/20. If somebody nudges a cap, this says what it is nudging.
	for _, c := range []struct {
		vida float64
		cap  float64
	}{{95, 22}, {80, 45}, {60, 63}, {40, 78}, {20, 89}} {
		neck := 100 * math.Sqrt(1-c.vida/100)
		if math.Abs(neck-c.cap) > 0.6 {
			t.Errorf("vida %v puts the neck at %.2f, but the cap is %v",
				c.vida, neck, c.cap)
		}
	}
}

func TestTheStateIsTheSessionAndNotTheAccount(t *testing.T) {
	// Three answers to one question, and the third is the session's own.
	//
	// A 50/30/20 mean DILUTED: the window full at 100 with the quotas at 20
	// and 10 came back as 58, which is "a gusto" - the pet reporting comfort
	// with no room left to work in. The tightest of the three necks fixed that
	// and bought a different lie: the quotas belong to the ACCOUNT, so two
	// sessions open at once, one at 6% of context and one at 64%, both sat
	// there saying "cansada" off a 5h quota at 81.
	f := func(v float64) *float64 { return &v }
	nan := math.NaN()
	for _, c := range []struct {
		name string
		ctx  *float64
		want float64
	}{
		{"the window is the whole of it", f(50), 50},
		{"nothing known is nothing claimed", nil, 0},
		{"a full window", f(100), 100},
		{"NaN is not a reading", &nan, 0},
		{"out of range is clamped", f(140), 100},
		{"and so is a negative", f(-5), 0},
	} {
		if got := ContextLoad(c.ctx); got != c.want {
			t.Errorf("%s: ContextLoad = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAQuotaCannotMoveTheState(t *testing.T) {
	// The quotas are not arguments any more, which is the point stated as a
	// type: an empty window is a fresh creature no matter what the account has
	// spent today, and band 1 is where you go to find out what it has spent.
	if got := StateFor(ContextLoad(ptr(6))).Label; got != "fresh" {
		t.Errorf("a window at 6%% gave %q, want fresh", got)
	}
	if got := StateFor(ContextLoad(ptr(81))).Label; got != "tired" {
		t.Errorf("a window at 81%% gave %q, want tired", got)
	}
}

func TestTheKOArrivesWithoutASpecialDoor(t *testing.T) {
	// StateFor used to take the context as a second argument purely so a full
	// window could force the k.o., because a weighted mean of three numbers
	// cannot reach 100 unless all three do. The context reaches it on its own.
	if got := StateFor(ContextLoad(ptr(100))).Label; got != "k.o." {
		t.Errorf("a full context gave %q, want k.o.", got)
	}
	if got := StateFor(ContextLoad(ptr(95))).Label; got != "drowning" {
		t.Errorf("95 gave %q, want drowning", got)
	}
}

func ptr(v float64) *float64 { return &v }

// --- the tree --------------------------------------------------------------

func TestEveryFormInTheTreeHasASprite(t *testing.T) {
	named := map[string]bool{Root: true}
	for parent, kids := range Tree {
		named[parent] = true
		for _, kid := range kids {
			named[kid] = true
		}
	}
	for _, s := range Secrets {
		named[s] = true
	}
	if len(named) != len(Sprites) {
		t.Fatalf("tree names %d forms, sprites has %d", len(named), len(Sprites))
	}
	for name := range named {
		if _, ok := Sprites[name]; !ok {
			t.Errorf("%s is in the tree with no sprite", name)
		}
	}
}

func TestEveryLevelFiveMarkHasAnUnlock(t *testing.T) {
	trades := []string{"refactor", "tidy", "bughunter", "architect", "sprinter", "marathon", "feral"}
	count := 0
	for _, trade := range trades {
		for _, mark := range Tree[trade] {
			count++
			if _, ok := Unlocks[mark]; !ok {
				t.Errorf("%s has no unlock", mark)
			}
		}
	}
	if count != len(Unlocks) {
		t.Errorf("%d marks against %d unlocks", count, len(Unlocks))
	}
}

func TestEveryForkHasADecidingCounter(t *testing.T) {
	for parent, kids := range Tree {
		for _, kid := range kids {
			_, branch := BranchBy[kid]
			_, unlock := Unlocks[kid]
			// A title is the one child of its mark, and what decides it is more
			// of the mark's own habit - not a fork between siblings, so it has
			// no BranchBy of its own.
			if _, title := TitleUnlock(parent); title && Titles[parent] == kid {
				continue
			}
			if !branch && !unlock {
				t.Errorf("%s is decided by nothing", kid)
			}
		}
	}
}

func TestTheLevelNeverGoesDown(t *testing.T) {
	seen := 0
	for xp := 0; xp < 1200; xp += 7 {
		level := LevelFor(xp)
		if level < seen {
			t.Fatalf("level dropped from %d to %d at xp %d", seen, level, xp)
		}
		seen = level
	}
}

func TestTheBranchFollowsTheHighestCounter(t *testing.T) {
	methodical := &State{XP: 200, Counters: map[string]int{"methodical": 9, "inquisitive": 1}}
	inquisitive := &State{XP: 200, Counters: map[string]int{"methodical": 1, "inquisitive": 9}}
	if form, _ := CurrentForm(methodical); form != "refactor" && form != "tidy" {
		t.Errorf("methodical went to %s", form)
	}
	if form, _ := CurrentForm(inquisitive); form != "bughunter" && form != "architect" {
		t.Errorf("inquisitive went to %s", form)
	}
}

func TestATieFallsBackToDesignOrder(t *testing.T) {
	// A tie is one day of each habit, not one COUNT of each: the three
	// counters are scaled before they race, so 5/5/5 is three different
	// amounts of work. See BranchScale.
	tied := &State{XP: 100, Counters: map[string]int{
		"methodical": BranchScale["methodical"], "inquisitive": BranchScale["inquisitive"],
		"impulsive": BranchScale["impulsive"]}}
	if form, _ := CurrentForm(tied); form != Tree[Root][0] {
		t.Errorf("tie went to %s, want %s", form, Tree[Root][0])
	}
}

func TestASecretWinsOverTheTreeAndAnUnknownOneIsIgnored(t *testing.T) {
	won := &State{XP: xpFor(5), Secret: "phoenix",
		Counters: map[string]int{"impulsive": 9, "ctx_maxed": 9}}
	if form, level := CurrentForm(won); form != "phoenix" || level != 5 {
		t.Errorf("secret gave %s/%d, want phoenix/5", form, level)
	}
	if form, _ := CurrentForm(&State{XP: xpFor(5), Secret: "godzilla"}); form == "godzilla" {
		t.Error("an unknown secret was handed over")
	}
}

func TestASecretStillWaitsForLevelFive(t *testing.T) {
	// The condition is met at level 4 - "dos temperamentos empatados al subir
	// a nivel 4" - but the form is a level-5 one. The canvas draws exactly
	// this pet: "refactor · nivel 4" with "488 para quimera" underneath.
	// Four days of each of the two temperaments - which is the tie, and is not
	// the same number twice since BranchScale.
	pending := &State{XP: 412, Secret: "chimera",
		Counters: map[string]int{
			"methodical": 4 * BranchScale["methodical"], "inquisitive": 4 * BranchScale["inquisitive"],
			"diffs": 3 * BranchScale["diffs"]}}
	form, level := CurrentForm(pending)
	if form == "chimera" {
		t.Error("the secret was handed over before level 5")
	}
	if form != "refactor" || level != 4 {
		t.Errorf("pending secret gave %s/%d, want refactor/4", form, level)
	}

	// One more level and it is hers.
	pending.XP = xpFor(5)
	if form, level := CurrentForm(pending); form != "chimera" || level != 5 {
		t.Errorf("at level-5 xp the secret gave %s/%d, want chimera/5", form, level)
	}
}

func TestAMarkNeedsItsHabit(t *testing.T) {
	base := &State{XP: xpFor(5), Counters: map[string]int{"methodical": 9, "diffs": 9}}
	if form, _ := CurrentForm(base); form != "refactor" {
		t.Fatalf("without the habit the form is %s", form)
	}
	base.Counters["diff_streak"] = 20
	if form, _ := CurrentForm(base); form != "surgeon" {
		t.Errorf("with the habit the form is %s", form)
	}
}

func TestLineageWalksBackToTheRoot(t *testing.T) {
	for name := range Sprites {
		if name == "phoenix" || name == "chimera" {
			continue
		}
		path := Lineage(name)
		if path[0] != Root || path[len(path)-1] != name {
			t.Errorf("Lineage(%s) = %v", name, path)
		}
	}
}

func TestNextThresholdRunsOutAtTheTop(t *testing.T) {
	if xp, ok := NextThreshold(0); !ok || xp != 60 {
		t.Errorf("NextThreshold(0) = %d,%v", xp, ok)
	}
	if _, ok := NextThreshold(xpFor(6)); ok {
		t.Error("there is something after level 5")
	}
}

// --- feeding ---------------------------------------------------------------

func TestEveryFoodHasALabel(t *testing.T) {
	for name, food := range Foods {
		if food.Label == "" {
			t.Errorf("%s has no label", name)
		}
	}
}

func TestAnUnknownFoodIsRefused(t *testing.T) {
	s := New()
	if Feed(s, "pizza", "", t0) || s.XP != 0 {
		t.Error("the pet ate a pizza")
	}
}

func TestAMealGivesXPAndTakesHunger(t *testing.T) {
	s := New()
	s.Hunger = 8
	if !Feed(s, "tests", "", t0) || s.XP != 15 || s.Hunger != 4 {
		t.Errorf("after tests: xp=%d hunger=%d", s.XP, s.Hunger)
	}
}

func TestXPNeverGoesNegative(t *testing.T) {
	s := New()
	Feed(s, "overflow", "", t0)
	if s.XP != 0 {
		t.Errorf("xp = %d", s.XP)
	}
}

func TestTheCooldownOnlyBindsTheHandFed(t *testing.T) {
	// "un tiro al plato · una vez cada cuatro horas": /feed waits, the meals
	// you earn do not.
	s := New()
	if !Feed(s, "feed", "", t0) {
		t.Fatal("the first feed was refused")
	}
	if Feed(s, "feed", "", t0.Add(FeedCooldown-time.Minute)) {
		t.Error("fed again inside the cooldown")
	}
	if !Feed(s, "feed", "", t0.Add(FeedCooldown)) {
		t.Error("still refused once the cooldown was up")
	}
	if s.XP != 6 {
		t.Errorf("xp = %d, want 6", s.XP)
	}
	// A green suite in the middle is not held back by it.
	if !Feed(s, "tests", "", t0.Add(FeedCooldown+time.Minute)) {
		t.Error("tests were refused by the feed cooldown")
	}
}

func TestTheCooldownSurvivesMidnight(t *testing.T) {
	// The log is cleared daily, so reading the last meal out of it would forget
	// a feed at 23:30 the moment the date rolled over.
	s := New()
	lateNight := time.Date(2026, 9, 1, 23, 30, 0, 0, time.Local)
	if !Feed(s, "feed", "", lateNight) {
		t.Fatal("the first feed was refused")
	}
	if Feed(s, "feed", "", lateNight.Add(90*time.Minute)) {
		t.Error("midnight reset the cooldown")
	}
}

func TestTheLogStaysCapped(t *testing.T) {
	s := New()
	for i := 0; i < 50; i++ {
		Feed(s, "tests", "", t0.Add(time.Duration(i)*time.Second))
	}
	if len(s.Log) > LogMax {
		t.Errorf("log has %d entries", len(s.Log))
	}
}

func TestTheStreakGrowsAndBreaks(t *testing.T) {
	s := New()
	for d := 0; d < 5; d++ {
		Feed(s, "commit", "", t0.Add(time.Duration(d)*day))
	}
	if s.Streak != 5 || s.BestStreak != 5 {
		t.Fatalf("streak=%d best=%d", s.Streak, s.BestStreak)
	}
	// A skipped day breaks the streak but not the record.
	Feed(s, "commit", "", t0.Add(8*day))
	if s.Streak != 1 || s.BestStreak != 5 {
		t.Errorf("after the gap streak=%d best=%d", s.Streak, s.BestStreak)
	}
}

func TestSeveralMealsTheSameDayAreOneDayOfStreak(t *testing.T) {
	s := New()
	for i := 0; i < 6; i++ {
		Feed(s, "commit", "", t0.Add(time.Duration(i)*time.Minute))
	}
	if s.Streak != 1 {
		t.Errorf("streak = %d", s.Streak)
	}
}

func TestHungerIsOnePerHourAndStopsAtTheCap(t *testing.T) {
	s := New()
	s.LastFed = t0.Unix()
	DecayHunger(s, t0.Add(3*time.Hour+time.Minute))
	if s.Hunger != 3 {
		t.Errorf("hunger = %d", s.Hunger)
	}
	DecayHunger(s, t0.Add(50*time.Hour))
	if s.Hunger != HungerMax {
		t.Errorf("hunger = %d, want the cap", s.Hunger)
	}
}

func TestHungerDoesNotMoveForAPetThatNeverAte(t *testing.T) {
	s := New()
	DecayHunger(s, t0.Add(99*time.Hour))
	if s.Hunger != 0 {
		t.Errorf("hunger = %d", s.Hunger)
	}
}

func TestAnOverflowBreaksTheCleanStreaks(t *testing.T) {
	s := New()
	// A green suite only counts once an hour now, so the streak has to be
	// built at the pace a person actually works at.
	for i := 0; i < 5; i++ {
		at := t0.Add(time.Duration(i) * 90 * time.Minute)
		Feed(s, "tests", "", at)
		Feed(s, "commit", "", at.Add(time.Second))
	}
	if s.Counters["test_streak"] != 5 {
		t.Fatalf("test_streak = %d", s.Counters["test_streak"])
	}
	Feed(s, "overflow", "", t0.Add(9*time.Hour))
	if s.Counters["test_streak"] != 0 || s.Counters["diff_streak"] != 0 {
		t.Error("the streaks survived the blowout")
	}
}

func TestThePhoenixNeedsTheFullHungerRoundTrip(t *testing.T) {
	s := New()
	s.XP = 950
	s.Counters = map[string]int{"impulsive": 9, "long_sessions": 9}
	if form, _ := CurrentForm(s); form != "marathon" {
		t.Fatalf("form = %s", form)
	}
	// Hunger has to RISE the way it really does: the peak is recorded by the
	// decay, not by the meal that brings it back down.
	s.LastFed = t0.Unix()
	DecayHunger(s, t0.Add(12*time.Hour))
	if s.Hunger != HungerMax {
		t.Fatalf("hunger = %d", s.Hunger)
	}
	for i := 0; i < 5; i++ {
		Feed(s, "tests", "", t0.Add(12*time.Hour+time.Duration(i)*90*time.Minute))
	}
	if s.Hunger != 0 || s.Secret != "phoenix" {
		t.Errorf("hunger=%d secret=%q", s.Hunger, s.Secret)
	}
}

func TestThePhoenixOnlyFromTheTwoFormsThatReachIt(t *testing.T) {
	s := New()
	s.XP = 950
	s.Counters = map[string]int{"methodical": 9, "ctx_low": 9}
	if form, _ := CurrentForm(s); form != "tidy" {
		t.Fatalf("form = %s", form)
	}
	s.LastFed = t0.Unix()
	DecayHunger(s, t0.Add(12*time.Hour))
	for i := 0; i < 5; i++ {
		Feed(s, "tests", "", t0.Add(12*time.Hour+time.Duration(i)*time.Minute))
	}
	if s.Secret != "" {
		t.Errorf("a tidy pet became %q", s.Secret)
	}
}

func TestTheChimeraNeedsATieAtLevelFour(t *testing.T) {
	// Four days of each, which is the tie the chimera means. It is not four
	// COUNTS of each: a commit and a green suite buy different amounts of
	// temperament, so the numbers differ. See BranchScale.
	s := New()
	s.XP = 400
	s.Counters = map[string]int{
		"methodical": 4 * BranchScale["methodical"], "inquisitive": 4 * BranchScale["inquisitive"]}
	Feed(s, "compact", "", t0)
	if s.Secret != "chimera" {
		t.Errorf("secret = %q", s.Secret)
	}
}

func TestASecretIsNeverOverwritten(t *testing.T) {
	s := New()
	s.Secret = "phoenix"
	s.XP = 400
	s.Counters = map[string]int{
		"methodical": 4 * BranchScale["methodical"], "inquisitive": 4 * BranchScale["inquisitive"]}
	Feed(s, "compact", "", t0)
	if s.Secret != "phoenix" {
		t.Errorf("secret = %q", s.Secret)
	}
}

// --- the life file ---------------------------------------------------------

var v1 = map[string]any{
	"xp": 450, "hambre": 3, "comio": 1788200000, "racha": 4, "mejor_racha": 9,
	"nivel_visto": 4, "hambre_tope": 5, "feed_hoy": 1,
	"ultimo_dia": "2026-09-01", "repo_dia": "r|2026-09-01", "hoy_dia": "2026-09-01",
	"secreta": "fenix",
	"contadores": map[string]any{"metodico": 40, "racha_diffs": 12,
		"sesiones_bajo_40": 3, "tests": 7, "algo_mio": 5},
	"hoy":        []any{map[string]any{"q": "tarea", "xp": 6, "t": 1788250000, "n": "x"}},
	"marcas_dia": map[string]any{"dias_docs": "2026-08-30"},
}

func write(t *testing.T, dir string, doc any) string {
	t.Helper()
	path := filepath.Join(dir, "pet.json")
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAV1FileIsTranslated(t *testing.T) {
	s := Load(write(t, t.TempDir(), v1))
	if s.XP != 450 || s.Hunger != 3 || s.LastFed != 1788200000 || s.Streak != 4 ||
		s.BestStreak != 9 || s.LevelSeen != 4 || s.RepoDay != "r|2026-09-01" {
		t.Fatalf("scalars: %+v", s)
	}
	if s.Secret != "phoenix" {
		t.Errorf("secret = %q", s.Secret)
	}
	want := map[string]int{"methodical": 40, "diff_streak": 12,
		"sessions_under_40": 3, "tests": 7, "algo_mio": 5}
	for k, v := range want {
		if s.Counters[k] != v {
			t.Errorf("counter %s = %d, want %d", k, s.Counters[k], v)
		}
	}
	if len(s.Log) != 1 || s.Log[0].Event != "task" {
		t.Errorf("log = %+v", s.Log)
	}
	if s.DayMarks["docs_days"] != "2026-08-30" {
		t.Errorf("day_marks = %v", s.DayMarks)
	}
}

func TestAV2FileIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	first := Load(write(t, dir, v1))
	path := filepath.Join(dir, "again.json")
	if !Save(first, path) {
		t.Fatal("save failed")
	}
	second := Load(path)
	if second.XP != first.XP || second.Secret != first.Secret ||
		len(second.Counters) != len(first.Counters) {
		t.Errorf("round trip changed the state")
	}
}

func TestAMissingFileIsANewborn(t *testing.T) {
	s := Load(filepath.Join(t.TempDir(), "nope.json"))
	if s.XP != 0 || s.Secret != "" || len(s.Counters) != 0 {
		t.Errorf("newborn = %+v", s)
	}
}

func TestJunkNeverPanicsAndNeverLies(t *testing.T) {
	junk := []string{
		"", "   ", "not json at all", "[]", "null", "42", `"a string"`,
		`{"xp": "muchos"}`, `{"xp": null}`, `{"xp": true}`, `{"xp": 1e999}`,
		`{"contadores": "no soy un dict"}`, `{"hoy": "tampoco"}`,
		`{"hoy": [1, 2, {"q": "tests"}]}`, `{"secreta": 7}`,
		`{"marcas_dia": {"a": 1}}`, `{"secret": "godzilla"}`,
	}
	dir := t.TempDir()
	for _, text := range junk {
		path := filepath.Join(dir, "pet.json")
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		s := Load(path)
		if s.Counters == nil || s.Log == nil || s.DayMarks == nil {
			t.Errorf("%q left a nil map", text)
		}
		if s.Secret != "" {
			if _, ok := Sprites[string(s.Secret)]; !ok {
				t.Errorf("%q produced secret %q", text, s.Secret)
			}
		}
	}
}

func TestTheLogIsCapped(t *testing.T) {
	entries := make([]any, 200)
	for i := range entries {
		entries[i] = map[string]any{"q": "tests", "xp": 1, "t": i, "n": ""}
	}
	s := Load(write(t, t.TempDir(), map[string]any{"log": entries}))
	if len(s.Log) != LogMax {
		t.Errorf("log has %d entries", len(s.Log))
	}
}

func TestTheWriteIsAtomicAndLeavesNoLitter(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	path := filepath.Join(dir, "pet.json")
	s := New()
	s.XP = 7
	if !Save(s, path) {
		t.Fatal("save failed")
	}
	if Load(path).XP != 7 {
		t.Error("what was saved is not what was read")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "pet.json" {
		t.Errorf("directory holds %v", entries)
	}
}

func TestTheEmptySecretIsWrittenAsNull(t *testing.T) {
	// The Python this replaces wrote null, and a user can move between the two
	// mid-session.
	path := filepath.Join(t.TempDir(), "pet.json")
	Save(New(), path)
	raw, _ := os.ReadFile(path)
	if !contains(string(raw), `"secret": null`) {
		t.Errorf("secret was not null:\n%s", raw)
	}
}

func TestMarkDayCountsDaysNotTimes(t *testing.T) {
	s := New()
	if !s.MarkDay("docs", t0) || s.MarkDay("docs", t0.Add(time.Hour)) {
		t.Fatal("the same day counted twice")
	}
	if s.Counters["docs_days"] != 1 {
		t.Fatalf("docs_days = %d", s.Counters["docs_days"])
	}
	s.MarkDay("docs", t0.Add(day))
	if s.Counters["docs_days"] != 2 {
		t.Fatalf("docs_days = %d", s.Counters["docs_days"])
	}
	// A skipped day restarts the run.
	s.MarkDay("docs", t0.Add(4*day))
	if s.Counters["docs_days"] != 1 {
		t.Errorf("docs_days = %d after the gap", s.Counters["docs_days"])
	}
}

func TestRecordMaxKeepsTheRecordNotTheSum(t *testing.T) {
	s := New()
	s.RecordMax("widest_commit", 5)
	s.RecordMax("widest_commit", 12)
	s.RecordMax("widest_commit", 3)
	if s.Counters["widest_commit"] != 12 {
		t.Errorf("widest_commit = %d", s.Counters["widest_commit"])
	}
}

// small helpers, so the test file pulls in nothing the package does not
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

func strip(s string) string {
	out := make([]rune, 0, len(s))
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\033':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case !inEscape:
			out = append(out, r)
		}
	}
	return string(out)
}

// --- what it says -----------------------------------------------------------

func TestSilenceByDefault(t *testing.T) {
	s := New()
	if line := Speak(s, EventNothing, "refactor", t0, first); line != "" {
		t.Errorf("spoke without an event: %q", line)
	}
}

// say is Speak followed by Remember: what happens to a bubble that reaches the
// screen. Speak on its own no longer books anything, because a bubble the band
// drops for width must not spend the cooldown.
func say(s *State, e Event, form string, now time.Time, pick func(int) int) string {
	line := Speak(s, e, form, now, pick)
	Remember(s, line, now)
	return line
}

func TestSpeakOnItsOwnBooksNothing(t *testing.T) {
	// The bug this split exists for: the bubble is the first thing band 4
	// drops when it runs short, and booking the line before it reached the
	// screen burned five minutes of silence for something nobody read.
	s := New()
	if Speak(s, EventBigMeal, "refactor", t0, first) == "" {
		t.Fatal("said nothing on a big meal")
	}
	if s.SaidAt != 0 || len(s.Said) != 0 {
		t.Errorf("Speak booked the line by itself: said=%v at=%d", s.Said, s.SaidAt)
	}
	// Unspoken means unbooked: the very next refresh may try again.
	if Speak(s, EventBigMeal, "refactor", t0.Add(time.Second), first) == "" {
		t.Error("a line that never reached the screen still started the cooldown")
	}
	// And once it does reach the screen, it books.
	line := Speak(s, EventBigMeal, "refactor", t0, first)
	Remember(s, line, t0)
	if s.SaidAt == 0 || len(s.Said) != 1 {
		t.Errorf("Remember booked nothing: said=%v at=%d", s.Said, s.SaidAt)
	}
}

func TestTheCooldownGagsIt(t *testing.T) {
	s := New()
	if say(s, EventBigMeal, "refactor", t0, first) == "" {
		t.Fatal("said nothing on a big meal")
	}
	if line := say(s, EventBigMeal, "refactor", t0.Add(SpeechCooldown-time.Second), first); line != "" {
		t.Errorf("spoke inside the cooldown: %q", line)
	}
	if say(s, EventBigMeal, "refactor", t0.Add(SpeechCooldown), first) == "" {
		t.Error("still silent once the cooldown was up")
	}
}

func TestItDoesNotRepeatItself(t *testing.T) {
	s := New()
	said := map[string]bool{}
	at := t0
	for i := 0; i < len(Repertoire["refactor"]); i++ {
		line := say(s, EventBigMeal, "refactor", at, first)
		if line == "" {
			t.Fatalf("silent on round %d", i)
		}
		if said[line] {
			t.Errorf("repeated %q before exhausting the repertoire", line)
		}
		said[line] = true
		at = at.Add(SpeechCooldown)
	}
	if len(s.Said) != SaidMemory {
		t.Errorf("remembers %d lines, want %d", len(s.Said), SaidMemory)
	}
	// Exhausted: it starts over rather than going mute.
	if say(s, EventBigMeal, "refactor", at, first) == "" {
		t.Error("went mute once the repertoire ran out")
	}
}

func TestAFormWithNoRepertoireOnlySaysTheSharedLines(t *testing.T) {
	// The fourteen marks and the two secrets have no voice of their own.
	if _, ok := Repertoire["surgeon"]; ok {
		t.Fatal("surgeon was given a repertoire; this test needs one without")
	}
	s := New()
	if line := Speak(s, EventBigMeal, "surgeon", t0, first); line != "" {
		t.Errorf("invented a line for a form with no repertoire: %q", line)
	}
	s = New()
	s.XP = 400
	if line := Speak(s, EventLevelUp, "surgeon", t0, first); line != "nivel 4. ya soy alguien." {
		t.Errorf("level-up line = %q", line)
	}
}

func TestTheSharedLinesCarryTheirNumber(t *testing.T) {
	s := New()
	s.Streak = 5
	if line := Speak(s, EventStreak, "surgeon", t0, first); line != "cinco días de racha. mañana no me falles." {
		t.Errorf("streak line = %q", line)
	}
	s = New()
	s.Streak = 11
	if line := Speak(s, EventStreak, "surgeon", t0, first); line != "11 días de racha. mañana no me falles." {
		t.Errorf("big streak line = %q", line)
	}
}

func TestEveryFormWithARepertoireHasThreeLines(t *testing.T) {
	for name, lines := range Repertoire {
		if len(lines) != 3 {
			t.Errorf("%s has %d lines, the canvas gives 3", name, len(lines))
		}
		if _, ok := Sprites[name]; !ok {
			t.Errorf("%s has a repertoire but no sprite", name)
		}
	}
}

func first(int) int { return 0 }

func TestTheXPBarMeasuresTheLevelNotTheTotal(t *testing.T) {
	// Against the running total the bar opens a third full on the morning
	// after a level-up. Against the stretch it opens empty and closes full,
	// which is the only reading that is not a lie.
	for _, c := range []struct {
		xp         int
		done, span int
	}{
		{0, 0, 60},     // newborn
		{30, 30, 60},   // halfway through level 1
		{59, 59, 60},   // one short
		{60, 0, 120},   // level 2 opens EMPTY, not at a third
		{120, 60, 120}, // halfway through level 2
		{179, 119, 120},
		{180, 0, 220},  // level 3 opens empty
		{400, 0, 1600}, // level 4 opens empty, and it is the long one
		{1999, 1599, 1600},
	} {
		done, span, ok := LevelProgress(c.xp)
		if !ok {
			t.Errorf("xp %d: no stretch left, want one", c.xp)
			continue
		}
		if done != c.done || span != c.span {
			t.Errorf("xp %d: %d/%d, want %d/%d", c.xp, done, span, c.done, c.span)
		}
	}
}

func TestTheXPStretchRunsOutAtTheTop(t *testing.T) {
	for _, xp := range []int{xpFor(6), xpFor(6) + 500, 99999} {
		if _, _, ok := LevelProgress(xp); ok {
			t.Errorf("xp %d still has a stretch above it", xp)
		}
	}
}

func TestAHandEditedNegativeXPReadsAsANewborn(t *testing.T) {
	// pet.json is user-writable. A negative XP must not hand back a zero span
	// - band 4 reads that as "no bar" and then skips the habit bar too.
	done, span, ok := LevelProgress(-50)
	if !ok {
		t.Fatal("negative xp ran out of ladder")
	}
	if wantDone, wantSpan, _ := LevelProgress(0); done != wantDone || span != wantSpan {
		t.Errorf("negative xp gave %d/%d, want a newborn's %d/%d",
			done, span, wantDone, wantSpan)
	}
}

func TestAtTheTopTheProgressBecomesTheHabit(t *testing.T) {
	// Level 5, trade but no mark: what is left to reach is the habit.
	s := New()
	s.XP = xpFor(5)
	s.Counters["inquisitive"] = 5
	s.Counters["tests"] = 9
	s.Counters["repro_before_fix"] = 4 // sabueso wants 10 -> 0.40
	s.Counters["test_streak"] = 12     // exterminator wants 15 -> 0.80
	form, _ := CurrentForm(s)
	if form != "bughunter" {
		t.Fatalf("form is %q, want bughunter", form)
	}
	mark, ok := NextMark(s, form)
	if !ok {
		t.Fatal("no mark within reach, want the closest one")
	}
	if mark.Form != "exterminator" {
		t.Errorf("closest mark is %q, want exterminator", mark.Form)
	}
	if mark.Done != 12 || mark.Threshold != 15 {
		t.Errorf("habit at %d/%d, want 12/15", mark.Done, mark.Threshold)
	}
}

func TestAMarkStillHasItsTitleToReach(t *testing.T) {
	// The mark used to be the end of the road. There is a title behind each
	// one now, asking for a good deal more of the same habit, so band 4 keeps
	// a bar to fill.
	s := New()
	s.XP = xpFor(5)
	s.Counters["inquisitive"] = 5
	s.Counters["tests"] = 9
	s.Counters["test_streak"] = 15 // earned it
	form, _ := CurrentForm(s)
	if form != "exterminator" {
		t.Fatalf("form is %q, want exterminator", form)
	}
	mark, ok := NextMark(s, form)
	if !ok {
		t.Fatal("a pet wearing its mark was offered nothing to reach")
	}
	if mark.Form != "wasp" || mark.Threshold != TitleAsks["wasp"] {
		t.Errorf("next = %s at %d/%d, want wasp at %d",
			mark.Form, mark.Done, mark.Threshold, TitleAsks["wasp"])
	}
	// And a pet wearing the TITLE has nothing beyond it.
	s.XP = xpFor(6)
	s.Counters["test_streak"] = TitleAsks["wasp"]
	title, _ := CurrentForm(s)
	if title != "wasp" {
		t.Fatalf("form is %q, want wasp", title)
	}
	if _, ok := NextMark(s, title); ok {
		t.Error("a pet already wearing its title was offered another")
	}
}

// onABranch is a pet standing on `bughunter`, secret in hand, with as much of
// the exterminator's habit as asked for.
func onABranch(secret Secret, level, streak int) *State {
	s := New()
	s.XP = xpFor(level)
	s.Secret = secret
	s.Counters = map[string]int{
		"inquisitive": 10 * BranchScale["inquisitive"], // probe over pattern
		"tests":       10 * BranchScale["tests"],       // bughunter over architect
		"test_streak": streak,                          // exterminator, and wasp behind it
	}
	return s
}

// A secret used to be the end of the road: walk returned it before the tree
// ran at all, so the pet stopped at rung 5 for good - no title, and the branch
// it was standing on stopped meaning anything the day the secret landed.
//
// The secret still wins its OWN rung. It does not win the one above.
func TestASecretStillGrowsIntoItsTitle(t *testing.T) {
	// The habit is not there yet: the secret is the best the pet has.
	held := onABranch("phoenix", 6, Unlocks["exterminator"].Threshold)
	if form, _ := CurrentForm(held); form != "phoenix" {
		t.Errorf("con la marca ganada y el título no, sale %q, se esperaba phoenix", form)
	}

	// And with the title's habit paid for, the title outranks it.
	earned := onABranch("phoenix", 6, TitleAsks["wasp"])
	if form, level := CurrentForm(earned); form != "wasp" || level != 6 {
		t.Errorf("con el título ganado sale %q/%d, se esperaba wasp/6", form, level)
	}
}

// The level the card prints beside a secret is the pet's, not a 5 nailed on.
// It used to be the literal, which nobody could see because every test that
// asked stood the pet at level 5.
func TestASecretDoesNotFreezeTheLevel(t *testing.T) {
	s := onABranch("chimera", 6, 0)
	if form, level := CurrentForm(s); form != "chimera" || level != 6 {
		t.Errorf("sale %q/%d, se esperaba chimera/6", form, level)
	}
}

// A title, once worn, is not handed back to the secret when the streak falls -
// the floor stops that, the same as for any other form.
func TestATitleIsNotTakenBackByTheSecret(t *testing.T) {
	s := onABranch("phoenix", 6, TitleAsks["wasp"])
	form, _ := CurrentForm(s)
	RememberForm(s, form)

	s.Counters["test_streak"] = 0 // the context blew and the streak went with it
	if got, _ := CurrentForm(s); got != "wasp" {
		t.Errorf("la racha se cayó y el título con ella: %q", got)
	}
}

// And the bar has something to point at again. While the secret was terminal
// this was correctly empty; now it is the title the pet is climbing towards,
// which the panel would otherwise never mention.
func TestASecretIsPointedAtItsTitle(t *testing.T) {
	s := onABranch("phoenix", 5, Unlocks["exterminator"].Threshold)
	form, _ := CurrentForm(s)
	mark, ok := NextMark(s, form)
	if !ok {
		t.Fatal("una secreta no tiene a dónde ir, y ahora sí lo tiene")
	}
	if mark.Form != "wasp" || mark.Threshold != TitleAsks["wasp"] {
		t.Errorf("apunta a %s %d/%d, se esperaba wasp .../%d",
			mark.Form, mark.Done, mark.Threshold, TitleAsks["wasp"])
	}
}

func TestAHabitPastItsThresholdDoesNotOverflowTheBar(t *testing.T) {
	// The habit can be met long before the XP is: at level 3 the mark is not
	// handed over yet, so the counter keeps running past its own threshold.
	s := New()
	s.XP = 180
	s.Counters["inquisitive"] = 5
	s.Counters["plans"] = 9
	s.Counters["longest_plan"] = 400 // cartographer wants 10
	form, _ := CurrentForm(s)
	if form != "architect" {
		t.Fatalf("form is %q, want architect", form)
	}
	mark, ok := NextMark(s, form)
	if !ok {
		t.Fatal("no mark within reach")
	}
	if mark.Done > mark.Threshold {
		t.Errorf("habit reported %d/%d, over its own threshold",
			mark.Done, mark.Threshold)
	}
	if got := mark.Share(); got > 1 {
		t.Errorf("share is %v, want it capped at 1", got)
	}
}

func TestEveryFormInTheTreeHasASpanishName(t *testing.T) {
	// The canvas names all 41. A form that reaches the panel without one
	// would print its id in the middle of a Spanish sentence.
	seen := map[string]bool{Root: true}
	for parent, kids := range Tree {
		seen[parent] = true
		for _, kid := range kids {
			seen[kid] = true
		}
	}
	for _, secret := range Secrets {
		seen[secret] = true
	}
	if len(seen) != 41 {
		t.Errorf("the tree has %d forms, the canvas draws 27", len(seen))
	}
	for form := range seen {
		if Names[form] == "" {
			t.Errorf("%q has no Spanish name", form)
		}
	}
	for _, temperament := range Temperaments {
		if Names[temperament] == "" {
			t.Errorf("the temperament %q has no Spanish name", temperament)
		}
	}
}

func TestAnUnnamedFormFallsBackToItsID(t *testing.T) {
	if got := Name("godzilla"); got != "godzilla" {
		t.Errorf("Name gave %q, want the id back", got)
	}
}

func TestEveryStateHasASpanishNameThatFitsTheCard(t *testing.T) {
	// The card centres the state in nine columns and adds a sparkle to the
	// freshest one, so a long word would be cut in half on screen.
	const cardWidth = 9
	for _, v := range Vitals {
		name := Names[v.Label]
		if name == "" {
			t.Errorf("the state %q has no Spanish name", v.Label)
			continue
		}
		width := len([]rune(name))
		if v.Sparkle {
			width += 2 // " ✦"
		}
		if width > cardWidth {
			t.Errorf("%q is %d columns on the card, which holds %d",
				name, width, cardWidth)
		}
	}
}

func TestAFutureFeedTimeDoesNotLockTheBowl(t *testing.T) {
	// A clock put forward once, or a hand-edited pet.json, used to leave
	// /feed refusing until real time caught up. The daily counter this
	// replaced could not do that: midnight always came.
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.FedAt = now.Add(72 * time.Hour).Unix()

	if !Feed(s, "feed", "", now) {
		t.Error("a fed_at in the future locked the bowl")
	}
	if left := Waiting(s, "feed", now); left > FeedCooldown {
		t.Errorf("wait of %v, longer than the cooldown itself", left)
	}
}

func TestTheCooldownStillBindsAfterTheFix(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	if !Feed(s, "feed", "", now) {
		t.Fatal("the first feed was refused")
	}
	if Feed(s, "feed", "", now.Add(3*time.Hour)) {
		t.Error("it ate again three hours in, cooldown is four")
	}
	if !Feed(s, "feed", "", now.Add(4*time.Hour+time.Second)) {
		t.Error("it refused after the four hours were up")
	}
}

func TestTheXPHasACeiling(t *testing.T) {
	// Without one the XP is a moat: every penalty drowns in the buffer.
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.XP = XPCeiling - 5
	Feed(s, "tests", "", now) // +15, would land at XPCeiling+10
	if s.XP != XPCeiling {
		t.Errorf("xp went to %d, want it capped at %d", s.XP, XPCeiling)
	}
	if LevelFor(s.XP) != 6 {
		t.Errorf("the ceiling dropped the pet to level %d, want it at the top",
			LevelFor(s.XP))
	}
}

func TestTheCeilingLeavesRoomToFallButNotToHide(t *testing.T) {
	top := Levels[len(Levels)-1].XP
	if XPCeiling <= top {
		t.Fatalf("ceiling %d is not above the last threshold %d", XPCeiling, top)
	}
	// The buffer has to be crossable: a couple of days of neglect, not fifty
	// blown contexts.
	if hours := (XPCeiling - top) / StarveXP; hours > 72 {
		t.Errorf("%d hours of starving to drop a level, too long to ever bite", hours)
	}
}

func TestStarvingDrainsTheXP(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.XP, s.Hunger = 500, 0
	s.LastFed = now.Unix()

	// Ten hours to reach the cap: hunger only, nothing billed yet.
	DecayHunger(s, now.Add(10*time.Hour))
	if s.Hunger != HungerMax {
		t.Fatalf("hunger is %d after ten hours, want the cap", s.Hunger)
	}
	if s.XP != 500 {
		t.Errorf("xp moved to %d before the cap was reached", s.XP)
	}

	// Six more hours, all of them at full hunger.
	DecayHunger(s, now.Add(16*time.Hour))
	if want := 500 - 6*StarveXP; s.XP != want {
		t.Errorf("xp is %d after six starving hours, want %d", s.XP, want)
	}
	if s.Hunger != HungerMax {
		t.Errorf("hunger climbed past its cap to %d", s.Hunger)
	}
}

func TestStarvingCanCostALevelButNeverKills(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.XP, s.Hunger = Levels[4].XP, HungerMax // level 5, at the line
	s.LastFed = now.Unix()

	DecayHunger(s, now.Add(2*time.Hour))
	if LevelFor(s.XP) != 4 {
		t.Errorf("two hours of starving at the line left it at level %d, want 4",
			LevelFor(s.XP))
	}

	// Left alone long enough it bottoms out at the larva and stays there. Never
	// dies. At StarveXP an hour the wall clock needed is the whole ladder, so
	// it is taken off Levels rather than written down: widening level 4 pushed
	// this from about six weeks to about twelve, and a hard-coded fifty days
	// would have turned this into a test that asserts nothing.
	hours := xpFor(6) + 48 // the ladder, and a day's slack for the decay to start
	DecayHunger(s, now.Add(time.Duration(hours)*time.Hour))
	if s.XP != 0 {
		t.Errorf("%d hours of neglect left %d xp, want it bottomed out", hours, s.XP)
	}
	if form, level := CurrentForm(s); form != Root || level != 1 {
		t.Errorf("bottomed out as %s/%d, want the larva at level 1", form, level)
	}
}

func TestAPetThatNeverAteDoesNotStarve(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.XP = 30
	DecayHunger(s, now.Add(400*time.Hour))
	if s.XP != 30 {
		t.Errorf("a newborn that never ate lost xp: %d", s.XP)
	}
}

func TestTheCostOfNeglectIsWhatWeMeantItToBe(t *testing.T) {
	// The balance, written down so a change to StarveXP or the ceiling has to
	// argue with it rather than drift past it.
	top := Levels[len(Levels)-1].XP

	toDropFromTheTop := (XPCeiling - top) / StarveXP
	if toDropFromTheTop < 24 || toDropFromTheTop > 72 {
		t.Errorf("%dh of neglect to lose level 5; wanted a couple of days, "+
			"not an afternoon and not a fortnight", toDropFromTheTop)
	}

	toBottomOut := XPCeiling / StarveXP / 24
	if toBottomOut < 21 {
		t.Errorf("%d days to fall from the top to the larva, too fast for a "+
			"holiday", toBottomOut)
	}

	// Hunger has to reach its cap before any of this starts, so a normal
	// night away costs nothing at all.
	s := New()
	s.XP, s.LastFed = 500, time.Now().Unix()
	DecayHunger(s, time.Now().Add(HungerMax*time.Hour))
	if s.XP != 500 {
		t.Errorf("a night away already cost %d xp", 500-s.XP)
	}
}

func TestAteAtSurvivesTheHungerClock(t *testing.T) {
	// LastFed is walked forward by DecayHunger, so it answers "how long has
	// the hunger been running", not "when did it last eat". A pet left for two
	// days used to report "comió hace 0m".
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	Feed(s, "tests", "", now)
	if s.AteAt != now.Unix() {
		t.Fatalf("eating did not record when: %d", s.AteAt)
	}

	later := now.Add(40 * time.Hour)
	DecayHunger(s, later)
	if s.AteAt != now.Unix() {
		t.Errorf("the hunger clock moved AteAt to %d", s.AteAt)
	}
	if hours := (later.Unix() - s.AteAt) / 3600; hours != 40 {
		t.Errorf("it reads as %dh since the last meal, want 40", hours)
	}
}

func TestAnOlderFileFallsBackToLastFed(t *testing.T) {
	// ate_at did not exist before the starvation drain; last_fed is the best
	// answer such a file has.
	dir := t.TempDir()
	path := filepath.Join(dir, "pet.json")
	if err := os.WriteFile(path, []byte(
		`{"xp":100,"hunger":2,"last_fed":1700000000}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if s := Load(path); s.AteAt != 1700000000 {
		t.Errorf("AteAt is %d, want it to fall back to last_fed", s.AteAt)
	}
}

func TestTheBiggestMealHasTheTightestBrake(t *testing.T) {
	// The table used to be upside down: /feed, worth +3, was the only food on
	// a cooldown, while a green suite at +15 could be repeated every nine
	// seconds. Whatever the numbers become, the big meals cannot be the free
	// ones.
	free, braked := 0, 0
	for name, food := range Foods {
		if food.XP >= 15 && food.Cooldown == 0 {
			t.Errorf("%q is worth %+d xp and has no brake", name, food.XP)
		}
		if food.Cooldown == 0 {
			free++
		} else {
			braked++
		}
	}
	if braked == 0 {
		t.Error("nothing on the table has a brake")
	}
}

func TestEachFoodKeepsItsOwnClock(t *testing.T) {
	// One shared timestamp meant two braked foods would gag each other.
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.Counters["nothing"] = 0

	if !Feed(s, "tests", "", now) {
		t.Fatal("the first suite was refused")
	}
	if !Feed(s, "feed", "", now) {
		t.Error("eating a green suite blocked /feed as well")
	}
	if Feed(s, "tests", "", now.Add(30*time.Minute)) {
		t.Error("a second suite got through inside the cooldown")
	}
	if !Feed(s, "tests", "", now.Add(TestsCooldown+time.Second)) {
		t.Error("the suite was still refused after its cooldown")
	}
}

func TestAnOldFileKeepsItsFeedCooldown(t *testing.T) {
	// fed_at was the only clock there was; a file written by an older build
	// must not come back with /feed already available.
	dir := t.TempDir()
	path := filepath.Join(dir, "pet.json")
	now := time.Now()
	raw := `{"xp":100,"hunger":2,"fed_at":` +
		strconv.FormatInt(now.Add(-time.Hour).Unix(), 10) + `}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Load(path)
	if Feed(s, "feed", "", now) {
		t.Error("an hour after a hand-feed it ate again; the cooldown is four")
	}
}

func TestItNeverSaysTheSameLineTwiceRunning(t *testing.T) {
	// A form carries three lines and SaidMemory is three, so the "not
	// recently" filter empties every third time. Starting over from the whole
	// repertoire let it repeat the line it had just said, which is the one
	// kind of repetition anybody notices.
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, form := range []string{"bughunter", "refactor", "spark", "feral"} {
		s := New()
		s.XP = 300
		previous := ""
		for i := 0; i < 40; i++ {
			at := now.Add(time.Duration(i) * 10 * time.Minute)
			line := say(s, EventBigMeal, form, at, nil)
			if line == "" {
				t.Fatalf("%s went quiet on turn %d", form, i)
			}
			if line == previous {
				t.Errorf("%s said %q twice running, on turn %d", form, line, i)
			}
			previous = line
		}
	}
}

func TestASingleLineRepertoireStillSpeaks(t *testing.T) {
	// The guard above must not gag a form that only has one thing to say.
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	Repertoire["testonly"] = []string{"una sola cosa que decir"}
	defer delete(Repertoire, "testonly")

	s := New()
	s.XP = 300
	for i := 0; i < 3; i++ {
		at := now.Add(time.Duration(i) * 10 * time.Minute)
		if line := Speak(s, EventBigMeal, "testonly", at, nil); line == "" {
			t.Fatalf("a one-line repertoire went quiet on turn %d", i)
		}
	}
}

func TestTheWholeRepertoireGetsAired(t *testing.T) {
	// Not repeating must not turn into only ever saying two of the three.
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s := New()
	s.XP = 300
	seen := map[string]int{}
	for i := 0; i < 300; i++ {
		at := now.Add(time.Duration(i) * 10 * time.Minute)
		seen[Speak(s, EventBigMeal, "bughunter", at, nil)]++
	}
	for _, line := range Repertoire["bughunter"] {
		if seen[line] == 0 {
			t.Errorf("%q never came up in 300 turns", line)
		}
	}
}

// reach builds the smallest state that should produce the given form: the XP
// its depth needs, the branch counter of every step on its way, and its own
// habit if it is a mark. Siblings are left at zero so the fork is unambiguous.
func reach(form string) *State {
	s := New()
	trail := Lineage(form)
	switch len(trail) {
	case 1:
		s.XP = 0
	case 2:
		s.XP = Levels[1].XP
	case 3:
		s.XP = Levels[2].XP
	case 4:
		s.XP = Levels[4].XP // a mark opens at level 5
	default:
		s.XP = Levels[len(Levels)-1].XP
	}
	for _, step := range trail {
		if counter, ok := BranchBy[step]; ok {
			s.Counters[counter] = 999
		}
		if u, ok := Unlocks[step]; ok {
			s.Counters[u.Counter] = u.Threshold
		}
		// A title asks for a good deal more of its mark's habit, so the state
		// that reaches one has to carry the bigger number.
		if parent, ok := Parent[step]; ok && Titles[parent] == step {
			if u, ok := TitleUnlock(parent); ok {
				s.Counters[u.Counter] = u.Threshold
			}
		}
	}
	return s
}

func TestEveryFormCanActuallyBeReached(t *testing.T) {
	// Having a sprite and having an unlock is not the same as being
	// reachable: a mark whose counter nobody feeds, or a fork whose sibling
	// always wins, would be a form nobody can ever see.
	for _, form := range allForms() {
		if form == "phoenix" || form == "chimera" {
			continue // off the tree, covered below
		}
		s := reach(form)
		if got, _ := CurrentForm(s); got != form {
			t.Errorf("%s is unreachable: that state gives %s", form, got)
		}
	}
}

func TestBothSecretsCanBeReached(t *testing.T) {
	for _, secret := range Secrets {
		s := New()
		s.XP = Levels[4].XP // level 5: a secret is not a title
		s.Secret = Secret(secret)
		if got, level := CurrentForm(s); got != secret || level != 5 {
			t.Errorf("%s is unreachable: gives %s/%d", secret, got, level)
		}
	}
}

func TestEveryCounterTheTreeNeedsHasAFeeder(t *testing.T) {
	// A counter nothing ever increments makes its form unreachable in
	// practice, however well the tree is wired.
	needed := map[string]string{}
	for form, counter := range BranchBy {
		needed[counter] = "the fork to " + form
	}
	for form, u := range Unlocks {
		needed[u.Counter] = "the mark " + form
	}

	// Everything a meal bumps, plus what the hook and the statusline bump by
	// hand. Kept as data so a counter that loses its feeder shows up here.
	fed := map[string]bool{
		// hook/app.go and statusline/card.go
		"repro_before_fix": true, "plans_before_code": true, "longest_plan": true,
		"single_tool_tasks": true, "sessions_under_40": true, "ctx_low": true,
		"sessions_15min": true, "short_sessions": true, "sessions_4h": true,
		"long_sessions": true, "same_repo_days": true, "docs_days": true,
		"widest_commit": true, "ctx100_sessions": true, "bypass_turns": true,
		// The ember branch, all three notches of it, paid from the session's
		// context peak in hook.CloseSession. They used to hang off the overflow
		// meal, which is the one that TAKES XP: see the comment there.
		"impulsive": true, "ctx_maxed": true,
	}
	for _, food := range Foods {
		for _, habit := range food.Habits {
			fed[habit] = true
		}
	}

	for counter, why := range needed {
		if !fed[counter] {
			t.Errorf("nothing feeds %q, so %s can never happen", counter, why)
		}
	}
}

func allForms() []string {
	seen := map[string]bool{Root: true}
	for parent, kids := range Tree {
		seen[parent] = true
		for _, kid := range kids {
			seen[kid] = true
		}
	}
	for _, s := range Secrets {
		seen[s] = true
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	return out
}

// The note on a meal comes from outside - a commit subject, the text of a plan
// task - and both the panel and the speech bubble put it on a row with other
// things. A newline in it broke the panel's log entry into three rows, with the
// time stranded on the last one.
func TestAMealsNoteIsFlattenedBeforeItIsStored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pet.json")
	now := time.Now()
	Update(path, func(s *State) bool {
		return Feed(s, "task", "arreglar\nla\nmascota", now)
	})

	log := Load(path).Log
	if len(log) != 1 {
		t.Fatalf("want one meal, got %d", len(log))
	}
	if got := log[0].Note; strings.ContainsAny(got, "\n\r\t") {
		t.Errorf("the note kept its control characters: %q", got)
	} else if got != "arreglar la mascota" {
		t.Errorf("note %q, want %q", got, "arreglar la mascota")
	}
}
