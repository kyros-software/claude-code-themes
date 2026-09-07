package panel

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

func write(t *testing.T, s *pet.State) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pet.json")
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func render(t *testing.T, s *pet.State, now time.Time) []string {
	t.Helper()
	var out bytes.Buffer
	if code := showPanel(&out, write(t, s), now); code != 0 {
		t.Fatalf("panel exited %d", code)
	}
	lines := strings.Split(theme.Strip(out.String()), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

func todayLog(t *testing.T, lines []string) []string {
	t.Helper()
	var out []string
	seen := false
	for _, l := range lines {
		if strings.TrimSpace(l) == "hoy" {
			seen = true
			continue
		}
		if seen && strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func TestTheNoteColumnDisappearsWhenNobodyUsesIt(t *testing.T) {
	// A green suite carries no note, and a day of them was spending
	// twenty-two painted columns on a gap.
	now := time.Date(2026, 3, 1, 13, 30, 0, 0, time.UTC)
	s := pet.New()
	s.XP, s.LogDay = 300, pet.Today(now)
	for i := 0; i < 3; i++ {
		s.Log = append(s.Log, pet.LogEntry{
			Event: "tests", XP: 15, At: now.Add(-time.Duration(i) * time.Minute).Unix()})
	}

	rows := todayLog(t, render(t, s, now))
	if len(rows) != 3 {
		t.Fatalf("got %d log rows, want 3: %q", len(rows), rows)
	}
	for _, r := range rows {
		// Past the indent there should be no run of blanks left: the widest
		// gap a full row needs is the two after the XP.
		body := strings.TrimLeft(r, " ")
		if strings.Contains(body, "   ") {
			t.Errorf("the row still carries an empty column: %q", r)
		}
		if n := len([]rune(r)); n > 40 {
			t.Errorf("the row is %d columns for %q", n, "tests en verde")
		}
	}
}

func TestTheNoteColumnComesBackForWhoeverFillsIt(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 30, 0, 0, time.UTC)
	s := pet.New()
	s.XP, s.LogDay = 300, pet.Today(now)
	s.Log = []pet.LogEntry{
		{Event: "tests", XP: 15, At: now.Add(-10 * time.Minute).Unix()},
		{Event: "commit", XP: 12, Note: "a1b2c3d arregla el pie",
			At: now.Add(-5 * time.Minute).Unix()},
	}

	rows := todayLog(t, render(t, s, now))
	if len(rows) != 2 {
		t.Fatalf("got %d log rows, want 2", len(rows))
	}
	if !strings.Contains(rows[1], "a1b2c3d arregla el pie") {
		t.Errorf("the note is missing: %q", rows[1])
	}
	// The clocks still line up, which is what the padding is for.
	if a, b := strings.LastIndex(rows[0], "13:"), strings.LastIndex(rows[1], "13:"); a != b {
		t.Errorf("the times fell out of column: %d vs %d\n%q\n%q", a, b, rows[0], rows[1])
	}
}

func TestALongNoteIsCutNotWrapped(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 30, 0, 0, time.UTC)
	s := pet.New()
	s.XP, s.LogDay = 300, pet.Today(now)
	s.Log = []pet.LogEntry{{Event: "commit", XP: 12, At: now.Unix(),
		Note: strings.Repeat("x", 80)}}

	rows := todayLog(t, render(t, s, now))
	if len(rows) != 1 {
		t.Fatalf("a long note broke the row into %d", len(rows))
	}
	if n := len([]rune(rows[0])); n > 60 {
		t.Errorf("the row came out %d columns wide: %q", n, rows[0])
	}
}
