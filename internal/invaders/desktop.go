package invaders

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// The desk itself: the two commands that can move a window on X11, and the
// handful of terminal emulators that can be told to run one thing.
//
// Every one of them is looked up and not depended on. wmctrl and xprop are two
// packages nobody installs on purpose, Wayland answers neither, and macOS has
// neither the commands nor the window ids - so all of this degrades to doing
// nothing, which is the same as the arena being off. That is the only behaviour
// this layer promises.

// TermEnv names the emulator and the arguments the arena opens the game with,
// for a desktop this list has never heard of:
//
//	CCPET_ARENA_TERM="wezterm start --"
//
// The binary and `invade` are appended, so whatever is named has to take a
// command as its trailing arguments.
const TermEnv = "CCPET_ARENA_TERM"

// terminals is how each emulator is asked to run one command, in the order they
// are tried. The command goes last in every one of them, which is what
// TestEveryTerminalWeKnowHowToOpenTakesTheCommandLast is for: an emulator whose
// flag swallowed the binary instead would open an empty shell, and an empty
// shell looks exactly like the arena having silently done nothing.
var terminals = []struct {
	Name string
	Args []string
}{
	{"gnome-terminal", []string{"--window", "--geometry=" + arenaGeometry, "--"}},
	{"konsole", []string{"--separate", "-e"}},
	{"xfce4-terminal", []string{"--geometry=" + arenaGeometry, "-x"}},
	{"kitty", nil},
	{"alacritty", []string{"-e"}},
	{"wezterm", []string{"start", "--"}},
	{"foot", nil},
	{"xterm", []string{"-geometry", arenaGeometry, "-e"}},
	// Debian's alternative, last: it is whatever the machine has, which is
	// usually one of the above under another name.
	{"x-terminal-emulator", []string{"-e"}},
}

// x11 is the real desk.
func x11() desktop {
	return desktop{
		Active: activeWindow,
		Title:  windowTitle,
		Raise:  func(title string) bool { return run("wmctrl", "-a", title) },
		Focus:  func(window string) bool { return run("wmctrl", "-i", "-a", window) },
		Open:   openArena,
	}
}

// activeWindow asks X which window has the focus. It is the window the prompt
// was just typed in, because typing is what gives a window the focus.
func activeWindow() string {
	out, err := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		return ""
	}
	return windowID(string(out))
}

// windowID reads the id out of xprop's line:
//
//	_NET_ACTIVE_WINDOW(WINDOW): window id # 0x4200006
//
// An id of 0x0 is xprop's way of saying nothing has the focus, and a line
// without a # at all is xprop saying the property is not there - a bare X server
// with no window manager. Both are no window.
func windowID(out string) string {
	_, rest, ok := strings.Cut(out, "#")
	if !ok {
		return ""
	}
	id := strings.TrimSpace(rest)
	if i := strings.IndexAny(id, ", \t\r\n"); i >= 0 {
		id = id[:i]
	}
	if id == "" || id == "0x0" || !strings.HasPrefix(id, "0x") {
		return ""
	}
	return id
}

// windowTitle is a window's name, or "" if it cannot be read.
func windowTitle(window string) string {
	if window == "" {
		return ""
	}
	out, err := exec.Command("xprop", "-id", window, "WM_NAME").Output()
	if err != nil {
		return ""
	}
	return titleOf(string(out))
}

// titleOf reads the name out of xprop's line, which comes as either
//
//	WM_NAME(STRING) = "ccpet invade"
//	WM_NAME(UTF8_STRING) = "ccpet invade"
//
// and, for a window that has no name yet, as a line with no quotes in it.
func titleOf(out string) string {
	i, j := strings.Index(out, `"`), strings.LastIndex(out, `"`)
	if i < 0 || j <= i {
		return ""
	}
	return out[i+1 : j]
}

// openArena starts the game in a terminal of its own.
//
// It does not wait, and the child gets no stdin, stdout or stderr: this runs
// inside a hook, and a hook that holds its pipe open until the game is over is a
// turn that never starts.
func openArena() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	name, args, err := terminalFor(os.Getenv(TermEnv))
	if err != nil {
		return err
	}
	cmd := exec.Command(name, append(args, exe, "invade")...)
	return cmd.Start()
}

// terminalFor picks the emulator: the one named by hand if there is one, and
// otherwise the first of the table that is installed.
func terminalFor(named string) (string, []string, error) {
	if fields := strings.Fields(named); len(fields) > 0 {
		if _, err := exec.LookPath(fields[0]); err != nil {
			return "", nil, err
		}
		return fields[0], fields[1:], nil
	}
	for _, t := range terminals {
		if _, err := exec.LookPath(t.Name); err == nil {
			return t.Name, t.Args, nil
		}
	}
	return "", nil, errors.New("no terminal emulator to open the game in")
}

// run is a command whose output nobody wants and whose failure is not an error,
// only a no.
func run(name string, args ...string) bool {
	if _, err := exec.LookPath(name); err != nil {
		return false
	}
	return exec.Command(name, args...).Run() == nil
}

// CanOpenAWindow says whether there is anything on this machine the arena could
// open the game in. It is asked by `ccpet arena on` so the answer arrives when
// the switch is thrown rather than as turn after turn of nothing happening.
func CanOpenAWindow() bool {
	_, _, err := terminalFor(os.Getenv(TermEnv))
	return err == nil
}
