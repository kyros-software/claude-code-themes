package hook

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// fire sends one hook event the way the CLI does, and hands back the pet.
func fire(t *testing.T, statePath string, now time.Time, payload map[string]any) *pet.State {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if code := Run(bytes.NewReader(raw), statePath, now); code != 0 {
		t.Fatalf("hook exited %d", code)
	}
	return pet.Load(statePath)
}

func greenSuite(session string) map[string]any {
	return map[string]any{
		"hook_event_name": "PostToolUse", "tool_name": "Bash",
		"session_id": session, "cwd": ".",
		"tool_input":    map[string]any{"command": "go test ./..."},
		"tool_response": map[string]any{"stdout": "ok  example 0.01s", "is_error": false},
	}
}

func anEdit(session string) map[string]any {
	return map[string]any{
		"hook_event_name": "PostToolUse", "tool_name": "Edit",
		"session_id": session, "cwd": ".",
		"tool_input":    map[string]any{"file_path": "main.go"},
		"tool_response": map[string]any{"is_error": false},
	}
}

func TestAGreenSuiteWithNothingBehindItIsNotAMeal(t *testing.T) {
	// The measured failure: eight identical green suites, +120 xp, nine
	// seconds apart, none of them with a change behind it.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", t.TempDir())
	statePath := filepath.Join(home, "pet.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 8; i++ {
		s := fire(t, statePath, now.Add(time.Duration(i)*10*time.Second),
			greenSuite("farm-session"))
		if s.XP != 0 {
			t.Fatalf("suite %d paid out %d xp with no edit behind it", i+1, s.XP)
		}
	}
}

func TestAnEditThenAGreenSuiteIsAMeal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", t.TempDir())
	statePath := filepath.Join(home, "pet.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	fire(t, statePath, now, anEdit("work-session"))
	s := fire(t, statePath, now.Add(time.Minute), greenSuite("work-session"))
	if s.XP != pet.Foods["tests"].XP {
		t.Fatalf("a change followed by a green suite gave %d xp, want %d",
			s.XP, pet.Foods["tests"].XP)
	}

	// And it does not pay twice for the same change.
	s = fire(t, statePath, now.Add(2*time.Minute), greenSuite("work-session"))
	if s.XP != pet.Foods["tests"].XP {
		t.Errorf("the same change was paid for twice: %d xp", s.XP)
	}
}

func TestTheCooldownStillHoldsAcrossEdits(t *testing.T) {
	// Editing between runs must not be a way round the hour either.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", t.TempDir())
	statePath := filepath.Join(home, "pet.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	fire(t, statePath, now, anEdit("s"))
	first := fire(t, statePath, now.Add(time.Second), greenSuite("s"))

	for i := 1; i <= 5; i++ {
		at := now.Add(time.Duration(i) * 5 * time.Minute)
		fire(t, statePath, at, anEdit("s"))
		s := fire(t, statePath, at.Add(time.Second), greenSuite("s"))
		if s.XP != first.XP {
			t.Fatalf("edit+suite %d got through the cooldown: %d xp", i, s.XP)
		}
	}

	fire(t, statePath, now.Add(pet.TestsCooldown+time.Minute), anEdit("s"))
	s := fire(t, statePath, now.Add(pet.TestsCooldown+2*time.Minute), greenSuite("s"))
	if s.XP <= first.XP {
		t.Errorf("still refused an hour later: %d xp", s.XP)
	}
}

func TestTheReproCycleIsCreditedEvenWithoutAPayout(t *testing.T) {
	// A red then a green is the bloodhound's habit; it is worth recording
	// whether or not the suite itself feeds.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", t.TempDir())
	statePath := filepath.Join(home, "pet.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	red := greenSuite("repro")
	red["tool_response"] = map[string]any{"stdout": "FAIL example", "is_error": true}
	fire(t, statePath, now, red)

	s := fire(t, statePath, now.Add(time.Minute), greenSuite("repro"))
	if s.Counters["repro_before_fix"] != 1 {
		t.Errorf("repro_before_fix = %d, want 1", s.Counters["repro_before_fix"])
	}
	if s.XP != 0 {
		t.Errorf("it also paid out %d xp with no edit behind it", s.XP)
	}
}
