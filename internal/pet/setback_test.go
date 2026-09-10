package pet

import (
	"testing"
	"time"
)

var defeatClock = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// Losing a run takes the level you were standing on, and exactly that: one
// rung, not two, and not a fixed number of XP that might not cost a level at
// all. Read off Levels rather than written down, so widening the ladder cannot
// leave this test asserting nothing.
func TestLosingCostsTheLevelYouWereStandingOn(t *testing.T) {
	for level := 2; level <= 6; level++ {
		s := New()
		s.XP = xpFor(level)
		cost := Setback(s, "wave 12", defeatClock)
		if got := LevelFor(s.XP); got != level-1 {
			t.Errorf("from level %d it landed on %d, want %d", level, got, level-1)
		}
		if cost != 1 {
			t.Errorf("standing on the threshold of level %d it took %d xp, want 1",
				level, cost)
		}
	}

	// And from just inside a level, where the drop still fits under the cap.
	s := New()
	s.XP = xpFor(4) + SetbackMax - 1
	if got := Setback(s, "", defeatClock); got != SetbackMax {
		t.Errorf("a whisker inside level 4 it took %d, want the cap of %d", got, SetbackMax)
	}
	if got := LevelFor(s.XP); got != 3 {
		t.Errorf("it landed on level %d, want 3", got)
	}
}

// One run is three minutes and level 4 is twelve days wide. Without a ceiling a
// bad afternoon would charge the fortnight, over and over, which is not a stake
// but a reason to stop opening the game. So the level goes when you were near
// the line, and a full buffer is never wiped.
func TestADefeatNeverCostsMoreThanADayOfFeeding(t *testing.T) {
	for level := 2; level <= 6; level++ {
		top := xpFor(level+1) - 1
		if level == 6 {
			top = xpFor(6) + 3000
		}
		for _, xp := range []int{xpFor(level), xpFor(level) + SetbackMax, top} {
			s := New()
			s.XP = xp
			cost := Setback(s, "", defeatClock)
			if cost > SetbackMax {
				t.Errorf("at %d xp it took %d, past the ceiling of %d", xp, cost, SetbackMax)
			}
			if s.XP < 0 {
				t.Errorf("at %d xp it left %d", xp, s.XP)
			}
			if LevelFor(s.XP) < level-1 {
				t.Errorf("at %d xp it fell from level %d to %d", xp, level, LevelFor(s.XP))
			}
		}
	}

	// Deep inside the widest rung on the tree it costs a day and no more, and
	// the level survives - which is the half of the trade that is not obvious.
	s := New()
	s.XP = xpFor(5) - 1 // one point off level 5, deep in the 1600-wide level 4
	if cost := Setback(s, "", defeatClock); cost != SetbackMax {
		t.Errorf("one point off level 5 it took %d, want %d", cost, SetbackMax)
	}
	if got := LevelFor(s.XP); got != 4 {
		t.Errorf("it fell to level %d; deep inside a level the level is meant to hold", got)
	}
}

// Near the line the level really does go, which is the half that makes it a
// stake at all.
func TestNearTheLineTheLevelReallyGoes(t *testing.T) {
	for level := 2; level <= 6; level++ {
		s := New()
		s.XP = xpFor(level) + SetbackMax/2
		Setback(s, "", defeatClock)
		if got := LevelFor(s.XP); got != level-1 {
			t.Errorf("half a day into level %d it stayed on %d", level, got)
		}
	}
}

// A larva has nothing under it. Losing has to be survivable at the bottom or
// the game becomes unplayable for exactly the pet most likely to be playing it.
func TestALarvaHasNoLevelLeftToLose(t *testing.T) {
	for _, xp := range []int{0, 1, xpFor(2) - 1} {
		s := New()
		s.XP = xp
		if cost := Setback(s, "wave 1", defeatClock); cost != 0 {
			t.Errorf("at %d xp it took %d, want nothing", xp, cost)
		}
		if s.XP != xp {
			t.Errorf("at %d xp it left %d", xp, s.XP)
		}
		if len(s.Log) != 0 {
			t.Errorf("at %d xp it wrote a log entry: %+v", xp, s.Log)
		}
	}
}

// The level falls and the shape does not. This is the whole reason the stake is
// playable: you lose a step of the kit, not the evolution you spent a week on.
// FormSeen and floor already guarantee it - this asserts the defeat path goes
// through them and not around.
func TestASetbackNeverTakesTheShapeDownTheTree(t *testing.T) {
	s := New()
	s.XP = xpFor(6)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": TitleAsks["wasp"]}
	form, level := CurrentForm(s)
	if form != "wasp" || level != 6 {
		t.Fatalf("the starting point is %s at level %d, want a wasp at 6", form, level)
	}
	RememberForm(s, form)

	Setback(s, "wave 40", defeatClock)

	got, gotLevel := CurrentForm(s)
	if got != "wasp" {
		t.Errorf("after one defeat it is a %s, want the wasp it still is", got)
	}
	if gotLevel != 5 {
		t.Errorf("after one defeat it is level %d, want 5", gotLevel)
	}
}

// A defeat is not a meal. It must not feed the hunger clock, count as having
// eaten, touch a cooldown or move the day streak - all of which Feed does, and
// any one of which would let a run of the game keep the pet alive.
func TestASetbackIsNotAMealAndDoesNotResetTheHungerClock(t *testing.T) {
	s := New()
	s.XP = xpFor(5)
	s.Hunger, s.LastFed, s.AteAt = 6, 111, 222
	s.Streak, s.BestStreak, s.LastDay = 4, 9, "2026-02-28"
	s.Meals = map[string]int64{"feed": 333}
	s.FedAt = 333
	before := s.Clone()

	Setback(s, "wave 7", defeatClock)

	for _, c := range []struct {
		name     string
		got, was any
	}{
		{"hunger", s.Hunger, before.Hunger},
		{"last_fed", s.LastFed, before.LastFed},
		{"ate_at", s.AteAt, before.AteAt},
		{"fed_at", s.FedAt, before.FedAt},
		{"streak", s.Streak, before.Streak},
		{"best_streak", s.BestStreak, before.BestStreak},
		{"last_day", s.LastDay, before.LastDay},
		{"meals", s.Meals["feed"], before.Meals["feed"]},
	} {
		if c.got != c.was {
			t.Errorf("%s moved: %v, was %v", c.name, c.got, c.was)
		}
	}
}

// The worst thing this function could possibly do. The habit counters decide
// which way the tree forks, so a defeat that moved one would let losing at a
// game change what the creature is growing into - silently, and only visible a
// week later when it evolved into something nobody chose.
func TestASetbackNeverMovesAHabitCounter(t *testing.T) {
	s := veteran()
	RememberForm(s, "wasp")
	before := s.Clone()

	Setback(s, "wave 40", defeatClock)

	if !sameMap(s.Counters, before.Counters) {
		t.Errorf("the counters moved:\n got %v\nwant %v", s.Counters, before.Counters)
	}
	if !sameMap(s.Branch, before.Branch) {
		t.Errorf("the branch choice moved: %v, was %v", s.Branch, before.Branch)
	}
	if s.FormSeen != before.FormSeen {
		t.Errorf("form_seen moved: %q, was %q", s.FormSeen, before.FormSeen)
	}
	if s.Secret != before.Secret {
		t.Errorf("the secret moved: %q, was %q", s.Secret, before.Secret)
	}
}

// XP has a floor at zero everywhere else in this package and it has one here.
func TestXPNeverGoesNegativeOnADefeat(t *testing.T) {
	for level := 1; level <= 6; level++ {
		s := New()
		s.XP = xpFor(level)
		Setback(s, "", defeatClock)
		if s.XP < 0 {
			t.Errorf("from level %d it left %d xp", level, s.XP)
		}
	}
}

// The row has to be readable and it has to be findable. It carries a negative
// XP so the panel tints it as a loss, and it says which wave it was.
func TestTheDefeatIsLoggedAsALossWithItsWave(t *testing.T) {
	s := New()
	s.XP = xpFor(4)
	cost := Setback(s, "wave 18", defeatClock)

	if len(s.Log) != 1 {
		t.Fatalf("it wrote %d log entries, want one", len(s.Log))
	}
	e := s.Log[0]
	if e.Event != DefeatEvent {
		t.Errorf("the entry is a %q, want a %q", e.Event, DefeatEvent)
	}
	if e.XP != -cost {
		t.Errorf("the entry says %d xp and it took %d", e.XP, cost)
	}
	if e.Note != "wave 18" {
		t.Errorf("the note is %q, want the wave", e.Note)
	}
	if e.At != defeatClock.Unix() {
		t.Errorf("the entry is stamped %d, want %d", e.At, defeatClock.Unix())
	}
}

// A note comes from the game rather than from a person, but it lands on a row
// beside other things, and the panel has been broken once already by a newline
// arriving in one. See theme.OneLine and Feed's own flattening.
func TestADefeatNoteIsFlattenedAndCut(t *testing.T) {
	s := New()
	s.XP = xpFor(5)
	Setback(s, "wave 9\nand a second line that also runs well past the forty rune cut", defeatClock)

	note := s.Log[0].Note
	if len([]rune(note)) > 40 {
		t.Errorf("the note is %d runes: %q", len([]rune(note)), note)
	}
	for _, r := range note {
		if r == '\n' || r == '\r' {
			t.Errorf("the note still carries a control character: %q", note)
		}
	}
}

// A defeat is not something you can order. It is deliberately not in Foods,
// because panel.Run dispatches its meal verbs by membership of that map: a food
// called "defeat" would be a hand-typed `ccpet defeat` that punishes the pet,
// and a row in the panel's table of what it eats.
func TestADefeatIsNotSomethingYouCanOrder(t *testing.T) {
	if _, ok := Foods[DefeatEvent]; ok {
		t.Error("defeat is in Foods, which makes it a command anybody can run")
	}
}

// sameMap compares two maps treating a nil one as an empty one. Clone
// materialises the containers it copies, so a state built with a nil Branch and
// its own clone differ under reflect.DeepEqual while meaning the same thing -
// which is a property of Clone, not something for this test to assert.
func sameMap[V comparable](a, b map[string]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}
