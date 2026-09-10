package invaders

import (
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// desk is a window manager that never opens a window: it counts what it was
// asked to do. Every arena test runs against this, because a suite that opened
// real terminal windows would be a suite nobody could run twice.
type desk struct {
	mu      sync.Mutex
	active  string
	titles  map[string]string
	opens   int
	raised  []string
	focused []string
	openErr error
}

func (d *desk) desktop() desktop {
	return desktop{
		Active: func() string {
			d.mu.Lock()
			defer d.mu.Unlock()
			return d.active
		},
		Title: func(window string) string {
			d.mu.Lock()
			defer d.mu.Unlock()
			return d.titles[window]
		},
		Raise: func(title string) bool {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.raised = append(d.raised, title)
			return true
		},
		Focus: func(window string) bool {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.focused = append(d.focused, window)
			return true
		},
		Open: func() error {
			d.mu.Lock()
			defer d.mu.Unlock()
			if d.openErr != nil {
				return d.openErr
			}
			d.opens++
			return nil
		},
	}
}

func (d *desk) counts() (opens int, raised, focused []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opens, append([]string(nil), d.raised...), append([]string(nil), d.focused...)
}

// arenaHome puts the switch and the three files somewhere disposable, and turns
// the arena on if asked. Every test in this file needs it: without it the first
// one would open a window on whoever ran the suite.
func arenaHome(t *testing.T, on bool) {
	t.Helper()
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	if on {
		if err := SetArena(true); err != nil {
			t.Fatal(err)
		}
	}
}

// The switch is the whole of the consent. A theme that starts opening terminal
// windows because it was installed is a theme somebody uninstalls, so with the
// arena off a prompt does nothing at all: no window, no raise, and not even the
// resume file, which is a write to somebody's ~/.claude that was never asked for.
func TestNothingOpensUntilTheArenaIsSwitchedOn(t *testing.T) {
	arenaHome(t, false)
	d := &desk{active: "0x1"}

	if home := enterArena(d.desktop(), time.Now()); home != "" {
		t.Errorf("the arena is off and it handed back the window %q", home)
	}
	opens, raised, _ := d.counts()
	if opens != 0 || len(raised) != 0 {
		t.Errorf("the arena is off and it opened %d windows and raised %v", opens, raised)
	}
	for _, path := range []string{ResumePath(), LivePath()} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("the arena is off and it wrote %s", path)
		}
	}
}

// One game, however many Claudes are open. The second session's prompt finds the
// heartbeat fresh and raises the window that is already playing.
func TestTheSecondSessionRaisesTheGameInsteadOfOpeningASecond(t *testing.T) {
	arenaHome(t, true)
	d := &desk{active: "0x1"}
	now := time.Now()

	enterArena(d.desktop(), now)
	enterArena(d.desktop(), now.Add(time.Second))

	opens, raised, _ := d.counts()
	if opens != 1 {
		t.Errorf("two prompts opened %d windows, want one", opens)
	}
	if len(raised) != 1 || raised[0] != arenaTitle {
		t.Errorf("the second prompt raised %v, want one %q", raised, arenaTitle)
	}
}

// And two sessions answering in the same instant are the reason the claim is
// taken under a lock before the window is opened rather than after it: both
// would otherwise look, both see nothing playing, and both open one.
func TestTwoSessionsAskingAtOnceStillOpenOneGame(t *testing.T) {
	arenaHome(t, true)
	d := &desk{active: "0x1"}
	now := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			enterArena(d.desktop(), now)
		}()
	}
	wg.Wait()

	if opens, _, _ := d.counts(); opens != 1 {
		t.Errorf("eight prompts at once opened %d windows, want one", opens)
	}
}

// A heartbeat nobody is keeping fresh is a window that has been closed, and the
// next prompt is entitled to a new one. Three seconds, so a busy machine that
// misses a beat does not lose its game.
func TestAGameThatIsGoneLetsTheNextPromptOpenAnother(t *testing.T) {
	arenaHome(t, true)
	now := time.Now()
	if err := Beat(now.Add(-10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if Live(now) {
		t.Fatal("a heartbeat from ten seconds ago counts as a game still playing")
	}

	d := &desk{active: "0x1"}
	enterArena(d.desktop(), now)
	if opens, _, _ := d.counts(); opens != 1 {
		t.Errorf("a stale heartbeat opened %d windows, want one", opens)
	}
}

// A window that never came must not hold the slot for three seconds: the claim
// was this process's and it goes back.
func TestAWindowThatFailedToOpenDoesNotHoldTheSlot(t *testing.T) {
	arenaHome(t, true)
	d := &desk{active: "0x1", openErr: errors.New("no terminal here")}
	now := time.Now()

	enterArena(d.desktop(), now)
	if Live(now) {
		t.Error("the window failed to open and the arena still thinks a game is playing")
	}
}

// Every turn is a wait, so every prompt has to take the game back out of the
// pause the last Stop left it in - including the prompts that only raise a window
// that was already open.
func TestEveryPromptWakesAGameThatWasPaused(t *testing.T) {
	arenaHome(t, true)
	d := &desk{active: "0x1"}
	now := time.Now()

	enterArena(d.desktop(), now)
	first, ok := mtime(ResumePath())
	if !ok {
		t.Fatal("the first prompt did not touch the resume file")
	}
	enterArena(d.desktop(), now.Add(time.Minute))
	second, ok := mtime(ResumePath())
	if !ok {
		t.Fatal("the resume file went missing")
	}
	if !second.After(first) {
		t.Errorf("the second prompt left the resume file at %v, want it moved past %v", second, first)
	}
}

// The window handed back is the one the prompt was typed in, which is the one
// that had the focus - read BEFORE the game is raised over it.
func TestTheWindowHandedBackIsTheOneThePromptWasTypedIn(t *testing.T) {
	arenaHome(t, true)
	d := &desk{active: "0x4200006", titles: map[string]string{"0x4200006": "claude"}}

	if home := enterArena(d.desktop(), time.Now()); home != "0x4200006" {
		t.Errorf("the arena handed back %q, want the window that had the focus", home)
	}
}

// Unless the focus was on the game itself, which is not a window to hand
// anything back to: the focus would go where it already is and the tab with
// Claude in it would never come forward.
func TestAPromptCannotHandTheFocusBackToTheGamesOwnWindow(t *testing.T) {
	arenaHome(t, true)
	d := &desk{active: "0x99", titles: map[string]string{"0x99": arenaTitle}}

	if home := enterArena(d.desktop(), time.Now()); home != "" {
		t.Errorf("the arena handed back %q, which is its own window", home)
	}
}

// The other half: the turn ends and the focus goes back, once, to that window.
func TestTheTurnEndingHandsTheFocusBack(t *testing.T) {
	arenaHome(t, true)
	d := &desk{}
	giveBackArena(d.desktop(), "0x4200006")

	_, _, focused := d.counts()
	if len(focused) != 1 || focused[0] != "0x4200006" {
		t.Errorf("the turn ended and the focus went to %v, want one window", focused)
	}
}

// And it stays where it is when there is nothing to go back to, or when the
// arena is off: a Stop still pauses the game either way, and pausing is Touch.
func TestTheFocusIsLeftAloneWhenThereIsNowhereToPutIt(t *testing.T) {
	for _, c := range []struct {
		name   string
		on     bool
		window string
	}{
		{"no window was remembered", true, ""},
		{"the arena is off", false, "0x1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			arenaHome(t, c.on)
			d := &desk{}
			giveBackArena(d.desktop(), c.window)
			if _, _, focused := d.counts(); len(focused) != 0 {
				t.Errorf("the focus was moved to %v", focused)
			}
		})
	}
}

// The heartbeat says which process is drawing the frames, and only that process
// may drop it. A second game started by hand and quit must not free a slot the
// arena's game is still holding, or the next prompt opens a second window on top
// of a game that is already playing.
func TestOnlyTheGameThatIsPlayingCanGiveTheSlotBack(t *testing.T) {
	arenaHome(t, true)
	now := time.Now()

	if err := Beat(now); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(LivePath())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(raw), strconv.Itoa(os.Getpid())+"\n"; got != want {
		t.Errorf("the heartbeat says %q, want this process %q", got, want)
	}
	Unbeat()
	if Live(now) {
		t.Error("a game that quit is still holding the slot")
	}

	// Somebody else's claim, and this process is not it.
	if err := os.WriteFile(LivePath(), []byte("999999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	Unbeat()
	if !Live(time.Now()) {
		t.Error("Unbeat dropped a claim that belonged to another game")
	}
}

// The switch is one file, and off means gone: a setting that says "off" inside a
// file is a setting that survives being deleted, and this one has to be readable
// with an ls by somebody wondering why terminals keep opening.
func TestTheSwitchIsOneFileAndOffTakesItAway(t *testing.T) {
	arenaHome(t, false)
	if ArenaOn() {
		t.Fatal("the arena is on before anybody turned it on")
	}
	if err := SetArena(true); err != nil {
		t.Fatal(err)
	}
	if !ArenaOn() {
		t.Error("the switch was thrown and the arena is still off")
	}
	if err := SetArena(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ArenaPath()); !os.IsNotExist(err) {
		t.Errorf("the switch is off and the file is still there: %v", err)
	}
	// Off twice is not an error: `ccpet arena off` twice is a thing a person does.
	if err := SetArena(false); err != nil {
		t.Errorf("turning it off twice failed: %v", err)
	}
}
