package pet

// The progress layer. Accumulated XP sets the level, and the SHAPE never goes
// back down the tree - the path only grows, see Advance. The level itself does: XP falls when the
// context blows (-15) and while the pet goes hungry, so a level can be lost,
// which TestStarvingCanCostALevelButNeverKills has always said and this
// comment used to deny. TestTheLevelNeverGoesDown is about LevelFor being
// monotonic in XP, which is a different claim and the only one that holds.
//
// At every fork the branch is not chosen by the user but by the habits of the
// stage that just ended (v3), so the shape you end up with is a readout of how
// you have been working lately.

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
//
// v3 (7-nov-2026): the ladder is one generation, and each rung is a STAGE
// whose habits decide the next fork, once. So every stage has to last long
// enough for a habit to show, and the whole life has to end in about two
// weeks, or there is no next life to see other forms in. At the design's 128
// XP for a normal day:
//
//	1 -> 2   200   ~1.6 days   habits that pick the temperament
//	2 -> 3   250   ~2 days     the trade
//	3 -> 5   550   ~4.3 days   the mark (level 4 forks nothing)
//	5 -> 6   500   ~3.9 days   the title
//
// About twelve days to the title, then TitleStay days wearing it. Level 2 at
// 60 was half a day: the per-session habits are booked as sessions close, and
// half a day had none of them yet, so the fork went to commits every time.
var Levels = []Level{{0, 1}, {200, 2}, {450, 3}, {700, 4}, {1000, 5}, {1500, 6}}

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
//
// v3 (7-nov-2026): tests and inquisitive went from 8 to 10. The 8 was the
// hourly cooldown - eight suites in a day was the most that could count - and
// v3 takes the cooldown off the habit (it still brakes the XP). Measured on
// one user's transcripts, a day leaning on tests runs about 27 suites and a day
// leaning on commits about 30 commits, so the two habits are the same size of
// number and get the same scale. [hipotesis] One user's month is a thin base;
// recalibrate with more.
var BranchScale = map[string]int{
	// Per event, off the 128 XP day.
	"methodical":  10, // commits at 12 xp, plus the compacts
	"inquisitive": 10, // green suites, plus closed tasks
	"diffs":       10, // commits alone
	"tests":       10, // green suites with an edit behind them
	"plans":       21, // closed plan tasks at 6 xp

	// Per session, capped by how many sessions fit in a day.
	"impulsive":      3, // one per session over 85%
	"ctx_low":        3, // one per session under 60%
	"ctx_maxed":      3, // one per session over 95%
	"short_sessions": 3, // one per session under 15 min
	"long_sessions":  2, // one per session over 90 min - fewer of those fit
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
//
// v2 gated each one behind its mark's habit, two and a half to five times
// over (TitleAsks). v3 hands the title over with the XP: a generation has to
// end for the next one to start, and a gate that one pet in ten clears is a
// generation that never ends. The habit already chose the mark; the title is
// that mark grown up.
var Titles = map[string]string{
	"surgeon": "scalpel", "weaver": "loom", "monk": "abbot", "gardener": "forest",
	"bloodhound": "wolf", "exterminator": "wasp", "cartographer": "atlas",
	"oracle": "sphinx", "bolt": "storm", "sniper": "falcon", "ox": "mammoth",
	"mole": "worm", "gremlin": "devil", "kraken": "leviathan",
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

// NextMark is the mark a trade is heading for: of its two marks, the one whose
// habit has come furthest THIS STAGE relative to what it asks. It is the race
// that pick will settle at level 5, shown while it is still being run.
//
// Only a trade has one. In v3 a mark's title arrives with the XP, not with a
// habit three times over, so a pet wearing a mark - or a secret, or a title -
// has its progress in the XP bar and gets false here.
//
// form is the pet's current form, as CurrentForm gave it.
func NextMark(s *State, form string) (Mark, bool) {
	if Tier(form) != 3 {
		return Mark{}, false
	}
	kid, ok := ripestMark(s, Tree[form], false)
	if !ok {
		return Mark{}, false
	}
	u := Unlocks[kid]
	done := s.Stage(u.Counter)
	if done < 0 {
		done = 0
	}
	// Clamped for the DISPLAY only: a bar does not overflow.
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
		done := s.Stage(u.Counter)
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

// RememberForm records the rung a form has reached. In v3 the path is the
// memory and nothing reads this back to hold the shape up; it is still written
// so that an older build reading the file keeps its floor.
func RememberForm(s *State, form string) {
	if Tier(form) >= Tier(s.FormSeen) {
		s.FormSeen = form
	}
}

// The v3 rule: a fork is decided ONCE, when the pet crosses into it, by the
// habits of the stage that just ended - and then it stays decided for the rest
// of the generation.
//
// v2 recomputed every fork from lifetime counters on every refresh, and that
// had two consequences it then spent most of this file fighting. The shape
// flickered between near-equal habits, so a fork taken had to be DEFENDED by a
// margin (BranchMargin, RememberBranch). And lifetime counters only grow, so a
// pet's branch was settled in its first weeks for good: one pet measured at
// methodical 1036 could not become anything else whatever it did. It wore 5
// forms out of 41 in two months.
//
// The fix is the v-pets' (Digimon resets care mistakes and training at every
// evolution): what decides a fork is what happened during that stage. See
// State.Stage and docs/design/v3-generaciones.md.

// targetTier is the rung a pet at this level stands on.
func targetTier(level int) int {
	switch {
	case level >= 6:
		return 6
	case level == 5:
		return 5
	case level >= 3:
		return 3
	}
	return level
}

// stageShare is how far a fork counter has come THIS STAGE, in its own unit:
// days of that habit. A counter with no scale divides by one.
func stageShare(s *State, form string) float64 {
	counter := BranchBy[form]
	if counter == "" {
		return 0
	}
	return float64(s.Stage(counter)) / float64(scaleOf(counter))
}

// pick decides the fork at parent with this stage's habits. Nothing it reads
// is written: it is the same answer whether the pet is crossing for real or
// the panel is asking what the next level would look like.
//
// Every fork has a route by default, the first child in design order, which is
// where a stage with no habit at all goes - the Digimon fallback. A pet is
// never stuck on a rung for want of a habit.
func pick(s *State, parent string) string {
	kids := Tree[parent]
	if len(kids) == 1 {
		return kids[0] // a mark's title
	}
	if _, isMarkFork := Unlocks[kids[0]]; isMarkFork {
		if kid, ok := ripestMark(s, kids, false); ok && stageDone(s, kid) > 0 {
			return kid
		}
		return kids[0]
	}
	best, bestShare := kids[0], stageShare(s, kids[0])
	for _, kid := range kids[1:] {
		if share := stageShare(s, kid); share > bestShare {
			best, bestShare = kid, share
		}
	}
	return best
}

// stageDone is a mark's habit this stage.
func stageDone(s *State, mark string) int {
	u, ok := Unlocks[mark]
	if !ok {
		return 0
	}
	return s.Stage(u.Counter)
}

// pathEnd follows the decisions already taken. A link that is not a real edge
// of the tree ends the path rather than being followed: Load drops those, so
// this only guards a state built in memory.
func pathEnd(s *State) string {
	here := Root
	for i := 0; i < 8; i++ {
		next, ok := s.Branch[here]
		if !ok || Parent[next] != here {
			break
		}
		here = next
	}
	return here
}

// Advance takes the forks the pet's level has reached and not yet decided,
// writes them down, starts a new stage after each one, and books every new
// form in the album. It returns the forms reached, in order.
//
// For callers that PERSIST the state - Save calls it, so no path can forget.
// The panel's what-if pet must not: asking what the next level would look like
// is not the pet living through it. CurrentForm answers that question without
// writing anything, by deciding the pending forks the same way in memory.
func Advance(s *State, at int64) []string {
	var reached []string
	level := LevelFor(s.XP)
	here := pathEnd(s)
	for Tier(here) < targetTier(level) && len(Tree[here]) > 0 {
		next := pick(s, here)
		if s.Branch == nil {
			s.Branch = map[string]string{}
		}
		s.Branch[here] = next
		s.Collect(next, at)
		if Tier(next) == 6 {
			s.TitleDays = 1 // the day it arrives is the first day it is worn
		}
		reached = append(reached, next)
		here = next
	}
	// One new stage for the whole jump, not one per fork: the habits of the
	// stage that just ended decide every fork crossed in it. Resetting between
	// forks left the later ones to decide on nothing - the route by default -
	// whenever a pet crossed two levels at once.
	if len(reached) > 0 {
		s.NewStage()
	}
	return reached
}

// TitleStay is how many days of use a title is worn before the generation
// closes and an egg is laid. Long enough to be seen, short enough that the
// next life comes: a title used to be the end of the game.
const TitleStay = 3

// AtTitle says the pet's path has reached a title.
func AtTitle(s *State) bool { return Tier(pathEnd(s)) == 6 }

// Hatch closes the generation and starts the next one: a new egg, back at the
// root with no XP and no path, and a fresh stage. What carries over is
// everything that is a record rather than a shape - the album, the lifetime
// counters, the streak, the hunger. Those are yours; the form is the pet's.
func Hatch(s *State, at int64) {
	if s.Generation < 1 {
		s.Generation = 1
	}
	s.Generation++
	s.XP = 0
	s.Branch = map[string]string{}
	s.FormSeen = ""
	s.Secret = ""
	s.LevelSeen = 1
	s.TitleDays = 0
	s.HungerPeak = s.Hunger
	s.NewStage()
	s.Collect(Root, at)
}

// CurrentForm is the shape the pet wears and its level.
//
// The shape is the decided path, plus any fork the XP has reached and Save has
// not yet written down. It never goes back down the tree: the path only grows,
// so a level lost to hunger or a blown context leaves the shape where it was.
// That is the rule v2 needed `floor` and `form_seen` to enforce, and here it
// falls out of the data.
//
// A secret is a rung-5 form and wins the rung it stands on, not the one above:
// a title still outranks it. See walk.
func CurrentForm(s *State) (string, int) {
	return walk(s)
}

func walk(s *State) (string, int) {
	here, level := treeWalk(s)
	if level >= 5 && s.Secret != "" && Tier(here) < 6 {
		if _, ok := Sprites[string(s.Secret)]; ok {
			return string(s.Secret), level
		}
	}
	return here, level
}

// treeWalk is the shape the TREE gives, with no secret laid over it.
func treeWalk(s *State) (string, int) {
	level := LevelFor(s.XP)
	here := pathEnd(s)
	for Tier(here) < targetTier(level) && len(Tree[here]) > 0 {
		here = pick(s, here)
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
		done := s.Stage(u.Counter)
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
	// What this stage has done: that is what the next fork reads. In v3 a
	// title asks for XP, not for a habit, so only a mark can be a goal.
	done := s.Stage(counter)
	if done < 0 {
		done = 0
	}
	if shape, threshold, ok := firstUnlock(ahead, counter); ok {
		return Goal{shape, threshold, done}
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
