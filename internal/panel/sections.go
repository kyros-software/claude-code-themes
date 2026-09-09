package panel

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The sections /pet grew when it stopped being a summary.
//
// It showed four numbers - xp, hunger, streak, and the habit of whichever mark
// was in the lead - out of the twenty-odd that actually decide the tree. The
// rest moved in silence: there was no way to see what a mark asked for, which
// sibling it beat, or why a meal was refused, short of reading the source.

// counterRow is one habit: its name, its number, and what that number is FOR.
type counterRow struct {
	label string
	count string
	goal  string       // "/10" when it opens something, empty when it does not
	tint  theme.Colour // what the number is painted
}

// tintFor turns a counter's goal into a colour, and that is the whole point of
// the colour: it MEANS something.
//
//	green   the threshold is met - this one is paid for, only the XP is left
//	amber   on its way to a shape this pet can still reach
//	white   counted, and leading nowhere from here
//
// The last is not a failure, it is most of the table: bypass_turns opens
// `gremlin`, which hangs off the ember branch a probe walked past long ago.
// Painting it like the two that are live is what made twenty-two identical
// white numbers out of a list where a third of them matter.
func tintFor(g pet.Goal) theme.Colour {
	switch {
	case g.Reached():
		return theme.Ident
	case g.Leads():
		return theme.Number
	}
	return theme.Emph
}

// pairs lays counters out in two columns, label left, number right, so a long
// list reads as a table instead of a paragraph.
func pairs(b *strings.Builder, rows []counterRow, indent string) {
	if len(rows) == 0 {
		return
	}
	dim, reset := theme.Fg(theme.Dim), theme.Reset
	// Widths from what is actually there, never hardcoded: half of these
	// labels are short and a fixed column spends the difference on nothing.
	labelW, numW, goalW := 0, 0, 0
	for _, r := range rows {
		if n := theme.Width(r.label); n > labelW {
			labelW = n
		}
		if n := len(r.count); n > numW {
			numW = n
		}
		if n := len(r.goal); n > goalW {
			goalW = n
		}
	}
	// pad is false for the last cell on a row: the goal column is the widest
	// thing that may be absent, and padding it at the end of a line leaves
	// trailing spaces that nothing can see and every diff can.
	cell := func(r counterRow, pad bool) string {
		goal := r.goal
		if pad {
			goal = fmt.Sprintf("%-*s", goalW, goal)
		}
		return dim + theme.PadRight(r.label, labelW) + reset +
			" " + theme.Fg(r.tint) + fmt.Sprintf("%*s", numW, r.count) + reset +
			dim + goal + reset
	}
	for i := 0; i < len(rows); i += 2 {
		if i+1 < len(rows) {
			b.WriteString(indent + cell(rows[i], true) + "   " +
				cell(rows[i+1], false) + "\n")
			continue
		}
		b.WriteString(indent + cell(rows[i], false) + "\n")
	}
}

// theFork is the two marks competing for level 5, and which one is winning.
//
// One line each, because the fork is a race and a race needs both runners: the
// leader on its own reads as a decision already made, and it is not - the
// counters keep moving for the whole of level 4.
func theFork(b *strings.Builder, s *pet.State, form string) {
	sibs := pet.Siblings(s, form)
	if len(sibs) < 2 {
		return
	}
	dim, reset, emph := theme.Fg(theme.Dim), theme.Reset, theme.Fg(theme.Emph)

	// The ripest first, which is the order the tree itself would pick.
	sort.SliceStable(sibs, func(i, j int) bool {
		return share(sibs[i]) > share(sibs[j])
	})
	nameW, counterW := 0, 0
	for _, sib := range sibs {
		if n := theme.Width(pet.Name(sib.Form)); n > nameW {
			nameW = n
		}
		if n := theme.Width(pet.CounterName(sib.Counter)); n > counterW {
			counterW = n
		}
	}
	b.WriteString("\n  " + dim + i18n.S().Level5Mark + reset + "\n")
	for _, sib := range sibs {
		tick, tint := " ", theme.Dim
		if sib.Reached() {
			// The habit is paid for; only the XP is left. Worth its own mark:
			// it is the difference between "keep going" and "just wait".
			tick, tint = "✓", theme.Ident
		}
		b.WriteString("    " + theme.Fg(tint) + tick + reset + " " +
			theme.Fg(theme.Number) + theme.PadRight(pet.Name(sib.Form), nameW) + reset +
			"   " + dim + theme.PadRight(pet.CounterName(sib.Counter), counterW) + reset +
			"  " + emph + strconv.Itoa(sib.Done) + reset +
			dim + "/" + strconv.Itoa(sib.Threshold) + reset + "\n")
	}
}

func share(s pet.Sibling) float64 {
	if s.Threshold <= 0 {
		return 0
	}
	return float64(s.Done) / float64(s.Threshold)
}

// habitGroups splits the counters into the two questions they answer: what you
// DO, and how your sessions LOOK. Anything not listed falls into habits, so a
// counter added later shows up rather than disappearing.
var sessionCounters = map[string]bool{
	"ctx_low": true, "short_sessions": true, "long_sessions": true,
	"sessions_under_40": true, "sessions_15min": true, "sessions_4h": true,
	"same_repo_days": true, "bypass_turns": true, "ctx100_sessions": true,
	"ctx_maxed": true, "impulsive": true,
}

// counters prints everything the pet has counted, grouped and sorted.
func counters(b *strings.Builder, s *pet.State, form string) {
	var habits, sessions []counterRow
	for _, name := range sortedKeys(s.Counters) {
		v := s.Counters[name]
		if v == 0 {
			continue // a counter at zero is a habit that has not started
		}
		goal := pet.GoalOf(s, form, name)
		row := counterRow{
			label: pet.CounterName(name),
			count: strconv.Itoa(v),
			tint:  tintFor(goal),
		}
		if goal.Leads() {
			row.goal = "/" + strconv.Itoa(goal.Threshold)
		}
		if sessionCounters[name] {
			sessions = append(sessions, row)
		} else {
			habits = append(habits, row)
		}
	}
	dim, reset := theme.Fg(theme.Dim), theme.Reset
	// The key, once, so the colour does not have to be guessed. Cheap enough
	// at one line, and without it the three tints are decoration.
	t := i18n.S()
	legend := "  " + dim + "· " + reset +
		theme.Fg(theme.Ident) + t.Met + reset + dim + "  ·  " + reset +
		theme.Fg(theme.Number) + t.OnTheWay + reset + dim + "  ·  " + reset +
		theme.Fg(theme.Emph) + t.LeadsNowhere + reset
	if len(habits) > 0 {
		b.WriteString("\n  " + dim + t.Habits + reset + legend + "\n")
		pairs(b, habits, "    ")
	}
	if len(sessions) > 0 {
		b.WriteString("\n  " + dim + t.Sessions + reset + "\n")
		pairs(b, sessions, "    ")
	}
}

// sortedKeys orders by value descending, then by name, so the list is stable
// between refreshes and the biggest number is at the top.
func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if m[out[i]] != m[out[j]] {
			return m[out[i]] > m[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// larder is what the pet can eat and what it has to wait for.
//
// Two of the five meals have a cooldown, and a refused meal used to say only
// "ya ha comido" at the moment you tried. Here it is visible before you try.
func larder(b *strings.Builder, s *pet.State, now time.Time) {
	dim, reset, emph := theme.Fg(theme.Dim), theme.Reset, theme.Fg(theme.Emph)
	names := make([]string, 0, len(pet.Foods))
	for name := range pet.Foods {
		// The overflow is not something you can decide to eat: it is what
		// happens TO the pet when the context blows. Listing it as "listo"
		// beside four meals reads like a menu item, and it is a penalty.
		if pet.Foods[name].XP < 0 {
			continue
		}
		names = append(names, name)
	}
	// By XP, richest first: what it is worth is the reason to reach for it.
	sort.Slice(names, func(i, j int) bool {
		if pet.Foods[names[i]].XP != pet.Foods[names[j]].XP {
			return pet.Foods[names[i]].XP > pet.Foods[names[j]].XP
		}
		return names[i] < names[j]
	})
	// Measured over EVERY row the section prints, the overflow included: it is
	// the longest label of the lot, and leaving it out of the count is what
	// pushed its own line one column out of true.
	labelW := 0
	for name, food := range pet.Foods {
		_ = name
		if n := theme.Width(food.Label()); n > labelW {
			labelW = n
		}
	}
	t := i18n.S()
	b.WriteString("\n  " + dim + t.Food + reset + "\n")
	for _, name := range names {
		food := pet.Foods[name]
		tint := theme.Ident
		if food.XP <= 0 {
			tint = theme.Bad
		}
		when := theme.Fg(theme.Ident) + t.Ready + reset
		if left := pet.Waiting(s, name, now); left > 0 {
			when = dim + t.In + " " + roughly(left) + reset
		}
		b.WriteString("    " + dim + theme.PadRight(food.Label(), labelW) + reset +
			"  " + theme.Fg(tint) + fmt.Sprintf("%+3d", food.XP) + reset +
			dim + " xp" + reset + "   " + when + "\n")
	}
	// And the one that is not a meal, said as what it is.
	if overflow, ok := pet.Foods["overflow"]; ok {
		b.WriteString("    " + dim + theme.PadRight(overflow.Label(), labelW) + reset +
			"  " + theme.Fg(theme.Bad) + fmt.Sprintf("%+3d", overflow.XP) + reset +
			dim + " xp" + reset + "   " + dim + t.IfContextBlows + reset + "\n")
	}
	_ = emph
}

// writeTo is here so the sections can be tested without a whole panel.
func writeTo(out io.Writer, b *strings.Builder) { io.WriteString(out, b.String()) }
