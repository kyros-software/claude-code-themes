//go:build linux

package invaders

import "syscall"

// The two termios ioctls, which are the only thing that differs between the
// unixes. Everything else in term_unix.go is shared - and the constants have to
// be split like this rather than switched on at run time because they are
// untyped on Linux and would not compile at all on a BSD.
const (
	getAttr = syscall.TCGETS
	setAttr = syscall.TCSETS
)
