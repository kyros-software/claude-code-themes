package statusline

import (
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The card on the right: two layers that never mix.
//
//	vitality  of the moment, from context and quota usage: eyes, feet, colour.
//	progress  permanent, from the XP in ~/.claude/pet.json: the silhouette.
//
// It also books the things that have to be charged to the pet and that no hook
// can see, because hooks receive neither the context usage nor the permission
// mode.

// Card is the right-hand column plus everything band 4 needs. Four rows on the
// right - the state label and the three of the compact sprite - against the
// four bands on the left.
type Card struct {
	Rows   [4]string
	Bubble string
	Facts  *session.Facts

	// Form and Mark are what a person READS - the canvas's Spanish, not the
	// English ids pet.json stores. Vital comes along for what is not text.
	Form  string
	Level int
	Vital pet.Vital

	// id is Form as pet.json spells it. Form is the canvas's Spanish, which is
	// for reading; the tree, the ramps and the rung all key off the English id,
	// so RememberShape needs this one and not that one.
	id string

	// Body is the exact colour the creature's torso is drawn in right now: its
	// branch's ramp, at the step the state picks. Band 1 borrows it so the
	// model pill and the context bar are painted the pet's own colour instead
	// of a second, parallel reading of the same session. See engine().
	Body theme.Colour

	// The bar in band 4: Done out of Span. What the two mean depends on where
	// the pet is - the stretch of XP up to the next level while there is one,
	// the habit that opens the next mark once there is not - and Mark says
	// which of the two it is: empty for XP, the mark's name for a habit.
	Done, Span int
	Mark       string

	// Worn is the MARK the pet is wearing, in the canvas's Spanish, and Form
	// is then the trade it is a variant of. Band 4 prints them together:
	// `cazabugs[sabueso]` is a pet that IS a sabueso, and a sabueso is one of
	// the two shapes a cazabugs can take.
	//
	// The tree forks at levels 2, 3 and 5; the mark is the level 5 fork. So
	// the bracket appears at level 5 and nowhere else:
	//
	//	nivel 4   cazabugs            no mark yet
	//	nivel 5   cazabugs[sabueso]   the fork, and which way it went
	//	nivel 6   lobo                the title needs no context
	//
	// It used to hold the OPPOSITE - the mark being headed for rather than the
	// one worn - which put `cazabugs[sabueso]` on a level 4 pet that was not a
	// sabueso and might never be. Two bright words with an all-but-invisible
	// bracket between them read as one compound name, and the tense the
	// bracket was carrying reached nobody.
	Worn string

	// LevelledUp says the bubble, if there is one, is the level announcement,
	// so Spoke knows to move LevelSeen along with it.
	LevelledUp bool

	// State is the vitality's name - "a gusto", "ahogada". It used to crown the
	// card; it sits in band 4 beside the level now, and the row it vacated went
	// back to the creature's crest. Jumped is true for the one refresh after it
	// changes, which is what makes it bold.
	State  string
	Jumped bool
}

// book applies the side effects on pet.json. A failure here must never take the
// statusline down with it, so every step is best-effort.
func book(p *Payload, previousLabel, label string, newTurn bool, statePath string, now time.Time) {
	blown := label == "k.o." && previousLabel != "" && previousLabel != "k.o."
	bypass := newTurn && p.Permissions == "bypass"
	if !blown && !bypass {
		return
	}
	pet.Update(statePath, func(s *pet.State) bool {
		write := false
		if blown {
			// ctx100_sessions is NOT touched here: `pet session` counts it on
			// close, which is once per session.
			write = pet.Feed(s, "overflow", "", now)
		}
		if bypass {
			s.Bump("bypass_turns", 1)
			write = true
		}
		return write
	})
}

// eventFor is the "primero el evento" half of the canvas's rule: what, if
// anything, is worth opening the pet's mouth about, in the order it matters.
func eventFor(s *pet.State, label string, levelledUp, bigMeal bool) pet.Event {
	switch {
	case levelledUp:
		return pet.EventLevelUp
	case label == "k.o.":
		return pet.EventCtxBlown
	case s.Hunger >= pet.HungerWarn:
		return pet.EventHungry
	case bigMeal:
		return pet.EventBigMeal
	case s.Streak >= 3:
		return pet.EventStreak
	}
	return pet.EventNothing
}

// RenderCard builds the right-hand column and whatever the pet has to say.
func RenderCard(p *Payload, facts session.Facts, rate rateFacts, newTurn, bubbleAllowed bool,
	statePath string, now time.Time) (Card, *pet.State) {
	usage := pet.ContextLoad(p.ContextPc)
	vital := pet.StateFor(usage)
	label := vital.Label

	s := pet.Load(statePath)
	pet.DecayHunger(s, now)
	form, level := pet.CurrentForm(s)
	// A level-up is spotted against the last level announced, which lives in
	// pet.json itself.
	levelledUp := level > s.LevelSeen

	rows := pet.DrawCard(form, vital, int(now.Unix()), s.Hunger >= pet.HungerWarn)

	previousLabel := facts.Label
	// Crossing a threshold makes the label bold for one refresh.
	jumped := previousLabel != "" && previousLabel != label

	// One peak, because there is one number: the state and the context are the
	// same reading now. There were two while the state read the tightest of
	// the three necks - the neck's peak for the state, the context's own for
	// the counters named after it - and the second outlived the first.
	ctxPeak := facts.CtxPeak
	if usage > ctxPeak {
		ctxPeak = usage
	}

	changed := previousLabel != label ||
		ctxPeak >= facts.CtxPeak+1.0 ||
		newTurn ||
		(p.APIMs != nil && (facts.APIMs == nil || *p.APIMs != *facts.APIMs))

	var out Card
	if changed {
		book(p, previousLabel, label, newTurn, statePath, now)
		t0 := facts.T0
		if t0 <= 1e9 {
			t0 = now.Unix()
		}
		promptID := p.PromptID
		if promptID == "" {
			promptID = facts.PromptID
		}
		out.Facts = &session.Facts{
			Label:    label,
			CtxPeak:  roundTo1(ctxPeak),
			T0:       t0,
			Repo:     p.Repo,
			PromptID: promptID,
			APIMs:    rate.apiMs,
			TPS:      rate.storedTPS(),
			TPSAt:    rate.tpsAt,
		}
	}

	// All four rows are the creature now, crest included. The state's name went
	// to band 4; see Card.State.
	out.Rows = rows
	out.State = pet.Name(label)
	if vital.Sparkle {
		out.State += " ✦"
	}
	out.Jumped = jumped

	out.Form, out.Level, out.Vital = pet.Name(form), level, vital
	out.id = form
	// The creature's own torso colour, taken the same way render.go takes it:
	// the branch owns the hue, the state picks the step.
	out.Body = pet.RampOf(form).Body[vital.Rank]
	// While there is a level above, the bar is XP. At the top it swaps to the
	// habit that opens the next mark, which is the only progress left; a pet
	// already wearing its mark has neither, and band 4 leans on the state.
	// The bracket is what the pet IS, not where it is going: a mark names
	// itself inside it and its trade outside. See Card.Worn.
	if parent, ok := pet.MarkParent(form); ok {
		out.Form, out.Worn = pet.Name(parent), pet.Name(form)
	}
	// The bar, which is a different question: XP while there is a level left
	// to open, and the habit that opens the next shape once there is not.
	mark, heading := pet.NextMark(s, form)
	if done, span, ok := pet.LevelProgress(s.XP); ok {
		out.Done, out.Span = done, span
	} else if heading {
		out.Done, out.Span, out.Mark = mark.Done, mark.Threshold, pet.Name(mark.Form)
	}

	if bubbleAllowed {
		// A big meal is one the hook just fed it; the statusline sees it as a
		// log entry still inside the window.
		bigMeal := false
		if n := len(s.Log); n > 0 {
			last := s.Log[n-1]
			since := now.Sub(time.Unix(last.At, 0))
			bigMeal = pet.BigMeal(last.Event) && since >= 0 && since < pet.BigMealWindow
		}
		event := eventFor(s, label, levelledUp, bigMeal)
		// Nothing is booked here. The line has still to get past the band's
		// layout, and band 4 drops the bubble before anything else when it
		// runs out of room - booking it here spent the five-minute cooldown,
		// and burned the phrase, on a line nobody saw. The caller calls Spoke
		// once it is actually on screen.
		out.Bubble = pet.Speak(s, event, form, now, nil)
		out.LevelledUp = levelledUp
	}
	return out, s
}

// RememberShape books the rung the creature was actually drawn on, so a habit
// that falls back later cannot take the shape back down the tree. See pet.Tier.
//
// It decides on the copy the render already read and only touches the file when
// the shape has actually moved - which over a whole pet's life is a handful of
// times. The statusline refreshes about once a second, so a Load here on every
// refresh would be a read a second to answer a question the caller can already
// answer for free, and a Save would be a write a second to restate a field that
// did not change.
//
// When it does write it re-reads first, for the same reason Spoke does:
// RenderCard persists nothing, and a hook may have fed the pet since it read.
// Saving the render's own copy would roll that meal back.
func RememberShape(c Card, s *pet.State, statePath string) {
	if s == nil || c.id == "" || c.id == s.FormSeen ||
		pet.Tier(c.id) < pet.Tier(s.FormSeen) {
		return
	}
	// pet.Save is what actually writes the rung - it derives it from the state
	// it is handed, so that no path can persist a pet and forget. All this has
	// to decide is WHEN a save is worth doing, which is why it does not set the
	// field itself: two writers for one field is how they end up disagreeing.
	//
	// Update re-reads under the lock, which is the same re-read this always
	// did, now with nothing able to slip in between the read and the write.
	pet.Update(statePath, func(*pet.State) bool { return true })
}

// Spoke books a bubble that reached the screen: the phrase goes into the
// do-not-repeat list, the cooldown starts, and a level announcement moves
// LevelSeen so it is not made twice.
//
// It re-reads the file rather than saving the copy the render worked from: a
// hook may have fed the pet in between, and this must not roll that back.
func Spoke(c Card, s *pet.State, statePath string, now time.Time) {
	if c.Bubble == "" {
		return
	}
	pet.Remember(s, c.Bubble, now)
	pet.Update(statePath, func(fresh *pet.State) bool {
		fresh.Said, fresh.SaidAt = s.Said, s.SaidAt
		if c.LevelledUp {
			fresh.LevelSeen = c.Level
		}
		return true
	})
}

func roundTo1(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}
