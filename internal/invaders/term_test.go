//go:build !windows

package invaders

import (
	"syscall"
	"testing"
)

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

// The bug that shipped a dead keyboard, as a test.
//
// The obvious raw-mode setting is VMIN 0 with VTIME 1: "come back in a tenth of
// a second with whatever there is". It does not work, because os.File turns a
// read that returns nothing into io.EOF - so the goroutine reading keys saw an
// EOF a tenth of a second after the game started, took it for a closed terminal
// and returned. Not one keypress reached the game for the rest of the run, and
// every test in this package passed, because they all drive the loop through a
// fake key channel and never go near a terminal.
func TestRawModeBlocksForAKeyRatherThanTimingOut(t *testing.T) {
	var prev syscall.Termios
	prev.Lflag = syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	prev.Iflag = syscall.IXON | syscall.ICRNL | syscall.IGNBRK

	next := rawTermios(prev)

	if next.Cc[syscall.VMIN] != 1 {
		t.Errorf("VMIN is %d, want 1: a read has to block for a key", next.Cc[syscall.VMIN])
	}
	if next.Cc[syscall.VTIME] != 0 {
		t.Errorf("VTIME is %d, want 0: a timed-out read comes back as io.EOF",
			next.Cc[syscall.VTIME])
	}
	// Compared one at a time rather than through a table: Termios.Lflag is 32
	// bits on Linux and 64 on Darwin, so a map of one of them will not build on
	// the other - and this file is compiled on both.
	if next.Lflag&syscall.ECHO != 0 {
		t.Error("ECHO is still set: every keypress would be printed over the frame")
	}
	if next.Lflag&syscall.ICANON != 0 {
		t.Error("ICANON is still set: nothing arrives until Enter")
	}
	if next.Lflag&syscall.ISIG != 0 {
		t.Error("ISIG is still set: ctrl-c would never reach Decode")
	}
	if next.Lflag&syscall.IEXTEN == 0 {
		t.Error("it cleared a flag it was not asked to")
	}
	if next.Iflag&syscall.IGNBRK == 0 {
		t.Error("it cleared an input flag it was not asked to")
	}
}
