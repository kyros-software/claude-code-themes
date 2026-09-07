package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// TestMain puts HOME somewhere disposable for EVERY test in this package.
//
// pet.Path() reads the config directory, and the dispatch reaches it through
// panel.Run and hook.Run without being handed a path - so a test that only
// isolates CLAUDE_CONFIG_DIR still writes to the real ~/.claude/pet.json. One
// did: a test that swept the usage text ran `ccpet day --dry` against the
// author's own pet and left a counter called "--dry" in it. Setting it here
// rather than per-test means the next test cannot forget.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ccpet-test-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", dir)
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(dir, ".claude"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// And a test that says so, because a TestMain is easy to delete by accident.
//
// It checks where pet.Path LANDS and not what HOME holds: on Linux
// os.UserHomeDir IS $HOME, so comparing the two only ever compares a value
// with itself.
func TestTheTestHomeIsNotTheRealOne(t *testing.T) {
	if !strings.Contains(pet.Path(), "ccpet-test-home-") {
		t.Fatalf("pet.Path() is %q: these tests would write to a live pet.json", pet.Path())
	}
}

func call(t *testing.T, argv []string, stdin string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(argv, strings.NewReader(stdin), &o, &e, time.Now())
	return code, o.String(), e.String()
}

// The dispatch, which had no test at all: this package was the only one in the
// repo at 0% coverage, and it is the one deciding which half of the program
// runs.
func TestDispatch(t *testing.T) {
	t.Run("version prints the stamped version", func(t *testing.T) {
		for _, flag := range []string{"version", "--version", "-v"} {
			code, out, _ := call(t, []string{"ccpet", flag}, "")
			if code != 0 {
				t.Errorf("%s: exit %d", flag, code)
			}
			if strings.TrimSpace(out) != version {
				t.Errorf("%s: printed %q, want %q", flag, strings.TrimSpace(out), version)
			}
		}
	})

	t.Run("help prints the usage", func(t *testing.T) {
		for _, flag := range []string{"-h", "--help", "help"} {
			code, out, _ := call(t, []string{"ccpet", flag}, "")
			if code != 0 || !strings.Contains(out, "ccpet statusline") {
				t.Errorf("%s: exit %d, out %q", flag, code, out)
			}
		}
	})

	t.Run("statusline renders from stdin", func(t *testing.T) {
		code, out, _ := call(t, []string{"ccpet", "statusline"},
			`{"model":{"display_name":"Opus 5"},"workspace":{"current_dir":"/"}}`)
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(out, "Opus 5") {
			t.Errorf("the model is not in the output: %q", out)
		}
	})

	// The symlink path: settings.json points at ~/.claude/ccpet-statusline, a
	// link to this binary, so argv[0] is the whole command. A bare path with no
	// argument is the shape that works whether or not the host uses a shell.
	t.Run("argv0 carrying statusline needs no argument", func(t *testing.T) {
		code, out, _ := call(t, []string{"/home/u/.claude/ccpet-statusline"},
			`{"model":{"display_name":"Opus 5"},"workspace":{"current_dir":"/"}}`)
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(out, "Opus 5") {
			t.Errorf("the symlink path did not render: %q", out)
		}
	})

	t.Run("an unreadable hook payload is not an error", func(t *testing.T) {
		if code, _, _ := call(t, []string{"ccpet", "hook"}, "not json"); code != 0 {
			t.Errorf("exit %d, want 0: a broken payload must never fail a tool call", code)
		}
	})
}

func TestSetupDispatch(t *testing.T) {
	t.Run("status reports on a config dir of its own", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		code, out, _ := call(t, []string{"ccpet", "setup", "status"}, "")
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		if !strings.Contains(out, "statusline:") {
			t.Errorf("no report: %q", out)
		}
	})

	t.Run("an unknown action is a usage error", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		code, _, errOut := call(t, []string{"ccpet", "setup", "nonsense"}, "")
		if code != 2 {
			t.Errorf("exit %d, want 2", code)
		}
		if !strings.Contains(errOut, "uso:") {
			t.Errorf("no usage on stderr: %q", errOut)
		}
	})

	t.Run("on then off leaves settings.json without the key", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		if code, _, e := call(t, []string{"ccpet", "setup", "on", "/root"}, ""); code != 0 {
			t.Fatalf("on: exit %d %q", code, e)
		}
		code, out, _ := call(t, []string{"ccpet", "setup", "status"}, "")
		if code != 0 || strings.Contains(out, "statusline: apagada") {
			t.Errorf("on did not take: %q", out)
		}
		if code, _, e := call(t, []string{"ccpet", "setup", "off"}, ""); code != 0 {
			t.Fatalf("off: exit %d %q", code, e)
		}
		_, out, _ = call(t, []string{"ccpet", "setup", "status"}, "")
		if !strings.Contains(out, "statusline: apagada") {
			t.Errorf("off did not take: %q", out)
		}
	})
}

// Every verb the dispatch answers has to be documented, and the usage is the
// only documentation there is. Checked in this direction rather than the other
// because the usage is prose - it has a title line and descriptions in Spanish,
// and scraping verbs out of it finds words, not commands.
func TestEveryDispatchedVerbIsDocumented(t *testing.T) {
	verbs := []string{
		"statusline", "hook", "setup", "link", "version", "help",
		"feed", "count", "day", "record", "session",
	}
	for _, verb := range verbs {
		if !strings.Contains(usage, "ccpet "+verb) {
			t.Errorf("the dispatch answers %q and the usage never mentions it", verb)
		}
	}
}

// And the meals the panel accepts are the ones the usage lists.
func TestTheMealsInTheUsageAreTheMealsThatExist(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	for _, meal := range []string{"tests", "commit", "compact", "task", "overflow"} {
		if !strings.Contains(usage, meal) {
			t.Errorf("the meal %q is not in the usage", meal)
		}
	}
}

func TestLinkAndPanel(t *testing.T) {
	t.Run("link reports where it pointed", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		if code, _, e := call(t, []string{"ccpet", "link", t.TempDir()}, ""); code != 0 {
			t.Errorf("exit %d: %s", code, e)
		}
	})

	t.Run("no arguments draws the panel", func(t *testing.T) {
		code, out, e := call(t, []string{"ccpet"}, "")
		if code != 0 {
			t.Fatalf("exit %d: %s", code, e)
		}
		if strings.TrimSpace(out) == "" {
			t.Error("the panel printed nothing")
		}
	})

	t.Run("something that is not a meal says so", func(t *testing.T) {
		code, _, errOut := call(t, []string{"ccpet", "bananas"}, "")
		if code != 2 {
			t.Errorf("exit %d, want 2", code)
		}
		if !strings.Contains(errOut, "no es comida") {
			t.Errorf("stderr: %q", errOut)
		}
	})

	t.Run("a meal is eaten", func(t *testing.T) {
		if code, _, e := call(t, []string{"ccpet", "commit"}, ""); code != 0 {
			t.Errorf("exit %d: %s", code, e)
		}
	})
}
