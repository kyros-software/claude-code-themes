package pet

import (
	"time"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// What counts as food, and what eating it does to the pet.

// HungerMax is the cap; HungerWarn is where the eyes go out and it asks to eat.
const (
	HungerMax  = 10
	HungerWarn = 7
)

// StarveXP is what an hour at full hunger costs. Hunger used to stop at the
// cap and sit there for free, which is why a pet could be left for a week and
// come back exactly as strong: the only thing that ever took XP away was
// blowing the context, at -15 a time.
const StarveXP = 1

// XPCeiling is as far as the XP goes: the last threshold plus XPHeadroom.
// Without a ceiling the XP is a moat - at 1641 with the top at 900 it took
// FIFTY blown contexts to drop a level, so every penalty we could add would be
// swallowed by the buffer before it meant anything.
//
// The headroom was "one level-1 stretch", which was 60 and meant two and a half
// days of starving to lose the top level. v3 moved level 2 to 200 and that
// definition to 200 hours, so the number the design actually meant is now
// written down as itself.
const XPHeadroom = 60

var XPCeiling = Levels[len(Levels)-1].XP + XPHeadroom

// FeedCooldown is how long /feed makes you wait. The design canvas calls it
// "un tiro al plato · una vez cada cuatro horas": a cooldown, not a daily
// counter, which is the same clock hunger already runs on.
const FeedCooldown = 4 * time.Hour

// TestsCooldown is how often a green suite counts.
//
// It is the biggest meal on the table and it had no brake at all, which made
// it the one thing worth farming: running the suite in a loop fed the pet
// +15 every few seconds - 120 XP in eight minutes, measured - so the ceiling
// and the starvation drain were decoration. Nothing that takes nine seconds
// to repeat can be worth a fifteenth of a level.
//
// An hour is not arbitrary: the canvas budgets level 5 at "una semana de uso
// normal", which is about 128 XP a day, and eight green suites in a working
// day is exactly that.
const TestsCooldown = time.Hour

// Food is one meal's effect. A Cooldown of 0 means you can eat it whenever the
// thing that earns it happens.
type Food struct {
	XP       int
	Hunger   int
	Cooldown time.Duration
	Label    string
	Habits   []string
}

// Foods, by event name.
var Foods = map[string]Food{
	"tests":   {15, -4, TestsCooldown, "tests en verde", []string{"inquisitive", "tests", "test_streak"}},
	"commit":  {12, -3, 0, "commit", []string{"methodical", "diffs", "diff_streak"}},
	"compact": {8, -3, 0, "compact", []string{"methodical"}},
	// The same meal when the CLI compacts on its own. It still feeds - the XP
	// budget was calibrated with compacts in it - but not `methodical`: a
	// window that fills until the CLI steps in is the opposite of "compacta y
	// commitea corto". Measured in one user's transcripts (7-nov-2026): 47
	// compacts, all 47 automatic, every one paid to `methodical`.
	//
	// v3: it pays `impulsive` instead, because it IS the ember gesture - "tira
	// al límite sin frenar" - and the only other source, a session peak at 85%,
	// came to 7 in some 650 sessions on that machine. The ember branch was a
	// third of the tree that a pet working that way could not see.
	"autocompact": {8, -3, 0, "compact automático", []string{"impulsive"}},
	"task":        {6, -1, 0, "tarea del plan", []string{"inquisitive", "plans"}},
	"feed":        {3, -2, FeedCooldown, "/feed", nil},
	// The overflow feeds NO habit. It used to carry impulsive and ctx_maxed,
	// which is what closed the arithmetic on the whole ember branch: the only
	// way to earn those two counters was the only meal that takes XP away. Both
	// are paid from the session's context peak now, in hook.CloseSession, so
	// this is what it always should have been - a penalty, and a streak breaker.
	"overflow": {-15, 0, 0, "contexto al 100%", nil},
}

// BigMeal is what the pet considers worth talking about: a green suite or a
// commit, not a compact or a hand-feed. See speech.go.
func BigMeal(event string) bool { return Foods[event].XP >= 12 }

// BigMealWindow is how long after eating one the pet still has something to
// say about it - which is how long the bubble stays on screen.
//
// It was ten seconds, chosen as "a refresh or two", and ten seconds is not
// long enough to be seen: it is the only door that opens at all for somebody
// who commits regularly, keeps a full belly and has no streak going, and it
// shut before they looked up. A minute is still an event and not a chat, and
// the five-minute cooldown is what actually keeps the pet quiet.
const BigMealWindow = 60 * time.Second

// streaksBrokenBy: a clean streak breaks when you blow the context, and the
// test streak also breaks on a red suite.
//
// "red" is not a food - nothing is eaten - so it never goes through Feed: the
// hook calls Break for it. It used to break nothing at all, which left
// `exterminator` ("15 tests verdes sin uno rojo") counting green suites in
// total. Measured on a real pet.json: test_streak equal to tests, 175 and 175,
// with 171 red -> green cycles booked in the same file.
var streaksBrokenBy = map[string][]string{
	"overflow": {"test_streak", "diff_streak"},
	"red":      {"test_streak"},
}

// Break zeroes the streaks a cause breaks, and reports whether any moved.
func Break(s *State, cause string) bool {
	moved := false
	for _, counter := range streaksBrokenBy[cause] {
		if s.Counters[counter] != 0 {
			s.Counters[counter] = 0
			moved = true
		}
	}
	return moved
}

// DecayHunger adds one hunger per hour without food, up to HungerMax. It does
// not kill: it only puts the eyes out.
func DecayHunger(s *State, now time.Time) {
	if s.LastFed == 0 {
		return
	}
	if hours := (now.Unix() - s.LastFed) / 3600; hours > 0 {
		s.Hunger += int(hours)
		// The overflow past the cap IS the count of hours spent starving, so
		// no second timestamp is needed to bill them.
		if starved := s.Hunger - HungerMax; starved > 0 {
			s.Hunger = HungerMax
			if s.XP -= starved * StarveXP; s.XP < 0 {
				s.XP = 0
			}
		}
		s.LastFed += hours * 3600
	}
	// The hunger peak is recorded HERE, where it actually rises. Recording it
	// only on eating misses exactly the moment the phoenix cares about.
	if s.Hunger > s.HungerPeak {
		s.HungerPeak = s.Hunger
	}
}

// Feed applies one meal and reports whether it landed.
func Feed(s *State, event, note string, now time.Time) bool {
	food, ok := Foods[event]
	if !ok {
		return false
	}
	// The day comes from now and not from the clock: otherwise passing another
	// time leaves the streak incoherent and there is no way to test it.
	day := Today(now)
	if s.LogDay != day {
		s.LogDay = day
		s.Log = s.Log[:0]
	}
	// The cooldown keeps its own timestamp: the log is capped at LogMax entries
	// and cleared daily, so reading the last meal out of it would forget a
	// feed at 23:00 the moment midnight passed.
	if food.Cooldown > 0 && waited(s, event, now) < food.Cooldown {
		// The cooldown brakes the XP, not the habit. A green suite an hour
		// after the last is still a green suite, and the race between branches
		// reads the habit: with the brake on it too, `tests` could not outrun
		// commits, which have none. Measured: 451 test runs in a month of
		// transcripts against a lifetime `tests` of 175.
		for _, counter := range food.Habits {
			s.Bump(counter, 1)
		}
		return false
	}

	if s.XP += food.XP; s.XP < 0 {
		s.XP = 0
	}
	if s.XP > XPCeiling {
		s.XP = XPCeiling
	}
	if food.Hunger != 0 {
		s.Hunger += food.Hunger
		if s.Hunger < 0 {
			s.Hunger = 0
		}
		if s.Hunger > HungerMax {
			s.Hunger = HungerMax
		}
	}
	if food.XP > 0 {
		s.LastFed = now.Unix()
		s.AteAt = now.Unix()
	}
	if food.Cooldown > 0 {
		if s.Meals == nil {
			s.Meals = map[string]int64{}
		}
		s.Meals[event] = now.Unix()
		// FedAt is kept in step so an older build, which knows only this one
		// clock, still honours the /feed cooldown.
		if event == "feed" {
			s.FedAt = now.Unix()
		}
	}

	if s.LastDay != day {
		if s.LastDay == Yesterday(now) {
			s.Streak++
		} else {
			s.Streak = 1
		}
		if s.Streak > s.BestStreak {
			s.BestStreak = s.Streak
		}
		s.LastDay = day

		// A day of use wearing the title. Past TitleStay the generation closes,
		// and this meal is the new egg's first.
		if s.TitleDays > 0 && AtTitle(s) {
			s.TitleDays++
			if s.TitleDays > TitleStay {
				Hatch(s, now.Unix())
				if food.XP > 0 {
					s.XP = food.XP
				}
			}
		}
	}

	// Flattened before it is stored, not before it is drawn: this note comes
	// from outside - a commit subject, the text of a plan task - and both the
	// panel and the speech bubble put it on a row with other things. A newline
	// in it broke the panel's log entry into three rows, with the time left
	// stranded on the last one. See theme.OneLine.
	note = theme.OneLine(note)
	if len(note) > 40 {
		note = truncate(note, 40)
	}
	s.Log = append(s.Log, LogEntry{Event: event, XP: food.XP, At: now.Unix(), Note: note})
	if len(s.Log) > LogMax {
		s.Log = s.Log[len(s.Log)-LogMax:]
	}

	// Secrets are checked BEFORE the habit counters move: the chimera compares
	// temperaments, and bumping first would trip it one meal early.
	CheckSecrets(s)

	for _, counter := range food.Habits {
		s.Bump(counter, 1)
	}
	for _, counter := range streaksBrokenBy[event] {
		s.Counters[counter] = 0
	}
	return true
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// waited is how long since the last hand-feed. A fed_at in the FUTURE - a
// clock put forward once, a hand-edited pet.json - would otherwise read as a
// negative wait and lock /feed until the clock caught up, which the daily
// counter this replaces could never do because midnight always came.
func waited(s *State, event string, now time.Time) time.Duration {
	at := s.Meals[event]
	if at == 0 && event == "feed" {
		at = s.FedAt
	}
	if at == 0 {
		return 1<<62 - 1
	}
	since := now.Sub(time.Unix(at, 0))
	if since < 0 {
		return 1<<62 - 1
	}
	return since
}

// Waiting is how long is left on a food's cooldown, or 0 if it can eat now.
func Waiting(s *State, event string, now time.Time) time.Duration {
	food, ok := Foods[event]
	if !ok || food.Cooldown == 0 {
		return 0
	}
	left := food.Cooldown - waited(s, event, now)
	if left < 0 {
		return 0
	}
	return left
}

// CheckSecrets looks for the two level-5 forms that are not on the tree. Called
// on eating, which is the only thing that moves hunger and XP.
func CheckSecrets(s *State) {
	if s.Secret != "" {
		return
	}
	if s.Hunger > s.HungerPeak {
		s.HungerPeak = s.Hunger
	}

	// phoenix: touch hunger 10 and come back to 0 in the SAME session, and only
	// from the two forms allowed to get that far. HungerPeak is zeroed when the
	// session closes.
	if s.Hunger == 0 && s.HungerPeak >= HungerMax {
		// walk, not CurrentForm: this asks which branch the counters put the
		// pet on today, and CurrentForm answers a different question - what it
		// LOOKS like, which never goes back down a rung. See pet.walk.
		if form, _ := walk(s); form == "feral" || form == "marathon" {
			s.Secret = "phoenix"
			return
		}
	}

	// chimera: two temperaments tied on reaching level 4. It inherits one's
	// eyes and the other's body, which is what its sprite draws.
	//
	// Tied the way the FORK reads a tie, which since BranchScale is not the
	// raw counters: `methodical` counts commits and `impulsive` counts
	// sessions, so the two landing on the same number is a coincidence of
	// units, not two ways of working that came out level. The pet this was
	// handing chimeras to was the ordinary one whose commits happened to cross
	// its suites.
	//
	// Scaled to the common denominator instead of compared as floats, so "tied"
	// stays exact integer arithmetic and does not depend on which pair of
	// scales happens to divide evenly.
	if LevelFor(s.XP) >= 4 {
		den := 1
		for _, t := range Temperaments {
			den *= scaleOf(t)
		}
		var top [3]int
		for i, t := range Temperaments {
			top[i] = s.Stage(t) * (den / scaleOf(t))
		}
		// three values: sort by hand rather than pull in sort for this
		for i := 0; i < 3; i++ {
			for j := i + 1; j < 3; j++ {
				if top[j] > top[i] {
					top[i], top[j] = top[j], top[i]
				}
			}
		}
		if top[0] > 0 && top[0] == top[1] {
			s.Secret = "chimera"
		}
	}
}
