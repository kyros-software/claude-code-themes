package pet

import (
	"encoding/json"
	"os"
	"testing"
)

// Reachability for the NINETY-SEVEN form tree, which is design work and not
// code yet: the shapes live in testdata/ATLAS-97.json and the doors in
// testdata/PUERTAS-97.json, and nothing in the runtime reads either.
//
// It is tested anyway, and here, because the last time this question was asked
// the answer arrived as prose and the arithmetic behind it died with the
// session. reach_test.go already asks it of the forty-one; this asks the same
// of the ninety-seven, off a file, so the next person can run it instead of
// believing it.
//
// The question is NOT "does the tree list the mark" - it lists all forty-two.
// It is "can a pet ever be that mark", and the two come apart because the ten
// counters the canvas picks marks with are not independent. Four of them are
// sums of others (`internal/pet/feeding.go:58`) and five more are nested
// notches of one measurement (`internal/hook/app.go:411-460`), so a race
// between a sum and its own addend has a winner before it starts. The doors
// file carries that structure in its `contadores` table, and everything below
// is derived from it rather than written down twice.

type gateTree struct {
	Bases    []string `json:"bases"`
	Counters map[string]struct {
		Of []string `json:"suma"`
	} `json:"contadores"`
	Temperaments []struct {
		Form    string   `json:"forma"`
		Counter string   `json:"contador"`
		Trades  []string `json:"oficios"`
	} `json:"temperamentos"`
	Trades []struct {
		Form    string `json:"forma"`
		Counter string `json:"contador"`
		Marks   []struct {
			Mark      string `json:"marca"`
			Title     string `json:"título"`
			Counter   string `json:"contador"`
			Threshold int    `json:"umbral"`
			TitleAsks int    `json:"umbral_título"`
			Canvas    string `json:"lienzo"`
		} `json:"marcas"`
	} `json:"oficios"`
}

func loadGates(t *testing.T) gateTree {
	t.Helper()
	raw, err := os.ReadFile("testdata/PUERTAS-97.json")
	if err != nil {
		t.Fatal(err)
	}
	var g gateTree
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

// derive turns the independent bases into every counter the tree reads. A
// counter is the sum of the disjoint bases listed for it, which is what makes
// `impulsive >= ctx_maxed >= ctx100_sessions` true by construction here for
// the same reason it is true in the hook: they count nested slices of one peak.
func (g gateTree) derive(base map[string]int) map[string]int {
	c := make(map[string]int, len(g.Counters)+len(base))
	for k, v := range base {
		c[k] = v
	}
	for name, def := range g.Counters {
		sum := 0
		for _, b := range def.Of {
			sum += base[b]
		}
		c[name] = sum
	}
	return c
}

// walk is levels 2 and 3: highest counter wins, ties fall back to list order.
// Same rule as topBranch, and the canvas keeps it - only level 5 changes.
func (g gateTree) walk(c map[string]int) (string, string) {
	temper := g.Temperaments[0]
	for _, cand := range g.Temperaments[1:] {
		if c[cand.Counter] > c[temper.Counter] {
			temper = cand
		}
	}
	trade := ""
	for _, name := range temper.Trades {
		if trade == "" || c[g.tradeCounter(name)] > c[g.tradeCounter(trade)] {
			trade = name
		}
	}
	return temper.Form, trade
}

func (g gateTree) tradeCounter(form string) string {
	for _, tr := range g.Trades {
		if tr.Form == form {
			return tr.Counter
		}
	}
	return ""
}

// winner is level 5: the largest share of what the mark asks for, ties to list
// order. That is ripestMark, which the runtime already does for the fourteen
// marks it has - the ninety-seven tree needs no new machinery for this, only a
// counter and a threshold per mark.
//
// canvas=true reads the door the design canvas gave the mark and ignores the
// threshold, which is the rule as drawn: a race between raw counts.
func (g gateTree) winner(c map[string]int, trade string, canvas bool) (string, bool) {
	var marks []struct {
		name string
		lot  float64
	}
	for _, tr := range g.Trades {
		if tr.Form != trade {
			continue
		}
		for _, m := range tr.Marks {
			counter, threshold := m.Counter, m.Threshold
			if canvas {
				counter, threshold = m.Canvas, 1
			}
			marks = append(marks, struct {
				name string
				lot  float64
			}{m.Mark, float64(c[counter]) / float64(threshold)})
		}
	}
	best, ties := 0, 1
	for i := 1; i < len(marks); i++ {
		switch {
		case marks[i].lot > marks[best].lot:
			best, ties = i, 1
		case marks[i].lot == marks[best].lot:
			ties++
		}
	}
	return marks[best].name, ties == 1
}

// witness builds a pet that IS the mark, rather than hunting for one at random.
//
// Random search was tried first and lies in the safe direction: it reported two
// reachable marks as dead because it never happened to roll the shape that wins
// them. So this walks up instead - it puts the mark's own counter high, then
// raises whatever is needed to land on the right temperament and trade, and
// never lowers anything. If no witness exists for any of the sweep values, the
// mark cannot be won.
//
// The loop over feeders matters: a compound counter can be raised through more
// than one base, and the base chosen also feeds a rival. Raising `inquisitive`
// through `plans` hands the branch to whichever sibling asks for `plans`, so
// every combination gets a turn before a mark is called unreachable.
func (g gateTree) witness(mark string, canvas bool, floor int) map[string]int {
	temperOf, trade, door := "", "", ""
	for _, tr := range g.Trades {
		for _, m := range tr.Marks {
			if m.Mark != mark {
				continue
			}
			trade, door = tr.Form, m.Counter
			if canvas {
				door = m.Canvas
			}
		}
	}
	temperCounter := ""
	for _, tp := range g.Temperaments {
		for _, name := range tp.Trades {
			if name == trade {
				temperOf, temperCounter = tp.Form, tp.Counter
			}
		}
	}

	sweep := []int{1, 2, 3, 4, 5, 6, 7, 8, 10, 13, 17, 22, 30, 45, 70, 110, 180,
		300, 500, 900, 1600, 3000, 6000, 15000, 50000, 200000, 2000000}
	for _, feedMark := range g.Counters[door].Of {
		for _, feedTemper := range g.Counters[temperCounter].Of {
			for _, feedTrade := range g.Counters[g.tradeCounter(trade)].Of {
				for _, v := range sweep {
					base := map[string]int{}
					for _, b := range g.Bases {
						base[b] = 0
					}
					base[feedMark] = v
					if g.settle(base, temperOf, trade, feedTemper, feedTrade) {
						c := g.derive(base)
						if c[door] < floor {
							continue
						}
						if won, alone := g.winner(c, trade, canvas); won == mark && alone {
							return base
						}
					}
				}
			}
		}
	}
	return nil
}

// settle raises counters until the pet lands on the wanted temperament and
// trade, or gives up. Only ever upwards: playing cannot lower a counter, so a
// witness that needed a subtraction would not be one.
func (g gateTree) settle(base map[string]int, temper, trade, feedTemper, feedTrade string) bool {
	for range 60 {
		c := g.derive(base)
		haveTemper, haveTrade := g.walk(c)
		if haveTemper == temper && haveTrade == trade {
			return true
		}
		if haveTemper != temper {
			want, rival := "", 0
			for _, tp := range g.Temperaments {
				if tp.Form == temper {
					want = tp.Counter
				} else if c[tp.Counter] > rival {
					rival = c[tp.Counter]
				}
			}
			base[feedTemper] += max(1, rival+1-c[want])
			continue
		}
		want, rival := g.tradeCounter(trade), 0
		for _, tp := range g.Temperaments {
			if tp.Form != temper {
				continue
			}
			for _, name := range tp.Trades {
				if name != trade && c[g.tradeCounter(name)] > rival {
					rival = c[g.tradeCounter(name)]
				}
			}
		}
		base[feedTrade] += max(1, rival+1-c[want])
	}
	return false
}

func TestEveryMarkOfTheNinetySevenTreeCanStillBeWon(t *testing.T) {
	g := loadGates(t)
	marks := 0
	for _, tr := range g.Trades {
		for _, m := range tr.Marks {
			marks++
			if g.witness(m.Mark, false, 0) == nil {
				t.Errorf("%s (%s, asks %d of %s) cannot be won by any pet: "+
					"another door in its trade is a counter that contains this "+
					"one, and asks proportionally less",
					m.Mark, tr.Form, m.Threshold, m.Counter)
			}
		}
	}
	if marks != 42 {
		t.Errorf("marks = %d, want 42", marks)
	}
}

// Levels 2 and 3, which the marks only prove in passing: a witness for a mark
// had to walk through its temperament and its trade to exist at all. Asserting
// it here says so out loud, and catches the case the runtime has already been
// bitten by once - `feral` was unreachable for a while because its counter was
// the only one that cost XP, so the branch could not be climbed at all.
func TestEveryTemperamentAndTradeCanBeReached(t *testing.T) {
	g := loadGates(t)
	for _, tp := range g.Temperaments {
		for _, trade := range tp.Trades {
			base := map[string]int{}
			for _, b := range g.Bases {
				base[b] = 0
			}
			reached := false
			for _, feedTemper := range g.Counters[tp.Counter].Of {
				for _, feedTrade := range g.Counters[g.tradeCounter(trade)].Of {
					fresh := map[string]int{}
					for k := range base {
						fresh[k] = 0
					}
					if g.settle(fresh, tp.Form, trade, feedTemper, feedTrade) {
						reached = true
					}
				}
			}
			if !reached {
				t.Errorf("no pet can ever be a %s (%s): its trade counter %q "+
					"cannot be raised past its siblings", trade, tp.Form,
					g.tradeCounter(trade))
			}
		}
	}
}

// The forty-two TITLES. A title races nobody - it sits behind its mark and asks
// for MORE of the same counter (TitleUnlock) - so the question is not who wins,
// it is whether the pet can go on raising that counter far enough WITHOUT
// losing the mark on the way. That is not free: every counter here feeds
// something else. Reaching `volcán` means raising sessions_4h to forty, which
// raises long_sessions by forty too, which is the door `kraken` is standing at.
//
// So each title gets a witness of its own: a pet that wins the mark AND is
// already past the title's number.
func TestEveryTitleCanBeReachedWithoutLosingItsMark(t *testing.T) {
	g := loadGates(t)
	for _, tr := range g.Trades {
		for _, m := range tr.Marks {
			switch {
			case m.TitleAsks <= 0:
				t.Errorf("%s, the title behind %s, has no threshold", m.Title, m.Mark)
			case m.TitleAsks <= m.Threshold:
				t.Errorf("%s asks %d of %s and its mark %s already asks %d: a title "+
					"that arrives with its mark is not a title",
					m.Title, m.TitleAsks, m.Counter, m.Mark, m.Threshold)
			case g.witness(m.Mark, false, m.TitleAsks) == nil:
				t.Errorf("%s cannot be reached: no pet holds %s while carrying "+
					"%d of %s", m.Title, m.Mark, m.TitleAsks, m.Counter)
			}
		}
	}
}

// Where the forty-two numbers come from, since this is the one place in the
// tree that had none.
//
// TitleAsks got its fourteen off the canvas, one factor per title, replacing a
// uniform multiplier that had been invented in this file - so writing a fresh
// uniform multiplier for the ninety-seven would put back exactly what that
// change removed. Ten of the twelve counters used by the forty-two doors are
// covered by a title the canvas DID calibrate, for the same counter or for a
// verified twin of it: `sessions_15min` is the same number as `short_sessions`
// (app.go:452-453), `ctx100_sessions` and `sessions_4h` are the same counters
// leviathan and mammoth ask for. Those ten factors are inherited, not chosen.
//
// `methodical` and `impulsive` have no relative among the fourteen and carry
// the median of the canvas's own twelve factors instead. Two invented numbers
// out of forty-two, both flagged in the file, is the floor available without
// the canvas back.
func TestTheTitleFactorsStayInsideTheCanvasRange(t *testing.T) {
	g := loadGates(t)
	for _, tr := range g.Trades {
		for _, m := range tr.Marks {
			factor := float64(m.TitleAsks) / float64(m.Threshold)
			if factor < 2.0 || factor > 4.5 {
				t.Errorf("%s asks x%.2f of %s, outside the x2.0-x4.5 the canvas "+
					"spent on its own fourteen titles", m.Title, factor, m.Mark)
			}
		}
	}
}

// The defect, pinned. The canvas picks a mark with a race between raw counts
// among the same ten counters that already decided the temperament and the
// trade - so twenty-one of the forty-two are unwinnable, and six more are
// only winnable on an exact tie the list order happens to award them.
//
// This test exists so that "put the canvas doors back, they read better" is a
// red suite and not a silent amputation. If it ever goes green, the reason is
// worth reading before the assertion is deleted.
func TestTheCanvasRuleLeavesHalfTheTreeUnwinnable(t *testing.T) {
	g := loadGates(t)
	var dead []string
	for _, tr := range g.Trades {
		for _, m := range tr.Marks {
			if g.witness(m.Mark, true, 0) == nil {
				dead = append(dead, m.Mark)
			}
		}
	}
	if len(dead) != 27 {
		t.Errorf("the canvas rule leaves %d marks unwinnable outright, want 27: %v",
			len(dead), dead)
	}
}
