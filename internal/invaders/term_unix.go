//go:build !windows

package invaders

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// The terminal, on everything that has a /dev/tty.
//
// No third-party package: this repo has no dependencies and is not going to
// acquire one for four ioctls. The shape follows internal/statusline/tty_unix.go,
// which already asks the controlling terminal for its size the same way.

// errNoTTY is what Run reports when there is no terminal to play in.
var errNoTTY = errors.New("no controlling terminal")

type term struct {
	f    *os.File
	prev syscall.Termios
}

// openTerm takes the controlling terminal. /dev/tty and not stdin, so the game
// still works when something has piped its input.
func openTerm() (*term, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &term{f: f}, nil
}

// raw turns off echo, line buffering and the signal characters, and returns the
// undo.
//
// VMIN 1 with VTIME 0: a read blocks until at least one byte arrives. The
// obvious setting is the other one - VMIN 0, VTIME 1, "come back in a tenth of a
// second with whatever there is" - and it does not work here, because os.File
// turns a read that returns nothing into io.EOF. The reader goroutine saw that
// EOF a tenth of a second after the game started, took it for a closed terminal
// and returned, and from then on NOT ONE KEY reached the game. Blocking is fine:
// the reader is a goroutine of its own and the frames are drawn by the loop.
func (t *term) raw() (restore func(), err error) {
	if err := ioctl(t.f.Fd(), getAttr, unsafe.Pointer(&t.prev)); err != nil {
		return nil, err
	}
	next := rawTermios(t.prev)
	if err := ioctl(t.f.Fd(), setAttr, unsafe.Pointer(&next)); err != nil {
		return nil, err
	}
	prev := t.prev
	return func() { ioctl(t.f.Fd(), setAttr, unsafe.Pointer(&prev)) }, nil
}

// size asks CCPET_INVADE_SIZE, then the terminal, then COLUMNS and LINES, and
// gives up at the universal default.
//
// The override is first because it is the only way to exercise the geometry
// without a terminal at all. After it the ioctl beats COLUMNS, which is the
// other way round from the statusline: that runs once per refresh with whatever
// environment it was handed, while this one listens for SIGWINCH and has to
// believe the window over a variable that went stale when it was resized.
func (t *term) size() (cols, rows int) {
	if c, r := envSize(); c > 0 && r > 0 {
		return c, r
	}
	var ws struct{ Row, Col, X, Y uint16 }
	if err := ioctl(t.f.Fd(), syscall.TIOCGWINSZ, unsafe.Pointer(&ws)); err == nil {
		if ws.Col > 0 && ws.Row > 0 {
			return int(ws.Col), int(ws.Row)
		}
	}
	if c, r := envOf("COLUMNS"), envOf("LINES"); c > 0 && r > 0 {
		return c, r
	}
	return 80, 24
}

// rawTermios is the flags a raw terminal wants, as a function of the flags it
// had. Split out from raw so it can be tested: everything else in here needs a
// terminal to say anything, and this is the part that was wrong.
func rawTermios(prev syscall.Termios) syscall.Termios {
	next := prev
	next.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	next.Iflag &^= syscall.IXON | syscall.ICRNL
	next.Cc[syscall.VMIN] = 1
	next.Cc[syscall.VTIME] = 0
	return next
}

func (t *term) Read(p []byte) (int, error)  { return t.f.Read(p) }
func (t *term) Write(p []byte) (int, error) { return t.f.Write(p) }
func (t *term) Close() error                { return t.f.Close() }

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg)); errno != 0 {
		return errno
	}
	return nil
}

// envSize is CCPET_INVADE_SIZE, which is how the geometry is exercised without
// a terminal at all.
func envSize() (cols, rows int) {
	raw := os.Getenv("CCPET_INVADE_SIZE")
	x := strings.IndexAny(raw, "xX*")
	if x < 0 {
		return 0, 0
	}
	c, cerr := strconv.Atoi(strings.TrimSpace(raw[:x]))
	r, rerr := strconv.Atoi(strings.TrimSpace(raw[x+1:]))
	if cerr != nil || rerr != nil || c <= 0 || r <= 0 {
		return 0, 0
	}
	return c, r
}

func envOf(name string) int {
	n, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return 0
	}
	return n
}
