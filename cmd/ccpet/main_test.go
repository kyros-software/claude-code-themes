package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/invaders"
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
	// And the language, for the same reason: these tests assert on wording,
	// and a developer with CCPET_LANG=en in their shell would fail every one
	// of them for a reason that has nothing to do with the code.
	os.Setenv(i18n.Env, string(i18n.ES))
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
// In BOTH languages: a verb documented in one help and forgotten in the other
// is a verb half the users cannot find.
func TestEveryDispatchedVerbIsDocumented(t *testing.T) {
	verbs := []string{
		"statusline", "hook", "setup", "link", "version", "help", "lang",
		"feed", "count", "day", "record", "session", "invade", "arena",
	}
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		usage := i18n.S().Usage
		for _, verb := range verbs {
			if !strings.Contains(usage, "ccpet "+verb) {
				t.Errorf("%s: the dispatch answers %q and the usage never mentions it", lang, verb)
			}
		}
	}
	i18n.Use("")
}

// And the meals the panel accepts are the ones the usage lists.
func TestTheMealsInTheUsageAreTheMealsThatExist(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, meal := range []string{"tests", "commit", "compact", "task", "overflow"} {
			if !strings.Contains(i18n.S().Usage, meal) {
				t.Errorf("%s: the meal %q is not in the usage", lang, meal)
			}
		}
	}
	i18n.Use("")
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

// --- the language setting ---------------------------------------------------

// `ccpet lang <x>` writes it down, and the confirmation is already in the
// language just chosen: the setting has to be believed by the process that
// made it, or the first thing a person sees after switching contradicts it.
func TestLangWritesTheSettingAndAnswersInIt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	t.Setenv(i18n.Env, "")
	defer i18n.Use("")

	code, out, errOut := call(t, []string{"ccpet", "lang", "en"}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if !strings.Contains(out, "language: en") {
		t.Errorf("the confirmation is %q, and it is not in English", strings.TrimSpace(out))
	}
	raw, err := os.ReadFile(filepath.Join(dir, "ccpet.json"))
	if err != nil {
		t.Fatalf("nothing was written: %v", err)
	}
	if !strings.Contains(string(raw), `"lang": "en"`) {
		t.Errorf("ccpet.json holds %s", raw)
	}

	// And the help that follows is English too, from the same file.
	i18n.Use("")
	if _, out, _ := call(t, []string{"ccpet", "help"}, ""); !strings.Contains(out, "the pet's panel") {
		t.Errorf("the help after switching is still Spanish:\n%s", out)
	}
}

// A language the theme does not speak is refused rather than stored: a
// silently ignored `ccpet lang fr` leaves a person believing it took.
func TestLangRefusesALanguageItDoesNotSpeak(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv(i18n.Env, "")
	defer i18n.Use("")
	code, _, errOut := call(t, []string{"ccpet", "lang", "fr"}, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut, "ccpet lang") {
		t.Errorf("it said %q", strings.TrimSpace(errOut))
	}
}

// `--lang` is for one command and leaves the setting alone. It is also the
// only way to read the panel in the other language without switching twice.
func TestTheLangFlagDoesNotTouchTheSetting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	t.Setenv(i18n.Env, "")
	defer i18n.Use("")

	for _, argv := range [][]string{
		{"ccpet", "--lang", "en", "help"},
		{"ccpet", "--lang=en", "help"},
	} {
		i18n.Use("")
		code, out, _ := call(t, argv, "")
		if code != 0 || !strings.Contains(out, "the pet's panel") {
			t.Errorf("%v printed:\n%s", argv, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "ccpet.json")); err == nil {
		t.Error("--lang wrote the setting down; it is meant to be for one command")
	}
}

// invade needs a terminal, and CI has none. It has to say so and come straight
// back rather than blocking on a /dev/tty that is not there - the shape of hang
// that only ever shows up in CI, ten minutes into a timeout.
//
// Skipped where there IS a terminal, and that is not shyness: invade takes the
// controlling terminal, hides the cursor and reads keys, so running it for real
// from `go test` on a developer's machine would swallow their shell until they
// pressed q. The path this test is about only exists where there is nothing to
// take, which is exactly where CI runs.
func TestInvadeWithoutATerminalSaysSoAndDoesNotHang(t *testing.T) {
	if f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		f.Close()
		t.Skip("there is a terminal here; invade would take it over")
	}
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	done := make(chan struct{})
	var code int
	var errOut string
	go func() {
		defer close(done)
		code, _, errOut = call(t, []string{"ccpet", "invade"}, "")
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("invade blocked instead of refusing")
	}
	if code != 2 {
		t.Errorf("exit %d, want 2: the environment is wrong, not the program", code)
	}
	if strings.TrimSpace(errOut) == "" {
		t.Error("it refused and said nothing about why")
	}
}

// The arena is a program that opens windows on somebody's desktop, so it is off
// until it is switched on, it says which it is whenever it is asked, and it
// refuses a word it does not know rather than guessing between on and off.
func TestTheArenaSwitchSaysWhichItIs(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	// No emulator anywhere, which makes the warning below deterministic on a
	// machine with a desktop and on a CI runner without one alike.
	t.Setenv("PATH", t.TempDir())

	code, out, _ := call(t, []string{"ccpet", "--lang", "en", "arena"}, "")
	if code != 0 || !strings.Contains(out, "arena: off") {
		t.Errorf("asking exited %d saying %q, want it off", code, out)
	}

	code, out, errOut := call(t, []string{"ccpet", "--lang", "en", "arena", "on"}, "")
	if code != 0 || !strings.Contains(out, "arena: on") {
		t.Errorf("switching it on exited %d saying %q", code, out)
	}
	if !strings.Contains(errOut, "no terminal emulator") {
		t.Errorf("there is nothing here to open a window with and it said %q", errOut)
	}
	if !invaders.ArenaOn() {
		t.Error("the switch is on and the arena disagrees")
	}

	if code, out, _ := call(t, []string{"ccpet", "--lang", "en", "arena"}, ""); !strings.Contains(out, "arena: on") {
		t.Errorf("asking again exited %d saying %q, want it still on", code, out)
	}

	if code, _, _ := call(t, []string{"ccpet", "--lang", "en", "arena", "off"}, ""); code != 0 {
		t.Errorf("switching it off exited %d", code)
	}
	if invaders.ArenaOn() {
		t.Error("the switch is off and the arena is still on")
	}

	code, _, errOut = call(t, []string{"ccpet", "--lang", "en", "arena", "maybe"}, "")
	if code != 2 || !strings.Contains(errOut, "usage:") {
		t.Errorf("a word it does not know exited %d saying %q, want a usage error", code, errOut)
	}
}
