package pet

import (
	"testing"
	"time"
	"unicode/utf8"
)

// allEvents is every event with shared lines.
var allEvents = []Event{
	EventHatch, EventHungry, EventLevelUp, EventStreak, EventCtxBlown,
	EventCompact, EventTask, EventFeed,
	EventMorning, EventLate, EventFriday, EventLong, EventTired, EventDrowning,
}

func TestNoLineIsSaidByTwoVoicesOrOverflowsTheBand(t *testing.T) {
	// The longest line the canvas wrote is 57 characters. Band 4 drops the
	// bubble when it does not fit, so a longer line is one that gets read on
	// fewer terminals than any of the originals.
	const longest = 57
	seen := map[string]string{}
	check := func(who, line string) {
		if other, ok := seen[line]; ok {
			t.Errorf("%q is said by both %s and %s", line, other, who)
		}
		seen[line] = who
		if n := utf8.RuneCountInString(line); n > longest {
			t.Errorf("%s: %q is %d characters, the band holds %d", who, line, n, longest)
		}
	}
	for form, lines := range Repertoire {
		for _, line := range lines {
			check(form, line)
		}
	}
	for _, e := range allEvents {
		lines := shared(e, 6, 7)
		if len(lines) < 3 {
			t.Errorf("%s has %d ways of being said, want at least 3", e, len(lines))
		}
		for _, line := range lines {
			check(string(e), line)
		}
	}
}

func TestAnEventIsNotAlwaysSaidTheSameWay(t *testing.T) {
	// Hunger is a standing state: it is said again every cooldown, and it
	// used to be the same sentence every time.
	s := New()
	s.Hunger = HungerWarn
	said := map[string]bool{}
	for i := 0; i < 4; i++ {
		at := t0.Add(time.Duration(i) * 10 * time.Minute)
		said[say(s, EventHungry, "loom", at, nil)] = true
	}
	if len(said) < 3 {
		t.Errorf("four times hungry, %d different lines", len(said))
	}
}

func TestASmallMealSpeaksAsItself(t *testing.T) {
	for meal, want := range map[string]Event{
		"compact": EventCompact, "task": EventTask, "feed": EventFeed,
		"commit": EventNothing, "tests": EventNothing, "overflow": EventNothing,
	} {
		if got := SnackEvent(meal); got != want {
			t.Errorf("SnackEvent(%q) = %q, want %q", meal, got, want)
		}
	}
}

func TestTheMomentIsRemarkedOnOnceADay(t *testing.T) {
	friday := time.Date(2026, 11, 20, 17, 30, 0, 0, time.Local)
	s := New()
	s.Log = []LogEntry{{Event: "commit", XP: 12}, {Event: "commit", XP: 12}}
	m := Moment{Now: friday, Meal: "commit"}
	if e := Ambient(s, m); e != EventFriday {
		t.Fatalf("a commit on a Friday afternoon: %q, want friday", e)
	}
	RememberEvent(s, EventFriday, friday)
	if e := Ambient(s, m); e != EventNothing {
		t.Errorf("said it once today and now %q again", e)
	}
	m.Now = friday.AddDate(0, 0, 7)
	if e := Ambient(s, m); e != EventFriday {
		t.Errorf("the Friday after: %q, want friday", e)
	}
}

func TestTheClockOnlyTalksToSomebodyWorking(t *testing.T) {
	night := time.Date(2026, 11, 18, 2, 0, 0, 0, time.Local)
	s := New()
	s.Log = []LogEntry{{Event: "commit", XP: 12}, {Event: "commit", XP: 12}}
	if e := Ambient(s, Moment{Now: night}); e != EventNothing {
		t.Errorf("two in the morning and no meal: %q, want silence", e)
	}
	if e := Ambient(s, Moment{Now: night, Meal: "commit"}); e != EventLate {
		t.Errorf("a commit at two in the morning: %q, want late", e)
	}
}

func TestWhatTheMomentPutsFirst(t *testing.T) {
	noon := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)
	s := New()
	s.Log = []LogEntry{{Event: "commit", XP: 12}}
	// The first meal of the day, in a session already three hours long, with
	// the context going under: the one that can still be fixed wins.
	m := Moment{Now: noon, Meal: "commit", Label: "drowning", Active: 4 * time.Hour}
	for _, want := range []Event{EventDrowning, EventMorning, EventLong, EventNothing} {
		got := Ambient(s, m)
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
		RememberEvent(s, got, noon)
	}
	// The state is read as it stands - the crossing itself lasts one refresh,
	// and may well fall inside a cooldown - and once said it is not news again.
	tired := New()
	if e := Ambient(tired, Moment{Now: noon, Label: "tired"}); e != EventTired {
		t.Errorf("tired: %q", e)
	}
	RememberEvent(tired, EventTired, noon)
	if e := Ambient(tired, Moment{Now: noon, Label: "tired"}); e != EventNothing {
		t.Errorf("still tired, already said: %q", e)
	}
}

func TestOnlyTheDailyEventsAreBooked(t *testing.T) {
	s := New()
	RememberEvent(s, EventBigMeal, t0)
	RememberEvent(s, EventHungry, t0)
	if len(s.DayMarks) != 0 {
		t.Errorf("a meal and hunger left day marks: %v", s.DayMarks)
	}
}

func TestTheDayMarksOfSpeechSurviveTheRoundTrip(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := New()
	RememberEvent(s, EventMorning, t0)
	if !Save(s, path) {
		t.Fatal("could not save")
	}
	if e := Ambient(Load(path), Moment{Now: t0, Meal: "commit"}); e == EventMorning {
		t.Error("said good morning, saved, loaded, and said it again")
	}
}
