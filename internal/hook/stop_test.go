package hook

import (
	"os"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/invaders"
	"github.com/kyros-software/claude-code-themes/internal/pet"
)

func stopPayload(session string) map[string]any {
	return map[string]any{"hook_event_name": "Stop", "session_id": session}
}

// Claude finishing is what pauses the game, and it is the only thing a Stop
// does. Not a meal: no XP, no hunger, no counters, no streak - the two systems
// share a config directory and nothing else.
func TestAStopTouchesThePauseFileAndFeedsNothing(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	// A pet with something to lose, saved so there is a file to compare.
	before := pet.New()
	before.XP, before.Hunger, before.Streak = 500, 4, 3
	before.Counters = map[string]int{"methodical": 20, "diffs": 9}
	if !pet.Save(before, statePath) {
		t.Fatal("the pet did not save")
	}
	was, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	fire(t, statePath, now, stopPayload("smoke"))

	info, err := os.Stat(invaders.PausePath())
	if err != nil {
		t.Fatalf("the pause file is not there: %v", err)
	}
	if !info.ModTime().Equal(now) {
		t.Errorf("the pause file is stamped %v, want %v", info.ModTime(), now)
	}

	is, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(is) != string(was) {
		t.Errorf("a Stop rewrote pet.json:\nwas %s\nis  %s", was, is)
	}
}

// A Stop is not a tool call. It returns before the second switch, so a payload
// that happens to carry a tool name must not be counted as one - the sniper
// counts distinct tools between two closed tasks, and a phantom one skews it.
func TestAStopIsNotAToolCall(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	payload := stopPayload("smoke")
	payload["tool_name"] = "Edit"
	s := fire(t, statePath, now, payload)

	if s.XP != 0 || len(s.Counters) != 0 {
		t.Errorf("a Stop carrying a tool name fed the pet: %+v", s)
	}
}

// The game watches the mtime and only pauses for something newer than what it
// saw at startup, so a second Stop has to move it forward or the second pause
// never fires.
func TestASecondStopMovesTheMtimeForward(t *testing.T) {
	first := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	fire(t, statePath, first, stopPayload("a"))
	one, err := os.Stat(invaders.PausePath())
	if err != nil {
		t.Fatal(err)
	}

	fire(t, statePath, first.Add(90*time.Second), stopPayload("b"))
	two, err := os.Stat(invaders.PausePath())
	if err != nil {
		t.Fatal(err)
	}
	if !two.ModTime().After(one.ModTime()) {
		t.Errorf("the second Stop left the mtime at %v", two.ModTime())
	}
}

// A Stop with nothing usable in it must not fail the turn. Nothing a hook does
// is worth reporting an error to the person using Claude.
func TestAStopWithNoSessionStillWorksAndNeverFails(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	fire(t, statePath, now, map[string]any{"hook_event_name": "Stop"})
	if _, err := os.Stat(invaders.PausePath()); err != nil {
		t.Errorf("no pause file: %v", err)
	}
}
