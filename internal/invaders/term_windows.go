//go:build windows

package invaders

import "errors"

// There is no /dev/tty to open and no termios to ask on Windows, so the game
// says so and stops rather than painting a frame nobody can steer. The same
// shape internal/statusline/tty_windows.go takes for the same reason.
var errNoTTY = errors.New("no controlling terminal")

type term struct{}

func openTerm() (*term, error) { return nil, errNoTTY }

func (t *term) raw() (func(), error)        { return func() {}, errNoTTY }
func (t *term) size() (cols, rows int)      { return 80, 24 }
func (t *term) Read(p []byte) (int, error)  { return 0, errNoTTY }
func (t *term) Write(p []byte) (int, error) { return 0, errNoTTY }
func (t *term) Close() error                { return nil }
