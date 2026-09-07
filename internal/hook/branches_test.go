package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// play runs the same day over and over - a few meals, then a session closed
// with a given context peak and length - until the pet reaches the level asked
// for, and reports which branch it took. This is reachability by PLAYING, which
// is not the same as reachability from a hand-built state.
func playTo(t *testing.T, level int, meals []string, peak float64, mins int) (string, *pet.State) {
	t.Helper()
	home := t.TempDir()
	statePath := filepath.Join(home, "pet.json")
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	for d := 0; d < 900; d++ {
		s := pet.Load(statePath)
		if pet.LevelFor(s.XP) >= level {
			break
		}
		at := start.AddDate(0, 0, d)
		for i, m := range meals {
			pet.Feed(s, m, "", at.Add(time.Duration(i)*90*time.Minute))
		}
		pet.Save(s, statePath)

		facts := filepath.Join(t.TempDir(), "facts.json")
		raw, _ := json.Marshal(map[string]any{
			"label": "x", "peak": peak, "t0": at.Unix(),
			"repo": "r", "structured": true})
		if err := os.WriteFile(facts, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		CloseSession(facts, statePath, at.Add(time.Duration(mins)*time.Minute))
	}
	s := pet.Load(statePath)
	form, _ := pet.CurrentForm(s)
	return form, s
}

func play(t *testing.T, meals []string, peak float64, mins int) (string, *pet.State) {
	t.Helper()
	return playTo(t, 2, meals, peak, mins)
}

func TestEveryTemperamentIsReachableByPlaying(t *testing.T) {
	// Three branches leave the larva. If one of them cannot be taken by any
	// way of working, a third of the tree is decoration.
	for _, c := range []struct {
		want  string
		how   string
		meals []string
		peak  float64
		mins  int
	}{
		{"pattern", "commits and compacts, context kept low",
			[]string{"commit", "compact"}, 45, 120},
		{"probe", "tests and plan tasks",
			[]string{"tests", "task"}, 45, 120},
		{"ember", "run at the limit, barely closing anything",
			[]string{"feed"}, 92, 300},
	} {
		form, s := play(t, c.meals, c.peak, c.mins)
		if form != c.want {
			t.Errorf("%s (%s) gave %q, want %q  [m %d / i %d / imp %d]",
				c.want, c.how, form, c.want,
				s.Counters["methodical"], s.Counters["inquisitive"],
				s.Counters["impulsive"])
		}
	}
}

func TestImpulsiveHasASourceThatDoesNotCostXP(t *testing.T) {
	// The arithmetic that broke the ember branch: its only source was the one
	// meal that takes XP away, so every point of it cost 15 XP while every
	// meal that paid that back fed a rival. The branch could not be climbed.
	before := pet.New()
	before.XP = 500
	statePath := filepath.Join(t.TempDir(), "pet.json")
	pet.Save(before, statePath)

	at := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	facts := filepath.Join(t.TempDir(), "facts.json")
	raw, _ := json.Marshal(map[string]any{
		"label": "x", "peak": float64(ImpulsivePeak), "t0": at.Unix(),
		"repo": "r", "structured": true})
	if err := os.WriteFile(facts, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	CloseSession(facts, statePath, at.Add(2*time.Hour))

	after := pet.Load(statePath)
	if after.Counters["impulsive"] == 0 {
		t.Error("a session run at the limit fed nothing to the ember branch")
	}
	if after.XP < before.XP {
		t.Errorf("it cost %d xp; the branch cannot be climbed if it does",
			before.XP-after.XP)
	}
}

func TestACalmSessionDoesNotFeedTheEmberBranch(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "pet.json")
	pet.Save(pet.New(), statePath)
	at := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	facts := filepath.Join(t.TempDir(), "facts.json")
	raw, _ := json.Marshal(map[string]any{
		"label": "x", "peak": float64(ImpulsivePeak) - 1, "t0": at.Unix(),
		"repo": "r", "structured": true})
	if err := os.WriteFile(facts, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	CloseSession(facts, statePath, at.Add(2*time.Hour))

	if s := pet.Load(statePath); s.Counters["impulsive"] != 0 {
		t.Errorf("a session under the threshold fed impulsive: %d",
			s.Counters["impulsive"])
	}
}

func TestCtxMaxedHasASourceThatDoesNotCostXP(t *testing.T) {
	// The same closed arithmetic that killed the ember branch, one level down:
	// ctx_maxed picks `feral` over its two siblings, and its only source used
	// to be the overflow - the one meal that takes XP away - while the siblings
	// were paid by simply having sessions, for free.
	before := pet.New()
	before.XP = 500
	statePath := filepath.Join(t.TempDir(), "pet.json")
	pet.Save(before, statePath)

	at := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	facts := filepath.Join(t.TempDir(), "facts.json")
	raw, _ := json.Marshal(map[string]any{
		"label": "x", "peak": float64(FeralPeak), "t0": at.Unix(),
		"repo": "r", "structured": true})
	if err := os.WriteFile(facts, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	CloseSession(facts, statePath, at.Add(2*time.Hour))

	after := pet.Load(statePath)
	if after.Counters["ctx_maxed"] == 0 {
		t.Error("a session run at the limit fed nothing to feral")
	}
	if after.XP < before.XP {
		t.Errorf("it cost %d xp; the branch cannot be climbed if it does",
			before.XP-after.XP)
	}
}

func TestTheThreeNotchesOfTheEmberBranchAreOrdered(t *testing.T) {
	// 85 says you worked high up, 95 that you did it without easing off, 100
	// that you hit the wall. A session only ever earns the notches it reached.
	for _, c := range []struct {
		peak                            float64
		impulsive, ctxMaxed, ctx100Sess int
	}{
		{84, 0, 0, 0},
		{85, 1, 0, 0},
		{94, 1, 0, 0},
		{95, 1, 1, 0},
		{99, 1, 1, 0},
		{100, 1, 1, 1},
	} {
		statePath := filepath.Join(t.TempDir(), "pet.json")
		pet.Save(pet.New(), statePath)
		at := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
		facts := filepath.Join(t.TempDir(), "facts.json")
		raw, _ := json.Marshal(map[string]any{
			"label": "x", "peak": c.peak, "t0": at.Unix(),
			"repo": "r", "structured": true})
		if err := os.WriteFile(facts, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		CloseSession(facts, statePath, at.Add(2*time.Hour))

		s := pet.Load(statePath)
		if s.Counters["impulsive"] != c.impulsive ||
			s.Counters["ctx_maxed"] != c.ctxMaxed ||
			s.Counters["ctx100_sessions"] != c.ctx100Sess {
			t.Errorf("peak %v gave impulsive=%d ctx_maxed=%d ctx100=%d, want %d/%d/%d",
				c.peak, s.Counters["impulsive"], s.Counters["ctx_maxed"],
				s.Counters["ctx100_sessions"], c.impulsive, c.ctxMaxed, c.ctx100Sess)
		}
	}
}

func TestEveryTradeOffTheEmberBranchIsReachableByPlaying(t *testing.T) {
	// Level 3 this time. `feral` is the one that could not be reached: its
	// counter cost XP and its two siblings did not, so playing hard enough to
	// earn it kept you off level 3 entirely.
	for _, c := range []struct {
		want  string
		how   string
		meals []string
		peak  float64
		mins  int
	}{
		{"sprinter", "short sessions, run high",
			[]string{"feed"}, 92, 10},
		{"marathon", "long sessions, run high",
			[]string{"feed"}, 92, 300},
		// Under 90 minutes feeds neither duration counter, so the only thing
		// rising is ctx_maxed. That is the shape feral describes: filling the
		// window fast and never easing off.
		{"feral", "run at the limit and never ease off",
			[]string{"feed"}, 97, 60},
	} {
		form, s := playTo(t, 3, c.meals, c.peak, c.mins)
		if form != c.want {
			t.Errorf("%s (%s) gave %q  [short %d / long %d / maxed %d / xp %d]",
				c.want, c.how, form, s.Counters["short_sessions"],
				s.Counters["long_sessions"], s.Counters["ctx_maxed"], s.XP)
		}
	}
}

func TestALongSessionAtTheLimitStillGoesToMarathon(t *testing.T) {
	// Worth pinning down, because it is not obvious and it is easy to read as
	// a bug: a session over 90 minutes at 95%+ feeds long_sessions AND
	// ctx_maxed, one each. They stay tied forever, and topBranch breaks a tie
	// on the order the design lists the siblings in - sprinter, marathon,
	// feral - so marathon takes it.
	//
	// feral is for filling the window FAST. If you want it, the sessions have
	// to be the short kind, which is the distinction the branch is drawing.
	form, s := playTo(t, 3, []string{"feed"}, 97, 300)
	if form != "marathon" {
		t.Errorf("form = %q, want marathon  [long %d / maxed %d]",
			form, s.Counters["long_sessions"], s.Counters["ctx_maxed"])
	}
	if s.Counters["ctx_maxed"] != s.Counters["long_sessions"] {
		t.Errorf("the two stopped being tied: maxed %d, long %d",
			s.Counters["ctx_maxed"], s.Counters["long_sessions"])
	}
}

// closeWith runs a session close over a facts file with the given peaks and
// hands back the counters. A ctxPeak below zero leaves ctx_peak out of the
// file entirely, which is what a session written before the two were told
// apart looks like.
// closeWith runs a session end over a facts file. quotaPeak is written as the
// legacy `peak` key - the neck peak the counters used to read - and a ctxPeak
// below zero leaves the file without a `ctx_peak` at all, which is what a file
// written before the two were told apart looks like.
func closeWith(t *testing.T, quotaPeak, ctxPeak float64) map[string]int {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "pet.json")
	pet.Save(pet.New(), statePath)
	at := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	doc := map[string]any{
		"label": "x", "peak": quotaPeak, "t0": at.Unix(), "repo": "r", "structured": true}
	if ctxPeak >= 0 {
		doc["ctx_peak"] = ctxPeak
	}
	raw, _ := json.Marshal(doc)
	facts := filepath.Join(t.TempDir(), "facts.json")
	if err := os.WriteFile(facts, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	CloseSession(facts, statePath, at.Add(2*time.Hour))
	return pet.Load(statePath).Counters
}

func TestNoCounterIsPaidOutOfAQuota(t *testing.T) {
	// All five counters here are named for the context and mean it. Three of
	// them always read it. The other two - impulsive and ctx_maxed, the whole
	// of the ember branch - used to read the tightest of the three necks on
	// purpose, on the argument that a squeezed quota is also "working at the
	// limit". It is not the same thing: the quotas are the ACCOUNT's, so
	// running four sessions in parallel spends them without any one window
	// ever filling, and the branch that means "you push without easing off"
	// was payable by opening tabs.
	c := closeWith(t, 100, 15) // a quota at the ceiling; the window stayed empty
	if c["ctx100_sessions"] != 0 {
		t.Error("a quota at 100 was counted as a full context window")
	}
	if c["ctx_low"] != 1 || c["sessions_under_40"] != 1 {
		t.Errorf("a context that peaked at 15 did not read as low: %v", c)
	}
	if c["impulsive"] != 0 || c["ctx_maxed"] != 0 {
		t.Errorf("a quota paid into the ember branch: %v", c)
	}
}

func TestAFullWindowWithIdleQuotasStillCountsEverything(t *testing.T) {
	c := closeWith(t, 100, 100)
	for _, name := range []string{"ctx100_sessions", "impulsive", "ctx_maxed"} {
		if c[name] != 1 {
			t.Errorf("%s = %d, want 1", name, c[name])
		}
	}
	if c["ctx_low"] != 0 || c["sessions_under_40"] != 0 {
		t.Errorf("a full window read as a calm session: %v", c)
	}
}

func TestAFactsFileFromBeforeTheSplitFallsBackToTheOnePeak(t *testing.T) {
	// No ctx_peak in the file. Treating the missing field as zero would say
	// the context never rose, and hand out ctx_low and sessions_under_40 to a
	// session that ran the window to the ceiling.
	c := closeWith(t, 100, -1)
	if c["ctx_low"] != 0 || c["sessions_under_40"] != 0 {
		t.Errorf("an upgraded session was paid for a low context it never had: %v", c)
	}
	if c["ctx100_sessions"] != 1 {
		t.Errorf("ctx100_sessions = %d, want 1", c["ctx100_sessions"])
	}
}
