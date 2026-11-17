//go:build !windows

package lockfile

import (
	"os"
	"syscall"
	"time"
)

// NoFollow goes on every open of a file in the scratch directory. That
// directory is $TMPDIR, which on a shared box other people can write to, and
// an open with O_CREATE and nothing else FOLLOWS a symlink planted at the path:
// the lock would be created, and the tool log appended, wherever it points.
// With this the open fails instead, which every caller already treats as "no
// lock" or "nothing logged".
const NoFollow = syscall.O_NOFOLLOW

func take(path string) (func(), bool) {
	f, err := os.OpenFile(Path(path), os.O_CREATE|os.O_RDWR|NoFollow, 0o600)
	if err != nil {
		return func() {}, false
	}
	deadline := time.Now().Add(Wait)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return func() {
				syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
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
