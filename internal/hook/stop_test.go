package hook

import (
	"os"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/invaders"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
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

// The arena is off until somebody switches it on, and until then a prompt is a
// hook that reads one stat and returns. Nothing is opened, nothing is written -
// not the resume file, not the heartbeat, not the pet.
//
// It also writes nothing at all, which is structural rather than checked here:
// Run has no writer to print to. It matters for this event above all the others,
// because whatever a UserPromptSubmit hook puts on stdout is appended to the
// prompt and read as something the user said.
func TestAPromptWithTheArenaOffOpensNothing(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	before := pet.New()
	before.XP = 500
	if !pet.Save(before, statePath) {
		t.Fatal("the pet did not save")
	}
	was, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	fire(t, statePath, now, map[string]any{
		"hook_event_name": "UserPromptSubmit", "session_id": "smoke",
		"prompt": "vale, haz el plan",
	})

	for _, path := range []string{
		invaders.ResumePath(), invaders.LivePath(), invaders.PausePath(),
		session.PathFor("smoke", "window"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("the arena is off and a prompt wrote %s", path)
		}
	}
	is, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(is) != string(was) {
		t.Errorf("a prompt rewrote pet.json:\nwas %s\nis  %s", was, is)
	}
}

// The window a prompt remembered is this session's, and it is swept with the
// session's other scratch. Two sessions waiting on two answers go back to two
// different tabs, which is the whole reason it is not one file like the pause.
func TestTheWindowAPromptRememberedIsThisSessionsAndIsSweptWithIt(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	mine := session.PathFor("mine", "window")
	if !session.WriteAtomic(mine, []byte("0x4200006\n")) {
		t.Fatal("the window did not save")
	}
	if got := windowOf("mine"); got != "0x4200006" {
		t.Errorf("windowOf gave %q, want the window the prompt came from", got)
	}
	if got := windowOf("another-session"); got != "" {
		t.Errorf("windowOf gave %q for a session that never asked", got)
	}

	fire(t, statePath, now, map[string]any{
		"hook_event_name": "SessionEnd", "session_id": "mine",
	})
	if _, err := os.Stat(mine); err == nil {
		t.Error("the session ended and its window is still on disk")
	}
}
