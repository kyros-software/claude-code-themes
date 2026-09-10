package invaders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// xprop's answer, verbatim from the machine this was written on.
func TestTheActiveWindowIsTheIdXpropPrints(t *testing.T) {
	for _, c := range []struct {
		out  string
		want string
	}{
		{"_NET_ACTIVE_WINDOW(WINDOW): window id # 0x4200006\n", "0x4200006"},
		{"_NET_ACTIVE_WINDOW(WINDOW): window id # 0x4200006, 0x0\n", "0x4200006"},
		// Nothing has the focus, or there is no window manager to say so.
		{"_NET_ACTIVE_WINDOW(WINDOW): window id # 0x0\n", ""},
		{"_NET_ACTIVE_WINDOW:  not found.\n", ""},
		{"", ""},
		// Not an id, and so not something to hand to wmctrl.
		{"_NET_ACTIVE_WINDOW(WINDOW): window id # nonsense\n", ""},
	} {
		if got := windowID(c.out); got != c.want {
			t.Errorf("windowID(%q) = %q, want %q", c.out, got, c.want)
		}
	}
}

// The title is read for one reason: telling the game's own window apart from a
// window worth handing the focus back to. It arrives as STRING on some servers
// and UTF8_STRING on others, and a window with no name yet has no quotes at all.
func TestTheTitleIsWhatIsInsideTheQuotes(t *testing.T) {
	for _, c := range []struct {
		out  string
		want string
	}{
		{`WM_NAME(STRING) = "ccpet invade"` + "\n", "ccpet invade"},
		{`WM_NAME(UTF8_STRING) = "ccpet invade"` + "\n", "ccpet invade"},
		// Seen on the machine this was written on, for a gnome-terminal window
		// whose title has an asterisk in it.
		{`WM_NAME(COMPOUND_TEXT) = "ccpet invade"` + "\n", "ccpet invade"},
		{`WM_NAME(UTF8_STRING) = "a tab called "invade" for short"` + "\n", `a tab called "invade" for short`},
		{"WM_NAME:  not found.\n", ""},
		{"", ""},
	} {
		if got := titleOf(c.out); got != c.want {
			t.Errorf("titleOf(%q) = %q, want %q", c.out, got, c.want)
		}
	}
}

// A machine with no emulator on it says so rather than pretending it opened
// something: `ccpet arena on` is where that is worth hearing, because the
// alternative is turn after turn of nothing happening.
func TestWithNoEmulatorAnywhereTheArenaSaysSo(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv(TermEnv, "")
	if CanOpenAWindow() {
		t.Error("an empty PATH still claims it can open a window")
	}
	if err := openArena(); err == nil {
		t.Error("openArena with no emulator reported success")
	}
}

// The emulator can be named by hand, for a desktop the table has never heard of.
// A name that is not installed is a refusal and not a command that fails later.
func TestTheEmulatorCanBeNamedByHand(t *testing.T) {
	dir := t.TempDir()
	fakeTerminal(t, dir, "myterm")
	t.Setenv("PATH", dir)

	t.Setenv(TermEnv, "myterm --new-window -e")
	name, args, err := terminalFor(os.Getenv(TermEnv))
	if err != nil {
		t.Fatal(err)
	}
	if name != "myterm" || strings.Join(args, " ") != "--new-window -e" {
		t.Errorf("terminalFor gave %q %v, want myterm with its own flags", name, args)
	}

	t.Setenv(TermEnv, "a-terminal-nobody-has")
	if _, _, err := terminalFor(os.Getenv(TermEnv)); err == nil {
		t.Error("a named emulator that is not installed was accepted")
	}
}

// What actually gets run, argv and all: the emulator, its flags, this binary and
// `invade` LAST. An emulator whose flag swallowed the binary would open an empty
// shell, and an empty shell looks exactly like the arena having done nothing.
func TestTheEmulatorIsHandedTheGameAsItsLastArguments(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "argv")
	fakeTerminal(t, dir, "gnome-terminal")
	t.Setenv("PATH", dir)
	t.Setenv(TermEnv, "")
	t.Setenv("CCPET_FAKE_TERM_OUT", out)

	if err := openArena(); err != nil {
		t.Fatal(err)
	}

	argv := waitForFile(t, out)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if len(argv) < 2 || argv[len(argv)-2] != exe || argv[len(argv)-1] != "invade" {
		t.Fatalf("the emulator was handed %v, want it ending in %q invade", argv, exe)
	}
	if !strings.Contains(strings.Join(argv, " "), "--geometry="+arenaGeometry) {
		t.Errorf("the window was asked for as %v, with no geometry in it", argv)
	}
}

// fakeTerminal writes an emulator that records its arguments instead of opening
// a window, which is the only kind this suite is allowed to run.
func fakeTerminal(t *testing.T, dir, name string) {
	t.Helper()
	script := "#!/bin/sh\nif [ -n \"$CCPET_FAKE_TERM_OUT\" ]; then printf '%s\\n' \"$@\" > \"$CCPET_FAKE_TERM_OUT\"; fi\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// waitForFile is the seam where a test meets a process that was deliberately not
// waited for: openArena starts the emulator and returns, because it runs inside a
// hook and a hook that waits for the game to end is a turn that never starts.
func waitForFile(t *testing.T, path string) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		raw, err := os.ReadFile(path)
		if err == nil && len(raw) > 0 {
			return strings.Fields(strings.TrimSpace(string(raw)))
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never appeared: the emulator was not run", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
