package statusline

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/session"
)

// leftover is a scratch file from a session that died without a SessionEnd,
// aged past the sweep's cutoff.
func leftover(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, session.Prefix+name)
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-session.MaxAge - time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return path
}

func refresh(t *testing.T, sessionID, promptID, apiMs string) {
	t.Helper()
	payload := `{"model":{"display_name":"Opus 5"},` +
		`"context_window":{"used_percentage":30},` +
		`"cost":{"total_api_duration_ms":` + apiMs + `},` +
		`"session_id":"` + sessionID + `","prompt_id":"` + promptID + `"}`
	if err := Run(strings.NewReader(payload), io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestTheSweepRunsOncePerSessionAndNotOncePerRefresh(t *testing.T) {
	// The sweep is a ReadDir of the whole of $TMPDIR. It used to hang off the
	// same condition as saving the facts, and one of the things in that
	// condition is total_api_duration_ms - which moves under every refresh
	// while the model is answering. Collecting day-old leftovers does not need
	// to happen once a second.
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), ".claude"))
	t.Setenv("COLUMNS", "120")
	t.Setenv("STATUSLINE_PET", "1")

	first := leftover(t, tmp, "dead-one")
	refresh(t, "sweeptest", "p1", "1000")
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Error("the first refresh of a session did not sweep")
	}

	// Same session, a refresh that changes something: the facts get saved
	// again, and this time the sweep must NOT come along for the ride.
	second := leftover(t, tmp, "dead-two")
	refresh(t, "sweeptest", "p2", "2000")
	if _, err := os.Stat(second); err != nil {
		t.Error("a later refresh swept again")
	}

	// A different session starts, and it sweeps for itself.
	refresh(t, "othersession", "p1", "1000")
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Error("a new session did not sweep")
	}
}
