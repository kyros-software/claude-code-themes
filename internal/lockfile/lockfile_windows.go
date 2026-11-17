//go:build windows

package lockfile

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

// NoFollow is nothing here: there is no O_NOFOLLOW to ask for, and making a
// symlink on Windows takes a privilege a stranger on the box does not have.
const NoFollow = 0

// LockFileEx is reached through kernel32 rather than through the syscall
// package: it is not in the standard library's Windows syscall surface, and the
// only other way to it is golang.org/x/sys, which is a dependency this module
// does not have and is not worth taking for two calls.
var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	failImmediately = 0x1
	exclusiveLock   = 0x2
	// One byte at offset zero is enough: the lock is a token, not a range.
	lockBytes = 1
)

func lockFileEx(h syscall.Handle, flags uint32, ol *syscall.Overlapped) error {
	r, _, err := procLockFileEx.Call(uintptr(h), uintptr(flags), 0,
		lockBytes, 0, uintptr(unsafe.Pointer(ol)))
	if r == 0 {
		return err
	}
	return nil
}

func unlockFileEx(h syscall.Handle, ol *syscall.Overlapped) {
	procUnlockFileEx.Call(uintptr(h), 0, lockBytes, 0, uintptr(unsafe.Pointer(ol)))
}

func take(path string) (func(), bool) {
	f, err := os.OpenFile(Path(path), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return func() {}, false
	}
	handle := syscall.Handle(f.Fd())
	deadline := time.Now().Add(Wait)
	for {
		var overlapped syscall.Overlapped
		if lockFileEx(handle, exclusiveLock|failImmediately, &overlapped) == nil {
			released := false
			return func() {
				if released {
					return
				}
				released = true
				var o syscall.Overlapped
				unlockFileEx(handle, &o)
				f.Close()
			}, true
		}
		if time.Now().After(deadline) {
			f.Close()
			return func() {}, false
		}
		time.Sleep(poll)
	}
}
