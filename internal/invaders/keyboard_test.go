package invaders

import (
	"reflect"
	"strings"
	"testing"
)

// xset's own output, verbatim from the desktop this was written on. The two
// numbers in it are the reason the whole file exists: half a second before a held
// arrow starts repeating is half a second of a ship that looks broken.
const xsetQ = `Keyboard Control:
  auto repeat:  on    key click percent:  0    LED mask:  00000002
  XKB indicators:
    00: Caps Lock:   off    01: Num Lock:    on
  auto repeat delay:  500    repeat rate:  33
  auto repeating keys:  00ffffffdffffbbf
                        fadfffefffedffff
  bell percent:  50    bell pitch:  400    bell duration:  100
`

func TestTheRepeatSettingIsReadOutOfXset(t *testing.T) {
	delay, rate, ok := repeatIn(xsetQ)
	if !ok {
		t.Fatal("xset's own output was not understood")
	}
	if delay != 500 || rate != 33 {
		t.Errorf("read delay %d rate %d, want 500 and 33", delay, rate)
	}
	if delay <= ourDelay {
		t.Errorf("this desktop's delay is %dms and ours is %dms: there would be nothing to fix",
			delay, ourDelay)
	}
}

// Anything else is left alone rather than guessed at: a desktop whose xset says
// something this has never seen keeps whatever it had.
func TestAnUnfamiliarXsetIsLeftAlone(t *testing.T) {
	for _, out := range []string{
		"", "Keyboard Control:\n  auto repeat:  off\n",
		"auto repeat delay:  nonsense    repeat rate:  nonsense\n",
		"auto repeat delay:  0    repeat rate:  0\n",
	} {
		if _, _, ok := repeatIn(out); ok {
			t.Errorf("%q was read as a setting", out)
		}
	}
}

// With no X, no xset or an explicit no, the keyboard is not touched at all - and
// restore on a keyboard that borrowed nothing must not run xset either.
func TestWithoutXNothingIsBorrowedAndNothingIsRestored(t *testing.T) {
	t.Setenv("DISPLAY", "")
	if k := borrowKeyboard(); k.ours {
		t.Error("with no DISPLAY it still thinks it has a keyboard to borrow")
	}

	t.Setenv("DISPLAY", ":0")
	t.Setenv(noXset, "1")
	k := borrowKeyboard()
	if k.ours {
		t.Errorf("%s is set and it borrowed anyway", noXset)
	}
	// Both are no-ops, and neither may panic: they are called from a defer.
	k.quicken()
	k.restore()
	if k.fast {
		t.Error("it thinks it changed something")
	}
}

// The window's own news is not a key. It must never reach the tick, which has no
// idea what a window is - and it must reach the keyboard, which is the only
// reason it is asked for.
func TestFocusIsNotAKeyAndNeverReachesTheTick(t *testing.T) {
	for _, in := range []string{"\033[I", "\033[O"} {
		k, n := Decode([]byte(in))
		if n != 3 {
			t.Errorf("%q consumed %d bytes", in, n)
		}
		if k != FocusIn && k != FocusOut {
			t.Errorf("%q decoded to %d", in, k)
		}
		// A frame with the focus in it has to be the same frame as one with
		// nothing in it - not the same as the frame before, since every tick
		// moves the clock on.
		g := aGame(t, "spark", 1)
		if !reflect.DeepEqual(Tick(g, k), Tick(g, None)) {
			t.Errorf("%q did something to the game", in)
		}
	}

	// And the loop hands it to the screen instead.
	raw := readSource(t, "run.go")
	if !strings.Contains(raw, "sc.Focus(k == FocusIn)") {
		t.Error("the loop does not hand the focus to the screen")
	}
	if !strings.Contains(raw, "1004") {
		t.Error("nothing turns focus reporting on, so the news never arrives")
	}
}
