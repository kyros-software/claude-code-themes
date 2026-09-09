package pet

// The progress layer. Accumulated XP sets the level, and the SHAPE never goes
// back down the tree - see floor. The level itself does: XP falls when the
// context blows (-15) and while the pet goes hungry, so a level can be lost,
// which TestStarvingCanCostALevelButNeverKills has always said and this
// comment used to deny. TestTheLevelNeverGoesDown is about LevelFor being
// monotonic in XP, which is a different claim and the only one that holds.
//
// At every fork the branch is not chosen by the user but by whichever
// behaviour counter is highest at that moment, so the shape you end up with is
// a readout of how you work.

import "sort"

// Root is the newborn form.
const Root = "spark"

// Tree is the evolution tree. Order inside a slice is the tie-break.
var Tree = map[string][]string{
	"spark":     {"pattern", "probe", "ember"},
	"pattern":   {"refactor", "tidy"},
	"probe":     {"bughunter", "architect"},
	"ember":     {"sprinter", "marathon", "feral"},
	"refactor":  {"surgeon", "weaver"},
	"tidy":      {"monk", "gardener"},
	"bughunter": {"bloodhound", "exterminator"},
	"architect": {"cartographer", "oracle"},
	"sprinter":  {"bolt", "sniper"},
	"marathon":  {"ox", "mole"},
	"feral":     {"gremlin", "kraken"},

	// Level 6, the titles: the final shape of each branch, one per mark. The
	// canvas calls the tier "nivel 5 · títulos - la forma final de cada rama"
	// and numbers the marks one lower than we do; the shape of the tree is the
	// same either way, and renumbering would move every XP threshold.
	"surgeon":      {"scalpel"},
	"weaver":       {"loom"},
	"monk":         {"abbot"},
	"gardener":     {"forest"},
	"bloodhound":   {"wolf"},
	"exterminator": {"wasp"},
	"cartographer": {"atlas"},
	"oracle":       {"sphinx"},
	"bolt":         {"storm"},
	"sniper":       {"falcon"},
	"ox":           {"mammoth"},
	"mole":         {"worm"},
	"gremlin":      {"devil"},
	"kraken":       {"leviathan"},
}

// Parent is Tree inverted.
var Parent = func() map[string]string {
	out := map[string]string{}
	for parent, kids := range Tree {
		for _, kid := range kids {
			out[kid] = parent
		}
	}
	return out
}()

// Level is an XP threshold and the level it opens.
type Level struct {
	XP    int
	Level int
}

// Levels: accumulated XP -> level.
//
// Six. The sixth is the titles, and the last two gaps are deliberately much
// wider than the curve that leads up to them - 60, 120, 220, then 1600 and
// 2500.
//
// That break is the point. Level 4 is the one rung of the tree where NOTHING
// forks: the temperament is chosen at 2, the trade at 3, and the mark waits
// for 5. So the whole of level 4 is the stretch where the habit that decides
// the mark is still moving and can still change its mind - a `bughunter` who
// starts reproducing bugs before fixing them leans `bloodhound`, one who
// strings green suites together leans `exterminator`, and either can overtake
// the other while the level lasts.
//
// It used to last 500 XP. Measured against a real day's feeding - about 12 XP
// a meal, some sixty an hour of actual work - that is eight hours: one long
// session, and the fork was decided before the habits had a week to say
// anything. 1600 puts it around twenty-five hours of work, which is what the
// stretch is FOR now that band 4 shows where the pet is heading while it
// climbs. See Card.Toward.
//
// The cost, with eyes open: XP falls - `overflow` takes 15, hunger takes more
// - so a level lost is now a much longer climb back. That is the same trade
// the wider gap buys, seen from the other side.
var Levels = []Level{{0, 1}, {60, 2}, {180, 3}, {400, 4}, {2000, 5}, {4500, 6}}

// BranchBy names the counter that decides each fork. Highest wins on level-up.
var BranchBy = map[string]string{
	"pattern": "methodical", "probe": "inquisitive", "ember": "impulsive",
	"refactor": "diffs", "tidy": "ctx_low",
	"bughunter": "tests", "architect": "plans",
	"sprinter": "short_sessions", "marathon": "long_sessions", "feral": "ctx_maxed",
}

// BranchScale is one branch's worth of each fork counter: what a person
// leaning hard into that habit reaches in a day of normal use.
//
// topBranch used to compare these counters RAW, and they are not the same kind
// of number. Four of them bump once per EVENT and never stop - a commit, a
// compact, a closed plan task - and five bump at most once per SESSION, at the
// close. A day holds a dozen commits and three or four sessions, so a raw race
// between `methodical` and `impulsive` was settled by the units before the
// habit got a word in: this machine's own pet.json read 39 against 2 after
// three weeks of work, and the 2 is the ceiling, not the effort.
//
// It is the bug ripestMark was fixed for at level 5 - "the first sibling past
// its line keeps the branch" - one rung up and worse, because up here there is
// no threshold to cross and so nothing to normalise against. Dividing by the
// scale puts every counter in the same unit, DAYS of that habit, and the fork
// goes back to being a race between ways of working.
//
// Where the numbers come from. The design budgets a normal day at 128 XP - the
// same budget behind the hourly cooldown on a green suite, "ocho suites verdes
// en una jornada" - so a day spent leaning on one meal is 128 divided by what
// that meal pays: ten commits at 12, eight suites at 15, twenty-one plan tasks
// at 6. The per-session counters cannot outrun the number of sessions in a day,
// which is three or four, measured over three weeks on this machine. The
// duration pair splits that between them, because a day does not hold three
// sessions of four hours.
//
// They are a calibration, not a law, and the test that defends them is
// TestTheEmberBranchSurvivesANormalDayOfWork in internal/hook: it plays a
// person who works at the limit AND commits, which is the case the raw race
// could not see.
var BranchScale = map[string]int{
	// Per event, off the 128 XP day.
	"methodical":  10, // commits at 12 xp, plus the compacts
	"inquisitive": 8,  // green suites at 15, where the hourly cooldown lands too
	"diffs":       10, // commits alone
	"tests":       8,  // the same cooldown
	"plans":       21, // closed plan tasks at 6 xp

	// Per session, capped by how many sessions fit in a day.
	"impulsive":      3, // one per session over 85%
	"ctx_low":        3, // one per session under 60%
	"ctx_maxed":      3, // one per session over 95%
	"short_sessions": 3, // one per session under 15 min
	"long_sessions":  2, // one per session over 90 min - fewer of those fit
}

// branchShare is how far a fork counter has come in its own unit: days of that
// habit.
//
// A counter with no scale divides by one, which is the raw number and the old
// behaviour. That is deliberate: a fork added without a scale keeps working and
// TestEveryForkCounterHasAScale says so out loud, rather than the shape quietly
// going wrong because of a zero nobody wrote.
func branchShare(s *State, form string) float64 {
	counter := BranchBy[form]
	if counter == "" {
		return 0
	}
	done := s.Counters[counter]
	if done < 0 {
		done = 0
	}
	return float64(done) / float64(scaleOf(counter))
}

// scaleOf is BranchScale with the missing entry read as 1, which is the raw
// count and the behaviour before there were scales at all.
func scaleOf(counter string) int {
	if scale := BranchScale[counter]; scale > 0 {
		return scale
	}
	return 1
}

// Unlock is the habit a level-5 mark asks for. Not XP: a habit.
type Unlock struct {
	Counter   string
	Threshold int
}

// Unlocks for the fourteen marks.
var Unlocks = map[string]Unlock{
	"surgeon":      {"diff_streak", 20},
	"weaver":       {"widest_commit", 10},
	"monk":         {"sessions_under_40", 5},
	"gardener":     {"docs_days", 2},
	"bloodhound":   {"repro_before_fix", 10},
	"exterminator": {"test_streak", 15},
	"cartographer": {"longest_plan", 10},
	"oracle":       {"plans_before_code", 5},
	"bolt":         {"sessions_15min", 10},
	"sniper":       {"single_tool_tasks", 8},
	"ox":           {"sessions_4h", 3},
	"mole":         {"same_repo_days", 5},
	"gremlin":      {"bypass_turns", 30},
	"kraken":       {"ctx100_sessions", 3},
}

// Titles are the level-6 forms, one per mark: "la forma final de cada rama".
// Each asks for MORE of the same habit its mark asked for - a bloodhound that
// reproduced ten bugs before fixing them becomes a wolf at thirty.
var Titles = map[string]string{
	"surgeon": "scalpel", "weaver": "loom", "monk": "abbot", "gardener": "forest",
	"bloodhound": "wolf", "exterminator": "wasp", "cartographer": "atlas",
	"oracle": "sphinx", "bolt": "storm", "sniper": "falcon", "ox": "mammoth",
	"mole": "worm", "gremlin": "devil", "kraken": "leviathan",
}

// TitleAsks is what each title asks of its mark's habit, straight off the
// canvas "Cómo llegar a cada forma".
//
// These used to be one number - a title asked TWICE its mark - and that number
// was invented here, not designed: the atlas carried the fourteen titles but no
// condition for any of them, so a uniform multiplier was the least-wrong guess
// available. The canvas has since spelled out a factor PER title, from x2.5 to
// x5, which is strictly better information than any single multiplier: the
// designer weighed each habit on its own.
//
// Two entries do not follow the canvas, and both for reasons the canvas could
// not have known:
//
//   - atlas. The canvas asks for "5 planes de 10 tareas cerrados", which is a
//     COUNT of big plans. longest_plan is a RecordMax - the largest single plan
//     ever closed - so writing 50 here would ask for one plan of fifty tasks,
//     which is not what was designed and is a different kind of quantity. The
//     doubling stands until there is a counter that can express what the canvas
//     means.
//   - devil. The canvas says 100. bypass_turns moves about thirty times a day -
//     measured, and this machine's own pet.json was already past 115 the day
//     the canvas was read - so 100 is a title that arrives already earned, which
//     is not a title. 200 puts it at about a week, in line with the rest.
//
// Everything else is the canvas's number.
var TitleAsks = map[string]int{
	"scalpel":   50,  // diff_streak, x2.5
	"loom":      25,  // widest_commit, x2.5
	"abbot":     15,  // sessions_under_40, x3
	"forest":    7,   // docs_days, x3.5 - "una semana entera ordenando"
	"wolf":      30,  // repro_before_fix, x3
	"wasp":      50,  // test_streak, x3.3
	"atlas":     20,  // longest_plan - see above, NOT the canvas's 50
	"sphinx":    20,  // plans_before_code, x4
	"storm":     30,  // sessions_15min, x3
	"falcon":    25,  // single_tool_tasks, x3
	"mammoth":   10,  // sessions_4h, x3.3
	"worm":      20,  // same_repo_days, x4
	"devil":     200, // bypass_turns - see above, NOT the canvas's 100
	"leviathan": 10,  // ctx100_sessions, x3.3
}

// TitleUnlock is what the title behind a mark asks for: its mark's counter, at
// the title's own threshold.
func TitleUnlock(mark string) (Unlock, bool) {
	base, ok := Unlocks[mark]
	if !ok {
		return Unlock{}, false
	}
	asks, ok := TitleAsks[Titles[mark]]
	if !ok {
		return Unlock{}, false
	}
	return Unlock{base.Counter, asks}, true
}

// Secrets are the two forms that do not come off the tree.
var Secrets = [2]string{"phoenix", "chimera"}

// Temperaments are the three level-2 counters, which the chimera compares.
var Temperaments = [3]string{"methodical", "inquisitive", "impulsive"}

// LevelFor maps accumulated XP to a level.
func LevelFor(xp int) int {
	level := 1
	for _, l := range Levels {
		if xp >= l.XP {
			level = l.Level
		}
	}
	return level
}

// NextThreshold is the XP at which the next level starts, or false at the top.
func NextThreshold(xp int) (int, bool) {
	for _, l := range Levels {
		if l.XP > xp {
			return l.XP, true
		}
	}
	return 0, false
}

// LevelProgress is how far INTO the current level the XP has come: how much of
// this level's stretch is done, and how long the stretch is.
//
// It measures the stretch and not the running total on purpose. Against the
// total, a bar is never empty the morning after a level-up - level 2 opens at
// a third full, levels 3 and 4 at just under a half - which reads as progress
// nobody made. Against the stretch it opens at zero and closes full.
//
// At the top there is no stretch left, and ok is false.
func LevelProgress(xp int) (done, span int, ok bool) {
	// pet.json is user-writable and a hand-edited negative reads as a newborn,
	// not as a bar running backwards.
	if xp < 0 {
		xp = 0
	}
	next, has := NextThreshold(xp)
	if !has {
		return 0, 0, false
	}
	base := 0
	for _, l := range Levels {
		if l.XP <= xp {
			base = l.XP
		}
	}
	return xp - base, next - base, true
}

// Mark is a level-5 mark within reach: the form it opens, the habit it asks
// for, and how far that habit has come.
type Mark struct {
	Form      string
	Counter   string
	Done      int
	Threshold int
}

// Share is how much of the habit is done, from 0 to 1.
func (m Mark) Share() float64 {
	if m.Threshold <= 0 {
		return 0
	}
	return float64(m.Done) / float64(m.Threshold)
}

// NextMark is the mark the pet is closest to wearing.
//
// This is what progress becomes once the XP runs out. The canvas: "las
// ramificaciones no dependen de la XP sino del hábito". A pet at the top of
// the ladder still has somewhere to go, and the habit is the only thing left
// that still moves - so it is the habit the bar measures.
//
// Closest is the largest share of the habit done, and a tie falls back to the
// order the design lists the siblings in, which is the tie-break CurrentForm
// already uses. A pet that wears a mark, or a secret, has nothing left to
// reach and gets false.
//
// form is the pet's current form: every caller has just worked it out with
// CurrentForm, and asking for it keeps this from walking the tree a second
// time on every refresh - and from disagreeing with the form on screen.
func NextMark(s *State, form string) (Mark, bool) {
	// A secret hides the branch but no longer hides the title, so there is
	// something left to point at. Ask the tree where the pet is really
	// standing: on a mark it has earned, and the bar measures that mark's
	// title - or on a bare trade, and the bar measures the nearest mark, which
	// is the gate on the way to its title either way.
	if secretForm(form) {
		form, _ = treeWalk(s)
	}

	// A pet wearing a mark is not finished any more: there is a title behind
	// it, asking for the same habit three times over. Only a title has nothing
	// left to reach.
	if _, worn := Unlocks[form]; worn {
		u, ok := TitleUnlock(form)
		if !ok {
			return Mark{}, false
		}
		done := s.Counters[u.Counter]
		if done < 0 {
			done = 0
		}
		if done > u.Threshold {
			done = u.Threshold
		}
		return Mark{Titles[form], u.Counter, done, u.Threshold}, true
	}
	if _, isTitle := titleForms[form]; isTitle {
		return Mark{}, false
	}

	// The same choice CurrentForm makes, so the mark the bar names is the mark
	// the walk would hand over. Unearned siblings count here - that is the
	// point of the bar - but the ranking is the identical ripeness, which is
	// why the two can no longer disagree.
	kid, ok := ripestMark(s, Tree[form], false)
	if !ok {
		return Mark{}, false
	}
	u := Unlocks[kid]
	done := s.Counters[u.Counter]
	if done < 0 {
		done = 0
	}
	// Clamped for the DISPLAY only: a bar does not overflow. The choice above
	// deliberately saw the uncapped number.
	if done > u.Threshold {
		done = u.Threshold
	}
	return Mark{Form: kid, Counter: u.Counter, Done: done, Threshold: u.Threshold}, true
}

// ripestMark picks between siblings that are opened by a habit rather than by
// a counter race: the one whose habit has come FURTHEST relative to what it
// asks. A tie falls back to the order the design lists them in, same as
// topBranch. With met=true only siblings already earned can win, which is what
// CurrentForm needs; with met=false the nearest one wins even unearned, which
// is what the bar in band 4 points at.
//
// It used to be "the first sibling in the list whose threshold is met", and
// that quietly amputated half the tree. Every counter here only ever goes UP,
// so the first sibling to cross its line kept the branch for good: once you
// had five sessions under 40% context you were a monk, and `gardener` - and
// `forest` behind it - stopped existing for that pet, no matter how many days
// of docs you wrote afterwards. Four of the seven pairs were already dead on
// this machine's own pet.json, which is eight of the forty-one forms gone.
//
// Comparing the RATIO instead keeps every mark reachable forever: whichever
// habit you push hardest, measured against what that habit asks, is the one
// you wear. It also makes this agree with NextMark, which was already choosing
// by share - so the bar can no longer promise a mark the walk would refuse.
func ripestMark(s *State, kids []string, met bool) (string, bool) {
	best, bestRipeness, found := "", 0.0, false
	for _, kid := range kids {
		u, ok := Unlocks[kid]
		if !ok || u.Threshold <= 0 {
			continue
		}
		done := s.Counters[u.Counter]
		if done < 0 {
			done = 0
		}
		if met && done < u.Threshold {
			continue
		}
		// Uncapped on purpose: capping at 1 would tie every earned sibling and
		// hand the branch back to list order, which is the bug this replaces.
		if ripeness := float64(done) / float64(u.Threshold); !found || ripeness > bestRipeness {
			best, bestRipeness, found = kid, ripeness, true
		}
	}
	return best, found
}

// Tier is the rung a FORM sits on: 1 for the root, 2 for a temperament, 3 for
// a trade, 5 for a mark or a secret, 6 for a title.
//
// It is not the pet's level. A trade is worn at level 3 and still worn at 4,
// and the pet's level is XP while the tier is shape. An unknown name - a
// hand-edited pet.json, a form renamed in some future atlas - is 0, which is
// below everything and therefore holds nothing back.
func Tier(form string) int {
	if form == "" {
		return 0
	}
	if form == Root {
		return 1
	}
	if titleForms[form] {
		return 6
	}
	if _, mark := Unlocks[form]; mark {
		return 5
	}
	for _, secret := range Secrets {
		if form == secret {
			return 5
		}
	}
	depth := 0
	for f, ok := Parent[form]; ok; f, ok = Parent[f] {
		depth++
	}
	if depth == 0 {
		return 0
	}
	return depth + 1
}

// floor holds the shape at the highest rung it has ever stood on.
//
// The walk is recomputed from the counters on every refresh and two of the
// habits - the clean test streak and the clean diff streak - go back to zero
// when the context blows. Without a floor that turned into a fall down the
// TREE: a level 6 `wasp` came back as a `bughunter`, a level 3 shape, still
// labelled level 6 because the level is XP and XP does not fall.
//
// The rule is that a form only ever moves sideways or up. An `exterminator`
// who loses the streak can become a `bloodhound` - the same rung, the other
// habit, which is a real change and says something true about the week - but
// it cannot go back to being a plain `bughunter`. What was earned at a rung
// stays at that rung.
//
// seen is State.FormSeen, and it is only ever written by RememberForm, which
// is the same split speech.go uses: work it out here, write it down there,
// when it has actually reached the screen.
//
// One thing the rung rule caught that it was never aimed at: a TITLE could not
// move. A pet wearing `abbot` whose temperament flips still has `bloodhound`
// earned and waiting on the other branch, but rung 6 beats rung 5, so it stayed
// an abbot until the new branch grew a title of its own - which is a different
// habit, three times over, and can be weeks away. The shape stopped saying
// anything about the month the pet was actually having.
//
// So the floor now asks WHY the walk came out lower, which is the question it
// meant all along. Two answers:
//
//   - The habit fell. `wasp` loses the test streak and the walk offers
//     `bloodhound`, the same trade's other mark. That is the fall the floor
//     exists to stop, and it still stops it.
//   - The branch changed. `abbot` in `tidy` becomes `bloodhound` in
//     `bughunter`. Nothing was lost: the pet is somewhere else, and the mark
//     over there is earned. That one goes through.
//
// The trade tells them apart. Same trade means a fall; a different trade means
// the pet moved, and a move down one rung to a mark it has genuinely earned is
// a truer picture than a title from a branch it left. Below rung 5 nothing
// passes either way: a walk that comes back a bare trade is XP falling or a
// branch with nothing earned on it yet, and neither is a move.
func floor(here, seen string) string {
	if Tier(seen) <= Tier(here) {
		return here
	}
	if Tier(here) >= 5 {
		if from, to := tradeOf(seen), tradeOf(here); from != "" && to != "" && from != to {
			return here
		}
	}
	return seen
}

// tradeOf is the rung-3 trade a form hangs off. The root, the temperaments and
// the two secrets hang off none and get "", which reads as "cannot tell" and
// makes floor keep its old answer rather than guess.
func tradeOf(form string) string {
	for f := form; f != ""; {
		if Tier(f) == 3 {
			return f
		}
		parent, ok := Parent[f]
		if !ok {
			return ""
		}
		f = parent
	}
	return ""
}

// RememberForm records the rung a form has reached, so a later fall in the
// counters cannot take the shape back down the tree with it. Callers that
// persist the state call it; callers that only ask what the pet looks like -
// the panel's what-if ghost, for one - must not.
func RememberForm(s *State, form string) {
	if Tier(form) >= Tier(s.FormSeen) {
		s.FormSeen = form
	}
}

// BranchMargin is what it costs to TAKE a fork off the branch already standing
// there: one full day of the habit, in the unit BranchScale is written in.
//
// Without it the fork went to whoever was ahead by any amount at all, and two
// habits that run level do not stay ahead of each other for long. Measured:
// methodical 4.60 days against inquisitive 4.38, which is a gap of 0.22 - two
// green suites one way, three commits the other. The pet changed name and
// sprite several times in an afternoon, on a fork that had not really been
// decided at any point.
//
// A day is the unit the scales are already denominated in, so the rule says
// itself: to take somebody's branch you have to out-work them by a day of the
// habit, not by a commit.
const BranchMargin = 1.0

// topBranch picks between siblings: the habit that has come furthest IN ITS
// OWN UNIT wins, a tie falls back to the order the design lists them in, and
// a fork already taken is DEFENDED - see BranchMargin.
//
// It compared the raw counters until it turned out that half of them count
// events and half count sessions, which is a race with a winner before it
// starts. See BranchScale.
//
// The price of the defence, paid with eyes open: the shape stops being a pure
// function of the counters. Two pets with identical numbers can wear different
// forms, because one of them got there by a road the other did not. Everything
// else in this file recomputes from the counters and says so; this is the one
// fact about a pet that is only in the file.
func topBranch(s *State, parent string) string {
	candidates := Tree[parent]
	best := candidates[0]
	bestShare := branchShare(s, best)
	for _, kid := range candidates[1:] {
		if share := branchShare(s, kid); share > bestShare {
			best, bestShare = kid, share
		}
	}

	held, defended := s.Branch[parent]
	if !defended || held == best {
		return best
	}
	// A held branch that is not a child of this fork is a hand-edited or
	// outdated file, and defends nothing. Load drops these, so reaching here
	// means the state was built in memory.
	found := false
	for _, kid := range candidates {
		if kid == held {
			found = true
		}
	}
	if !found {
		return best
	}
	if bestShare >= branchShare(s, held)+BranchMargin {
		return best
	}
	return held
}

// RememberBranch writes down which side of each fork the pet is standing on,
// so the next walk knows who is defending it.
//
// It records what topBranch ALREADY decided, hysteresis included, so calling it
// twice changes nothing: the held branch stays held until a rival earns the
// margin, and the moment one does, that is what gets written.
//
// Only the forks the pet has actually reached. A level 2 pet has not chosen a
// trade, and writing a guess would hand the level 3 fork to a defender that
// was never there - which is the fork deciding itself one level early.
//
// Like RememberForm, this is for callers that PERSIST the state. The panel's
// what-if pet must not call it: asking what the next level would look like is
// not the pet living through it.
func RememberBranch(s *State) {
	level := LevelFor(s.XP)
	if level < 2 {
		return
	}
	if s.Branch == nil {
		s.Branch = map[string]string{}
	}
	here := topBranch(s, Root)
	s.Branch[Root] = here
	if level >= 3 {
		s.Branch[here] = topBranch(s, here)
	}
}

// CurrentForm walks the tree from the root as far as XP and habits allow.
//
// A secret is a level-5 form and waits for level 5 like every other one. Its
// CONDITION is met earlier - the chimera's is "dos temperamentos empatados al
// subir a nivel 4" - and that is the whole point of the canvas drawing a pet
// that reads "refactor · nivel 4" with "488 para quimera" underneath: the
// secret is won and the XP is what is still missing. Handing it over on the
// spot skipped level 4 whole and put a level 5 next to 412 XP.
func CurrentForm(s *State) (string, int) {
	here, level := walk(s)
	// A shape never goes back down the tree, however far a streak falls. See
	// floor.
	// The level is NOT raised to meet the rung, and that is deliberate.
	//
	// Since the floor, the two measure different things and can pull apart: the
	// shape is a high-water mark and the level is current XP, which falls -
	// `overflow` costs 15 of it and starving costs more. So a pet sitting on
	// 1900 that blows the context reads "avispa nivel 5", a title beside a
	// level that could not have earned one.
	//
	// Clamping the level up to Tier(form) would tidy that away, and it would
	// also cancel a penalty the design means: TestStarvingCanCostALevelButNever
	// Kills says out loud that going hungry costs levels. A shape you earned
	// and a level you are currently at are two different facts, and the card
	// shows both.
	return floor(here, s.FormSeen), level
}

// walk is CurrentForm without the floor: the shape the counters say RIGHT NOW,
// with no memory of where the pet has already been.
//
// The two are separate because they answer different questions and only one of
// them is about the screen. CurrentForm is what the pet looks like, and a look
// does not go backwards. walk is which branch the pet is on today, which is
// what CheckSecrets needs: it asks whether the pet is a `feral` or a
// `marathon` - two rung 3 shapes - and the floor can lift a shape above rung 3.
//
// That coupling was harmless by coincidence: every mark under those two rides a
// counter that only grows, so the floor had nothing to lift. Coincidence is not
// a reason, and the alternative was a test standing guard over a line nobody
// would think to connect. Asking the right question costs nothing.
func walk(s *State) (string, int) {
	here, level := treeWalk(s)

	// A secret is a rung-5 form, and it wins the rung it stands on: it is
	// rarer than either mark and the pet earned it doing something neither
	// mark asks for. What it must NOT do is win rungs above its own.
	//
	// It used to return here, before the walk, and that quietly ended the
	// pet's life at rung 5: no mark, no title, three quarters of its branch
	// gone the day the secret landed. A chimera is a level 5 form, not a
	// tombstone. Now the walk runs first and the secret only takes over when
	// the tree came back with rung 5 or less - so a title, which is rung 6,
	// goes on being something a chimera can grow into.
	//
	// The marks it skips past are not a loss: the secret already occupies
	// their rung, and the title behind them asks for the same habit, more of
	// it. The habit is still the gate; only the shape on the way is different.
	if level >= 5 && s.Secret != "" && Tier(here) < 6 {
		if _, ok := Sprites[string(s.Secret)]; ok {
			return string(s.Secret), level
		}
	}
	return here, level
}

// treeWalk is the shape the TREE gives, with no secret laid over it.
//
// Kept apart from walk because two callers need the branch the pet is actually
// standing on rather than the shape it wears: the walk itself, to know whether
// a title outranks the secret, and NextMark, to know which title a pet wearing
// one is climbing towards. Asking walk would get the secret back, which is the
// one answer neither of them can use.
func treeWalk(s *State) (string, int) {
	level := LevelFor(s.XP)
	here := Root
	if level >= 2 {
		here = topBranch(s, Root)
	}
	if level >= 3 {
		here = topBranch(s, here)
	}
	if level >= 5 {
		if kid, ok := ripestMark(s, Tree[here], true); ok {
			here = kid
		}
	}
	// Level 6 is the title, and it only exists for a pet that actually wears
	// the mark: the two are the same habit, the title asking three times as
	// much of it, so you cannot skip the mark on the way past.
	if level >= 6 {
		if title, ok := Titles[here]; ok {
			if u, ok := TitleUnlock(here); ok && s.Counters[u.Counter] >= u.Threshold {
				here = title
			}
		}
	}
	return here, level
}

// secretForm says a form is one of the two off the tree.
func secretForm(form string) bool {
	for _, secret := range Secrets {
		if form == secret {
			return true
		}
	}
	return false
}

// Lineage is the path walked to get here: temperament -> trade -> mark.
func Lineage(form string) []string {
	path := []string{form}
	for {
		parent, ok := Parent[path[0]]
		if !ok {
			return path
		}
		path = append([]string{parent}, path...)
	}
}

// titleForms is Titles inverted: the fourteen forms with nothing beyond them.
var titleForms = func() map[string]bool {
	out := map[string]bool{}
	for _, title := range Titles {
		out[title] = true
	}
	return out
}()

// MarkParent is the trade a mark is a variant of, and false for anything that
// is not a mark.
//
// The fourteen marks are the level 5 fork: a `bloodhound` is one of the two
// shapes a `bughunter` can take, and saying so is what band 4's bracket is for.
// A trade has no mark yet and a title has outgrown the question, so both get
// false and are printed on their own.
func MarkParent(form string) (string, bool) {
	if _, isMark := Unlocks[form]; !isMark {
		return "", false
	}
	parent, ok := Parent[form]
	return parent, ok
}

// Sibling is one of the marks a trade can turn into, and how close the pet is.
type Sibling struct {
	Form      string
	Counter   string
	Done      int
	Threshold int
}

// Reached says the habit is already paid for; only the XP is left.
func (s Sibling) Reached() bool { return s.Done >= s.Threshold }

// Siblings are the marks that compete for the level 5 fork, in the order the
// tree lists them, whichever one is currently ahead.
//
// The panel used to show only the leader, which is the one thing that cannot
// be read as a race: 33 of 10 against 1 of 15 says the fork is settled, and
// "sabueso 33/10" on its own says nothing about the alternative it beat.
//
// form is the trade being asked about. A mark or a title has no fork left and
// gets nothing.
func Siblings(s *State, form string) []Sibling {
	children, ok := Tree[form]
	if !ok {
		return nil
	}
	out := make([]Sibling, 0, len(children))
	for _, child := range children {
		u, ok := Unlocks[child]
		if !ok {
			return nil // not a level 5 fork: these children are not marks
		}
		done := s.Counters[u.Counter]
		if done < 0 {
			done = 0
		}
		out = append(out, Sibling{child, u.Counter, done, u.Threshold})
	}
	return out
}

// Ahead is every form still reachable from one, itself excluded.
//
// The tree only ever goes down, so this is a plain walk with no cycle to
// guard against - TestTheTreeHasNoCycles pins that.
func Ahead(form string) map[string]bool {
	out := map[string]bool{}
	var walk func(string)
	walk = func(f string) {
		for _, child := range Tree[f] {
			if out[child] {
				continue
			}
			out[child] = true
			walk(child)
		}
	}
	walk(form)
	return out
}

// Goal is what a counter is FOR, from where the pet is standing now.
type Goal struct {
	// Form is the shape the counter opens, and Threshold what it asks for.
	// Both are zero when the counter opens nothing this pet can still reach.
	Form      string
	Threshold int
	Done      int
}

// Reached says the counter has already paid for its goal.
func (g Goal) Reached() bool { return g.Threshold > 0 && g.Done >= g.Threshold }

// Leads says the counter is going somewhere at all.
func (g Goal) Leads() bool { return g.Form != "" }

// GoalOf is the nearest shape a counter still opens for this pet.
//
// The panel prints twenty-two counters and they are not equal: some are two
// away from a mark, some belong to a branch this pet walked past years ago and
// will never see again. Without this they were twenty-two identical white
// numbers, which is a table with no information in it.
//
// Nearest means the mark before the title, because the mark comes first. A
// counter whose shapes are all behind the pet gets an empty Goal - it is still
// counted, it just no longer leads anywhere.
func GoalOf(s *State, form, counter string) Goal {
	ahead := Ahead(form)
	done := s.Counters[counter]
	if done < 0 {
		done = 0
	}
	// Marks first: a mark is always closer than the title behind it.
	for _, stage := range []func() (string, int, bool){
		func() (string, int, bool) { return firstUnlock(ahead, counter) },
		func() (string, int, bool) { return firstTitle(ahead, counter) },
	} {
		if shape, threshold, ok := stage(); ok {
			return Goal{shape, threshold, done}
		}
	}
	return Goal{Done: done}
}

func firstUnlock(ahead map[string]bool, counter string) (string, int, bool) {
	for _, mark := range sortedForms(Unlocks) {
		if ahead[mark] && Unlocks[mark].Counter == counter {
			return mark, Unlocks[mark].Threshold, true
		}
	}
	return "", 0, false
}

func firstTitle(ahead map[string]bool, counter string) (string, int, bool) {
	for _, mark := range sortedForms(Unlocks) {
		title, ok := Titles[mark]
		if !ok || !ahead[title] || Unlocks[mark].Counter != counter {
			continue
		}
		if u, ok := TitleUnlock(mark); ok {
			return title, u.Threshold, true
		}
	}
	return "", 0, false
}

// sortedForms keeps the walk deterministic: a map range would pick a different
// mark between two refreshes when a counter opens more than one.
func sortedForms(m map[string]Unlock) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
