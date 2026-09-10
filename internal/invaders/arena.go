package invaders

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/config"
	"github.com/kyros-software/claude-code-themes/internal/lockfile"
)

// The arena: the game plays itself around your turns.
//
// Waiting for an answer is the one thing this theme never had an answer for. With
// the arena on, submitting a prompt opens the game in a window of its own and
// hands it the keyboard; the Stop that ends the turn pauses the game and gives
// the focus back to the tab you typed in. You play while Claude works and you
// stop where you were when it is done.
//
// Three rules, and they are what the tests below are about:
//
//   - ONE game, however many Claudes you have open. The running game keeps a
//     heartbeat file fresh; every session's prompt looks at it, and the one that
//     finds it stale is the only one that opens a window.
//   - The game is never opened, raised or paused unless the arena is switched
//     on. It is off until `ccpet arena on`, because a theme that starts opening
//     terminal windows on its own is a theme somebody uninstalls.
//   - Nothing here is ever an error the turn can see. No window manager, no X, a
//     terminal emulator nobody has heard of: the arena does nothing at all and
//     the prompt goes through exactly as it did before.

const (
	// arenaTitle is what the game calls its window, and so the handle the arena
	// raises it by. A title and not a pid: under gnome-terminal every window
	// belongs to one server process, so a pid identifies the emulator and not
	// the window, and the game is a grandchild of it either way.
	arenaTitle = "ccpet invade"

	// arenaGeometry is the window the game asks for, where the emulator takes a
	// geometry at all. Comfortably over the 60x18 floor: a window opened at the
	// minimum is a window that refuses to draw the moment a font is a point
	// bigger than the one this was measured with.
	arenaGeometry = "100x30"

	// liveFor is how stale the heartbeat may be before the game counts as gone.
	// Three beats: one missed beat is a busy machine, three is a window that has
	// been closed.
	liveFor = 3 * time.Second

	// beatEvery is how often the running game says it is still there.
	beatEvery = time.Second
)

// ArenaPath is the switch. A file whose existence is the setting, like the two
// signals beside it, and not a key in ccpet.json: this is read by a hook on
// every prompt of every session and written by hand at most twice, so a stat
// beats a read-parse-merge-rename, and it can never lose somebody's language
// setting to a botched write.
func ArenaPath() string { return filepath.Join(config.Dir(), "ccpet-arena") }

// ArenaOn says whether the game is allowed to open itself.
func ArenaOn() bool {
	_, err := os.Stat(ArenaPath())
	return err == nil
}

// SetArena writes the switch, or takes it away.
func SetArena(on bool) error {
	path := ArenaPath()
	if !on {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// LivePath is the heartbeat: mtime says when the game last drew a frame, and the
// contents say which process is drawing them.
func LivePath() string { return filepath.Join(config.Dir(), "ccpet-invade.live") }

// Beat says the game is still running, and whose game it is.
//
// The pid is written every beat rather than once at the start because the file
// is claimed by whoever opens the window - see enter - and only the game itself
// can say the claim has been made good.
func Beat(now time.Time) error {
	path := LivePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chtimes(path, now, now)
}

// Live says a game is running somewhere - in this terminal, in another tab, for
// another session. It is the whole of "only one game, however many Claudes".
func Live(now time.Time) bool {
	at, ok := mtime(LivePath())
	if !ok {
		return false
	}
	return now.Sub(at) < liveFor
}

// Unbeat gives the slot back the moment the game exits, so the next prompt opens
// a window instead of waiting out the heartbeat.
//
// Only ever its own claim: a second game started by hand while the arena's is
// running must not free a slot it does not hold. Without the check, quitting the
// hand-started one would leave the arena convinced nothing is playing and open a
// second window on the next prompt.
func Unbeat() {
	raw, err := os.ReadFile(LivePath())
	if err != nil {
		return
	}
	if strings.TrimSpace(string(raw)) != strconv.Itoa(os.Getpid()) {
		return
	}
	os.Remove(LivePath())
}

// desktop is everything the arena needs from a window manager, which is four
// small questions and one door.
//
// A struct of functions and not an interface with a mock, for the reason the
// screen in run.go gives: the fake is a dozen lines in a test, and this way the
// policy below can be tested without a display, without an emulator and without
// a window ever appearing on somebody's screen while the suite runs.
type desktop struct {
	// Active is the window with the focus right now, "" if that cannot be told.
	Active func() string
	// Title is a window's name, used for exactly one thing: telling the game's
	// own window apart from a window worth handing the focus back to.
	Title func(window string) string
	// Raise brings the window with this title forward.
	Raise func(title string) bool
	// Focus brings this exact window forward.
	Focus func(window string) bool
	// Open starts the game in a terminal of its own.
	Open func() error
}

// Enter is a turn starting: the game opens if it is not open, comes forward, and
// comes out of the pause the last turn left it in.
//
// It returns the window the prompt was typed in, which is the window the focus
// goes back to when the turn ends. Reading it is the first thing that happens
// here and it has to be: the very next step raises the game over it.
func Enter(now time.Time) string { return enterArena(x11(), now) }

func enterArena(d desktop, now time.Time) string {
	if !ArenaOn() {
		return ""
	}
	home := d.Active()
	if home != "" && d.Title(home) == arenaTitle {
		// The prompt came from the game's own window, which cannot happen by
		// typing and so is not a window to hand anything back to.
		home = ""
	}

	// Whatever else happens, a game that is paused should be playing: the wait
	// has started again.
	Resume(now)

	// The claim is taken under the lock and made before the window is opened,
	// not after it. Two sessions answering the same instant would otherwise both
	// look, both see nothing running, and both open a window.
	release, _ := lockfile.Take(LivePath())
	live := Live(now)
	if !live {
		Beat(now)
	}
	release()

	if live {
		d.Raise(arenaTitle)
		return home
	}
	if err := d.Open(); err != nil {
		// The claim was this process's and the window never came: drop it rather
		// than make the next three seconds of prompts think a game is running.
		Unbeat()
	}
	return home
}

// GiveBack is the turn ending: the focus goes back to the tab the prompt was
// typed in. The pause itself is Touch, called by the hook whether the arena is
// on or not - a game in a tmux pane you opened by hand still wants stopping.
func GiveBack(window string) { giveBackArena(x11(), window) }

func giveBackArena(d desktop, window string) {
	if !ArenaOn() || window == "" {
		return
	}
	d.Focus(window)
}
