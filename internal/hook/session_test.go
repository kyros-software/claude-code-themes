package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
)

func writeFacts(t *testing.T, doc map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "facts.json")
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A terminal left open overnight is not a marathon. The duration is the
// working time the statusline booked, plus the tail up to the close if that
// tail is not itself a long pause.
func TestTheDurationIsWorkingTimeNotTheWallClock(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "pet.json")
	t0 := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	// Twenty minutes of work, then the desk left alone until the next morning.
	facts := writeFacts(t, map[string]any{
		"label": "x", "ctx_peak": 30, "t0": t0.Unix(),
		"active": 20 * 60, "seen": t0.Add(20 * time.Minute).Unix(), "repo": "r"})
	CloseSessionWorked(facts, statePath, t0.Add(14*time.Hour), true)
	s := pet.Load(statePath)
	if s.Counters["long_sessions"] != 0 || s.Counters["sessions_4h"] != 0 {
		t.Errorf("an idle night counted as a long session: %v", s.Counters)
	}
	// 20 minutes is not short either: under 15 is.
	if s.Counters["short_sessions"] != 0 {
		t.Errorf("twenty minutes of work counted as a short session")
	}
}

// Facts from before Active existed still close on the wall clock.
func TestOldFactsFallBackToTheWallClock(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "pet.json")
	t0 := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	facts := writeFacts(t, map[string]any{"label": "x", "peak": 30, "t0": t0.Unix(), "repo": "r"})
	CloseSessionWorked(facts, statePath, t0.Add(5*time.Hour), true)
	if s := pet.Load(statePath); s.Counters["sessions_4h"] != 1 {
		t.Errorf("sessions_4h = %d, want 1 from the wall clock", s.Counters["sessions_4h"])
	}
}

// A session that only talked books no session habit; one that reached the
// impulsive peak does, work or not.
func TestASessionWithNoWorkBooksNoHabit(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "pet.json")
	t0 := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	quiet := writeFacts(t, map[string]any{"label": "x", "ctx_peak": 10, "t0": t0.Unix(),
		"active": 300, "seen": t0.Add(5 * time.Minute).Unix(), "repo": "r"})
	CloseSessionWorked(quiet, statePath, t0.Add(6*time.Minute), false)
	s := pet.Load(statePath)
	for _, c := range []string{"ctx_low", "sessions_under_40", "short_sessions", "same_repo_days"} {
		if s.Counters[c] != 0 {
			t.Errorf("a session with no work paid %s = %d", c, s.Counters[c])
		}
	}

	high := writeFacts(t, map[string]any{"label": "x", "ctx_peak": 90, "t0": t0.Unix(),
		"active": 300, "seen": t0.Add(5 * time.Minute).Unix(), "repo": "r"})
	CloseSessionWorked(high, statePath, t0.Add(6*time.Minute), false)
	if s := pet.Load(statePath); s.Counters["impulsive"] != 1 {
		t.Errorf("a session at 90%% paid impulsive = %d, want 1", s.Counters["impulsive"])
	}
}

// The hook marks a session as worked on an edit, a commit or a test run, and
// SessionEnd reads that mark.
func TestTheHookMarksASessionAsWorked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", t.TempDir())
	statePath := filepath.Join(home, "pet.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	closeAfter := func(id string, work map[string]any) *pet.State {
		if ok := session.Save(session.PathFor(id, ""), session.Facts{
			Label: "x", CtxPeak: 10, T0: now.Unix(), Active: 60, Seen: now.Unix(), Repo: "r"}); !ok {
			t.Fatal("could not write facts")
		}
		if work != nil {
			fire(t, statePath, now, work)
		}
		return fire(t, statePath, now.Add(time.Minute),
			map[string]any{"hook_event_name": "SessionEnd", "session_id": id})
	}

	if s := closeAfter("talk", nil); s.Counters["ctx_low"] != 0 {
		t.Fatalf("a session with no tool use paid ctx_low = %d", s.Counters["ctx_low"])
	}
	if s := closeAfter("work", anEdit("work")); s.Counters["ctx_low"] != 1 {
		t.Errorf("a session with an edit paid ctx_low = %d, want 1", s.Counters["ctx_low"])
	}
}

func TestTouchSkipsLongPauses(t *testing.T) {
	var f session.Facts
	f.Touch(1000)
	f.Touch(1000 + 600)                       // ten minutes: work
	f.Touch(1000 + 600 + session.IdleGap + 1) // away from the desk
	f.Touch(1000 + 600 + session.IdleGap + 61)
	if f.Active != 660 {
		t.Errorf("active = %d, want 660", f.Active)
	}
}
