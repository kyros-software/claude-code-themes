//go:build !windows

package invaders

import "testing"

// The geometry has to be exercisable without a terminal, or every test that
// depends on the field size needs a tty and CI has none.
func TestTheSizeFallsBackToTheEnvironmentAndThenToEightyByTwentyFour(t *testing.T) {
	tm := &term{}

	t.Run("the override wins", func(t *testing.T) {
		t.Setenv("CCPET_INVADE_SIZE", "72x20")
		if c, r := tm.size(); c != 72 || r != 20 {
			t.Errorf("got %dx%d, want 72x20", c, r)
		}
	})

	t.Run("junk in the override is ignored rather than believed", func(t *testing.T) {
		for _, raw := range []string{"", "wide", "72", "x20", "-5x-5", "0x0", "72x"} {
			t.Setenv("CCPET_INVADE_SIZE", raw)
			t.Setenv("COLUMNS", "")
			t.Setenv("LINES", "")
			if c, r := tm.size(); c < MinCols || r < 1 {
				t.Errorf("%q gave %dx%d", raw, c, r)
			}
		}
	})

	t.Run("then columns and lines", func(t *testing.T) {
		t.Setenv("CCPET_INVADE_SIZE", "")
		t.Setenv("COLUMNS", "100")
		t.Setenv("LINES", "30")
		// A zero file descriptor is not a terminal, so the ioctl fails and the
		// environment is what is left.
		if c, r := tm.size(); c != 100 || r != 30 {
			t.Errorf("got %dx%d, want 100x30", c, r)
		}
	})

	t.Run("and the universal default", func(t *testing.T) {
		t.Setenv("CCPET_INVADE_SIZE", "")
		t.Setenv("COLUMNS", "")
		t.Setenv("LINES", "")
		if c, r := tm.size(); c != 80 || r != 24 {
			t.Errorf("got %dx%d, want 80x24", c, r)
		}
	})
}

// The two termios ioctls are the only thing that differs between the unixes, and
// getting them wrong is a compile error on half of CI's matrix rather than a
// test failure. This asserts they are set at all; `GOOS=darwin go vet ./...` is
// what actually proves the split.
func TestTheTermiosIoctlsAreSetForThisPlatform(t *testing.T) {
	if getAttr == 0 || setAttr == 0 {
		t.Fatalf("getAttr = %v, setAttr = %v", getAttr, setAttr)
	}
	if getAttr == setAttr {
		t.Error("reading and writing the terminal attributes are the same ioctl")
	}
}

// There is no terminal in a test, and asking for one has to be an error rather
// than a hang or a panic.
func TestOpeningATerminalThatIsNotThereIsAnErrorAndNotAHang(t *testing.T) {
	tm, err := openTerm()
	if err != nil {
		return // no /dev/tty here, which is the case this is about
	}
	// There is one (someone is running the tests from a terminal); it must at
	// least close cleanly without having been put into raw mode.
	if err := tm.Close(); err != nil {
		t.Errorf("closing it failed: %v", err)
	}
}
