package invaders

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tmux is a convenience and never a dependency, so its absence has to be a
// sentence the player can act on rather than an error they have to look up.
func TestSplitSaysSoWhenThereIsNoTmux(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("TMUX", "")

	var out, errOut bytes.Buffer
	if code := split(&out, &errOut); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "tmux") {
		t.Errorf("it does not mention tmux: %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "invade") {
		t.Errorf("it does not say what to do instead: %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("it wrote %q to stdout", out.String())
	}
}

// Splitting from outside tmux would create a detached session nobody ever sees,
// which looks exactly like the command having silently done nothing. So it
// insists on already being in one, even when tmux is installed.
func TestSplitRefusesFromOutsideTmux(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "tmux")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TMUX", "")

	var out, errOut bytes.Buffer
	if code := split(&out, &errOut); code != 1 {
		t.Errorf("with tmux installed but not running, exit %d, want 1", code)
	}
	if strings.TrimSpace(errOut.String()) == "" {
		t.Error("it refused without saying why")
	}
}

// And inside tmux it actually asks tmux to split. The fake records what it was
// called with, which is the only part worth asserting: whether tmux then does
// the right thing is tmux's business.
func TestSplitAsksTmuxForAPaneWhenItIsInOne(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "args")
	fake := filepath.Join(dir, "tmux")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + log + "\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")

	var out, errOut bytes.Buffer
	if code := split(&out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("tmux was never called: %v", err)
	}
	got := string(raw)
	for _, want := range []string{"split-window", "-h", "invade"} {
		if !strings.Contains(got, want) {
			t.Errorf("tmux was called with %q, which does not carry %q", got, want)
		}
	}
}

// It is one file and one guard line at the top of Run, and it has to stay that
// way: the moment it needs a change anywhere else it stops being the ten-line
// convenience it was agreed as.
func TestSplitStaysTenLinesAndOneGuard(t *testing.T) {
	if got := strings.Count(mustRead(t, "run.go"), "split("); got != 1 {
		t.Errorf("run.go mentions split %d times, want the one guard", got)
	}
	src := mustRead(t, "split.go")
	if strings.Contains(src, "pet.") || strings.Contains(src, "Game") {
		t.Error("split.go has grown into the game")
	}
}
