//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package invaders

import "syscall"

// The BSDs, macOS included, spell the same two ioctls differently.
const (
	getAttr = syscall.TIOCGETA
	setAttr = syscall.TIOCSETA
)
