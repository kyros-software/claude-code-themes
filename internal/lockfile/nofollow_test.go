//go:build !windows

package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

// A symlink planted where the lock goes must not be followed: the lock is
// refused, and nothing is created on the far end.
func TestAPlantedSymlinkIsNotFollowed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")
	elsewhere := filepath.Join(dir, "elsewhere")
	if err := os.Symlink(elsewhere, Path(path)); err != nil {
		t.Skip("no symlinks here:", err)
	}
	release, ok := Take(path)
	release()
	if ok {
		t.Fatal("the lock was taken through a symlink")
	}
	if _, err := os.Lstat(elsewhere); !os.IsNotExist(err) {
		t.Fatalf("the open reached the far end of the link: %v", err)
	}
}
