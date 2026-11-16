package voice

import (
	"sort"
	"time"
)

// The kinds of step a session takes, as the hook names them.
const (
	KindEdit   = "edit"
	KindDoc    = "doc"
	KindRead   = "read"
	KindPlan   = "plan"
	KindTest   = "test"
	KindCommit = "commit"
	KindSystem = "system"
	KindOther  = "other"
)

const (
	// ruleNeeds is how many steps a session takes before the rules say what
	// it is. More than the model needs: a tally of four is a coin toss.
	ruleNeeds = 8
	// filesKept bounds the per-file edit counts a session carries.
	filesKept = 40
)

// note books one step in the session's tally.
func (f *File) note(c Context) {
	if c.Kind == "" {
		return
	}
	if f.Tally == nil {
		f.Tally = map[string]int{}
	}
	f.Tally[c.Kind]++
	f.Steps++
	if f.Tools == nil {
		f.Tools = map[string]int{}
	}
	if len(f.Tools) < filesKept || f.Tools[c.Tool] > 0 {
		f.Tools[c.Tool]++
	}
	switch c.Kind {
	case KindTest:
		// A suite run before anything was touched: looking at the failure
		// first. It is what tells a hunt from a suite being kept green.
		if f.Tally[KindEdit] == 0 {
			f.Repro = true
		}
	case KindEdit:
		if f.Files == nil {
			f.Files = map[string]int{}
		}
		if len(f.Files) < filesKept || f.Files[c.Target] > 0 {
			f.Files[c.Target]++
		}
	}
}

// settle halves the tally once a shape has been read off it, so the next
// reading is mostly about what the session did since. Without it a session
// that spent its first hour on tests would be an exterminador for the day.
func (f *File) settle() {
	// Whether the failure was looked at first is a fact about a stretch of
	// work, not about the session: kept, one early test run made every suite
	// for the rest of the day a hunt.
	f.Repro = false
	for _, m := range []map[string]int{f.Tally, f.Files, f.Tools} {
		for k, v := range m {
			if v /= 2; v == 0 {
				delete(m, k)
			} else {
				m[k] = v
			}
		}
	}
}

// Rules reads a session's shape off what it has done, with no model: the line
// of the tree whose kind of work this is. It is the first of Candidates.
//
// It is coarser than asking. It tells tests from docs from a wide refactor; it
// cannot tell exploring a design from reading around. What it is, is free,
// instant and the same answer every time - which is why it is what runs when
// nobody has switched the voice on, and what runs first when somebody has.
func Rules(f File, c Context, worn map[string]int) string {
	if all := Candidates(f, c, worn); len(all) > 0 {
		return all[0]
	}
	return ""
}

// typical is the share of a session's steps each kind of work takes when the
// session is about that: what a fit of 1 means. Small for the kinds that are
// one command among many - a suite is run once for every dozen edits - and
// large for the ones that are most of what any session does.
//
// These replaced fixed thresholds (docs over 30%, tests over 20%, a plan over
// 25%), which three weeks of one machine's real sessions - 182 of them, 68,000
// tool calls - showed to be bars almost nothing cleared: no sabueso, no
// cartógrafo, six exterminadores, while tejedor took 29% of all steps.
var typical = map[string]float64{
	KindDoc: 0.10, KindTest: 0.05, KindPlan: 0.04, KindSystem: 0.10,
	KindEdit: 0.35, KindRead: 0.60,
}

// NoveltyWeight is what a fit is worth for a line already worn once more than
// the least-worn one, and again for each time after that, down to noveltyFloor
// steps. It is pet.SeenWeight's idea, across sessions instead of across lives:
// the work still chooses, and a line that keeps being chosen has to be fitted
// harder each time, until the others get their turn.
const (
	NoveltyWeight = 0.7
	noveltyFloor  = 8
	// doubtBand is how close to the best a fit has to be to be a doubt.
	doubtBand = 0.8
	// keepBand is how close to the best the shape a session already wears
	// has to be to stay on. Wide on purpose: a shape that flips every half
	// hour between two near-equal fits is a pet nobody recognises.
	keepBand = 0.5
	// liveWeight is how many wears a line counts for while another open
	// session has it on: one more, so NoveltyWeight of the fit. Windows doing
	// the same kind of work come out as different creatures when the work
	// shows a second sign nearly as strong; plain work is still what it is.
	liveWeight = 1
	// signFloor is the least fit that counts as a sign of anything. One read
	// among nine commands is not a monje.
	signFloor = 0.3
)

// fits is how well the session fits each line of the tree, before novelty:
// zero for no sign of it, about one for a session that is plainly that.
func fits(f File, c Context) map[string]float64 {
	t := f.Tally
	total := 0
	for _, n := range t {
		total += n
	}
	out := map[string]float64{}
	if total == 0 {
		return out
	}
	of := func(kind string) float64 { return float64(t[kind]) / float64(total) / typical[kind] }
	top := func(v, most float64) float64 {
		if v > most {
			return most
		}
		return v
	}

	edits, files, most := t[KindEdit], len(f.Files), 0
	for _, n := range f.Files {
		if n > most {
			most = n
		}
	}

	out["gardener"] = top(of(KindDoc), 4)
	if f.Repro && edits > 0 {
		out["bloodhound"] = top(of(KindTest), 4) // the failure was looked at first
	} else {
		out["exterminator"] = top(of(KindTest), 4)
	}
	out["cartographer"] = top(of(KindPlan), 4)
	out["gremlin"] = top(of(KindSystem), 4)
	if t[KindCommit] > 0 && c.Active > 0 && c.Active < 15*60 {
		out["bolt"] = 1.5 // in, a change, out
	}
	// The three shapes of editing: wide, deep in one file, or neither.
	if edits >= 6 {
		out["weaver"] = top(float64(files)/8, 2)
		out["mole"] = top(float64(most)/float64(edits)/0.6, 1.5)
	}
	if edits > 0 && files <= 3 {
		out["surgeon"] = top(of(KindEdit), 1)
	}
	if edits == 0 && total >= 12 {
		out["oracle"] = top(of(KindRead), 1.5)
	}
	out["monk"] = top(of(KindRead), 1.5) * 0.7
	// One tool for everything - and not the editor: eight files edited is one
	// tool too, and it is a tejedor.
	if len(f.Tools) == 1 && f.Steps >= ruleNeeds && edits == 0 && t[KindDoc] == 0 {
		out["sniper"] = 1.2
	}
	if c.Active >= 4*60*60 {
		out["ox"] = 1
	}
	return out
}

// Candidates is the lines of the tree the session fits, best first. One is an
// answer. Two or three within doubtBand of each other is a doubt, and with the
// voice on that is what the model is asked: not which of fourteen, which of
// these.
//
// worn is how many times each line has been put on, on this machine. Every fit
// is weighed by it, which is what lets all fourteen turn up: left to the fit
// alone a person's sessions are the same three shapes for ever, because their
// work is. A session with no sign of anything - it happens to one in six -
// wears the line worn least, since for that session any shape is as true as
// another.
func Candidates(f File, c Context, worn map[string]int) []string {
	all := ranked(f, c, worn)
	var out []string
	for _, s := range all {
		if s.score < all[0].score*doubtBand || len(out) == 3 {
			break
		}
		out = append(out, s.mark)
	}
	return out
}

type scored struct {
	mark  string
	score float64
}

// ranked is every line the session shows a sign of, weighed and best first.
func ranked(f File, c Context, worn map[string]int) []scored {
	// The state of the session outranks the kind of work: a context about to
	// burst is a kraken whatever is being typed into it, and nobody is asked.
	if c.Label == "drowning" {
		return []scored{{"kraken", 1}}
	}
	fit := fits(f, c)
	if len(fit) == 0 {
		return nil
	}
	least := -1
	for _, l := range lines {
		if l.mark == "kraken" {
			continue // only a bursting context wears it: not part of the turn
		}
		if n := worn[l.mark]; least < 0 || n < least {
			least = n
		}
	}
	var all []scored
	for _, l := range lines { // the tree's order, so ties are settled the same way every time
		v := fit[l.mark]
		if v < signFloor {
			continue
		}
		over := worn[l.mark] - least
		if over > noveltyFloor {
			over = noveltyFloor
		}
		for ; over > 0; over-- {
			v *= NoveltyWeight
		}
		all = append(all, scored{l.mark, v})
	}
	if len(all) == 0 {
		// No sign of any kind of work. The line worn least, first in the
		// tree's order among those.
		for _, l := range lines {
			if l.mark != "kraken" && worn[l.mark] <= least {
				return []scored{{l.mark, 0}}
			}
		}
		return nil
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].score > all[j].score })
	return all
}

// rule is the rules' turn: read the shape if one is due. It reports the mark
// newly put on, or "", and the doubt - the lines the session fitted about
// equally - if there is one for the model to settle.
//
// live is asked for the lines the other open sessions wear, and only when a
// shape is really being read: it looks through the scratch directory.
func (f *File) rule(c Context, worn map[string]int, live func() []string, now time.Time) (put string, doubt []string) {
	if f.Steps < ruleNeeds {
		return "", nil
	}
	since := now.Sub(time.Unix(f.MarkAt, 0))
	if f.Mark != "" && since >= 0 && since < ShapeEvery {
		return "", nil
	}
	// The shape it wears is not held against it: its own turn is one of the
	// times that line was put on.
	own := make(map[string]int, len(worn))
	for k, v := range worn {
		own[k] = v
	}
	if f.Mark != "" && own[f.Mark] > 0 {
		own[f.Mark]--
	}
	// Read twice. What the work is, by itself, decides whether the shape it
	// wears still holds; what the other windows wear only weighs on which
	// one it moves to. Weighed together, a session that had turned to
	// nothing but suites stayed a bosque because the avispa was taken.
	plain := ranked(*f, c, own)
	worn = own
	if live != nil {
		crowded := make(map[string]int, len(own))
		for k, v := range own {
			crowded[k] = v
		}
		for _, mark := range live() {
			crowded[mark] += liveWeight
		}
		worn = crowded
	}
	all := ranked(*f, c, worn)
	if len(all) == 0 {
		if f.Mark == "" {
			return "", nil // nothing done at all: asked again at the next step
		}
		f.MarkAt = now.Unix()
		f.settle()
		return "", nil
	}
	mark := all[0].mark
	doubt = Candidates(*f, c, worn)
	for _, s := range plain {
		// It keeps what it wears while that is still a fair reading of the
		// work: the shape moves when the work does, not when two fits tie.
		// And then there is nothing to ask about either.
		if s.mark == f.Mark && s.score >= plain[0].score*keepBand {
			mark, doubt = f.Mark, nil
			break
		}
	}
	if len(doubt) < 2 {
		doubt = nil
	}
	f.MarkAt = now.Unix()
	f.settle()
	if mark == f.Mark {
		return "", doubt
	}
	f.Mark = mark
	return mark, doubt
}
