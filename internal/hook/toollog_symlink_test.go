//go:build !windows

package hook

import (
	"os"
	"path/filepath"
	"testing"
)

// The tool log lives in $TMPDIR. A symlink planted at its path must not turn
// an append into a write to wherever the link points.
func TestTheToolLogDoesNotWriteThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools")
	victim := filepath.Join(dir, "victim")
	if err := os.WriteFile(victim, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, path); err != nil {
		t.Skip("no symlinks here:", err)
	}
	logTool(path, "Bash")
	raw, err := os.ReadFile(victim)
	if err != nil || string(raw) != "mine\n" {
		t.Fatalf("the far end of the link was written to: %q %v", raw, err)
	}
}
