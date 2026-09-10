package pet

import (
	"time"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Losing at ccpet invade, and what it costs the creature.

// DefeatEvent is the log entry a lost run leaves. A constant because the panel
// looks it up to label the row and the game writes it: two literals of the same
// string is one rename away from a panel that prints "defeat" in Spanish.
const DefeatEvent = "defeat"

// SetbackMax is the most XP one lost run may cost: a day's feeding, read off the
// ladder rather than written down.
//
// It exists because the two halves of this measured badly together. A run at
// level 4 lasts about three minutes before the swarm outgrows the kit, and level
// 4 is deliberately the widest rung on the tree - 1600 XP, about twelve days of
// real work. Dropping straight to the threshold would have charged those twelve
// days for one bad three-minute run, over and over. With the cap the level still
// goes whenever you were near the line, and a full buffer is never wiped.
var SetbackMax = Levels[1].XP

// Setback is what losing a run costs: the level you were standing on.
//
// It is not in Foods, and that is deliberate twice over. A meal is something you
// can order - panel.Run dispatches its verbs by membership of that map, so a
// "defeat" food would hand anybody a `ccpet defeat` that punishes the pet by
// hand, and would show up in the panel's table of what it eats. And a meal's XP
// is a fixed number in a table, while this one has to be whatever it takes to
// cross the threshold below.
//
// The XP goes to one point under the threshold of the level it was on - so the
// level goes now and the next decent meal brings it back - but never by more
// than SetbackMax, which is what stops a run three minutes long from charging a
// fortnight. The effect is that dying near a threshold costs you the level and
// dying deep inside one costs you a day, which is the right way round: the
// player with the most to lose is the one who just got there.
//
// The two obvious alternatives are worse. A fixed cost like the overflow's -15
// does not reliably cost a level at all, and dropping to the PREVIOUS threshold
// is 1600 XP between level 5 and 4, which is not a stake, it is a reason never
// to open the game again.
//
// What it must NOT touch is the longer list: hunger and its clock, the meal
// timestamps, the streak, and above all the habit counters. Those decide which
// way the tree forks, and a game that could move them would be a game that
// changes what the creature is growing into. See
// TestASetbackNeverMovesAHabitCounter.
//
// The shape does not follow the level down either, and that is not this
// function's doing: FormSeen and floor already hold the rung, and
// TestTheShapeHoldsWhileTheLevelIsStillAllowedToFall is what keeps it true. You
// lose a step of the kit, not your evolution.
//
// It returns the XP taken, which is 0 for a larva with no level left to lose -
// and 0 is what tells pet.Update there is nothing to write.
func Setback(s *State, note string, now time.Time) int {
	level := LevelFor(s.XP)
	if level <= 1 {
		return 0
	}
	cost := s.XP - (Levels[level-1].XP - 1)
	if cost > SetbackMax {
		cost = SetbackMax
	}
	s.XP -= cost

	// The log is today's, the same way Feed keeps it: without this a defeat
	// lands at the end of yesterday's rows.
	if day := Today(now); s.LogDay != day {
		s.LogDay = day
		s.Log = s.Log[:0]
	}
	note = theme.OneLine(note)
	if len([]rune(note)) > 40 {
		note = truncate(note, 40)
	}
	s.Log = append(s.Log, LogEntry{Event: DefeatEvent, XP: -cost, At: now.Unix(), Note: note})
	if len(s.Log) > LogMax {
		s.Log = s.Log[len(s.Log)-LogMax:]
	}
	return cost
}
