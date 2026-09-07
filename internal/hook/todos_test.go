package hook

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// TodoWrite is the whole plan branch - the oracle, the cartographer and the
// sniper all read it - and it feeds the pet a task on every closed item. It
// had no test at all.

func todos(items ...[2]string) map[string]any {
	list := make([]any, 0, len(items))
	for _, it := range items {
		list = append(list, map[string]any{"status": it[0], "content": it[1]})
	}
	return map[string]any{
		"hook_event_name": "PostToolUse",
		"tool_name":       "TodoWrite",
		"session_id":      "todos-test",
		"tool_input":      map[string]any{"todos": list},
	}
}

func setUpHookHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TMPDIR", t.TempDir())
	return filepath.Join(home, "pet.json")
}

func TestAPlanWrittenBeforeAnyCodeFeedsTheOracle(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)

	fire(t, statePath, now, todos(
		[2]string{"pending", "uno"}, [2]string{"pending", "dos"},
		[2]string{"pending", "tres"}, [2]string{"pending", "cuatro"},
	))

	if got := pet.Load(statePath).Counters["plans_before_code"]; got != 1 {
		t.Errorf("plans_before_code = %d, se esperaba 1", got)
	}
}

// It counts once per session, not once per TodoWrite: the same plan re-sent is
// the same plan.
func TestThePlanIsOnlyCountedOncePerSession(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)
	plan := todos(
		[2]string{"pending", "uno"}, [2]string{"pending", "dos"},
		[2]string{"pending", "tres"}, [2]string{"pending", "cuatro"},
	)
	fire(t, statePath, now, plan)
	fire(t, statePath, now, plan)
	if got := pet.Load(statePath).Counters["plans_before_code"]; got != 1 {
		t.Errorf("plans_before_code = %d tras dos envios, se esperaba 1", got)
	}
}

// And touching code first shuts the door, which is what "before code" means.
func TestEditingFirstShutsTheOracleOut(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)

	fire(t, statePath, now, map[string]any{
		"hook_event_name": "PostToolUse", "tool_name": "Edit",
		"session_id": "todos-test",
	})
	fire(t, statePath, now, todos(
		[2]string{"pending", "uno"}, [2]string{"pending", "dos"},
		[2]string{"pending", "tres"}, [2]string{"pending", "cuatro"},
	))

	if got := pet.Load(statePath).Counters["plans_before_code"]; got != 0 {
		t.Errorf("plans_before_code = %d despues de editar, se esperaba 0", got)
	}
}

func TestAPlanClosedInFullRecordsItsLength(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)

	fire(t, statePath, now, todos(
		[2]string{"completed", "uno"}, [2]string{"completed", "dos"},
		[2]string{"completed", "tres"},
	))

	s := pet.Load(statePath)
	if got := s.Counters["longest_plan"]; got != 3 {
		t.Errorf("longest_plan = %d, se esperaba 3", got)
	}
	// Closing tasks is a meal.
	if s.XP != pet.Foods["task"].XP {
		t.Errorf("xp = %d, se esperaba la comida de una tarea (%d)", s.XP, pet.Foods["task"].XP)
	}
}

// RecordMax, not Bump: a shorter plan afterwards must not shrink the record.
func TestAShorterPlanDoesNotShrinkTheRecord(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	statePath := setUpHookHome(t)

	fire(t, statePath, now, todos(
		[2]string{"completed", "1"}, [2]string{"completed", "2"},
		[2]string{"completed", "3"}, [2]string{"completed", "4"},
	))
	fire(t, statePath, now, todos([2]string{"completed", "1"}))

	if got := pet.Load(statePath).Counters["longest_plan"]; got != 4 {
		t.Errorf("longest_plan = %d tras un plan mas corto, se esperaba 4", got)
	}
}

// A hostile payload must not take the hook down: it runs on every tool call.
func TestTodoWriteSurvivesRubbish(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, input := range []any{
		nil,
		"not an object",
		map[string]any{"todos": "not a list"},
		map[string]any{"todos": []any{"not an object", 42, nil}},
		map[string]any{"todos": []any{map[string]any{"status": 7, "content": nil}}},
	} {
		statePath := setUpHookHome(t)
		fire(t, statePath, now, map[string]any{
			"hook_event_name": "PostToolUse", "tool_name": "TodoWrite",
			"session_id": "todos-test", "tool_input": input,
		})
	}
}

func TestToolsUsedIgnoresTodoWriteAndBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tools")
	if err := os.WriteFile(path,
		[]byte("Read\n\nTodoWrite\n  Edit  \nRead\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := toolsUsed(path)
	if len(got) != 2 || !got["Read"] || !got["Edit"] {
		t.Errorf("toolsUsed = %v, se esperaban solo Read y Edit", got)
	}
	if len(toolsUsed("")) != 0 || len(toolsUsed(path+".missing")) != 0 {
		t.Error("una ruta vacia o inexistente deberia dar un conjunto vacio")
	}
}

func TestTruncateCountsRunesNotBytes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		n    int
		want string
	}{
		{"hola", 10, "hola"},
		{"hola", 4, "hola"},
		{"hola", 2, "ho"},
		{"añadió cañón", 6, "añadió"}, // multibyte: 6 runes, not 6 bytes
		{"", 3, ""},
		{"abc", 0, ""},
	} {
		if got := truncate(tc.in, tc.n); got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, se esperaba %q", tc.in, tc.n, got, tc.want)
		}
	}
}
