package pet

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kyros-software/claude-code-themes/internal/lockfile"
	"strings"
	"sync"
	"testing"
	"time"
)

// The lock is only as good as the rule that everyone goes through it. This
// walks the source and fails if any production file outside this package still
// does its own Load-mutate-Save, which is the shape that lost 91% of the XP
// before pet.Update existed.
func TestNoProductionCodeCallsSaveDirectly(t *testing.T) {
	root := filepath.Join("..", "..")
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && (d.Name() == ".git" || d.Name() == "bin"):
			return filepath.SkipDir
		case d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go"):
			return nil
		}
		// This package is where Save lives; Update itself has to call it.
		if filepath.Dir(path) == filepath.Join(root, "internal", "pet") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			code := line
			if c := strings.Index(code, "//"); c >= 0 {
				code = code[:c]
			}
			if strings.Contains(code, "pet.Save(") {
				offenders = append(offenders,
					filepath.ToSlash(path)+":"+itoa(i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Errorf("these call pet.Save directly instead of pet.Update, which reopens"+
			" the lost update:\n  %s", strings.Join(offenders, "\n  "))
	}
}

// Two writers, one file, one lock: every meal has to survive.
func TestUpdateKeepsEveryConcurrentMeal(t *testing.T) {
	const meals = 100
	want := meals * Foods["compact"].XP
	now := time.Now()
	path := filepath.Join(t.TempDir(), "pet.json")

	var wg sync.WaitGroup
	for i := 0; i < meals; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Update(path, func(s *State) bool {
				Feed(s, "compact", "", now)
				return true
			})
		}()
	}
	wg.Wait()

	got := Load(path)
	if got.XP != want {
		t.Errorf("%d concurrent meals gave %d XP, want %d", meals, got.XP, want)
	}
	// The log is capped at LogMax, so only the tail of a run this size is
	// there; what matters is that it filled up rather than being overwritten
	// down to a handful, which is what the lost update did to it.
	if wantLog := min(meals, LogMax); len(got.Log) != wantLog {
		t.Errorf("%d concurrent meals left %d log entries, want %d",
			meals, len(got.Log), wantLog)
	}
}

// A change that says "nothing to write" must leave the file untouched.
func TestUpdateSkipsTheWriteWhenNothingChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pet.json")
	Update(path, func(s *State) bool { s.XP = 42; return true })
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if Update(path, func(s *State) bool { s.XP = 999; return false }) {
		t.Error("Update reported a write it was told not to make")
	}
	if got := Load(path).XP; got != 42 {
		t.Errorf("the skipped update wrote anyway: XP %d", got)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("the file was rewritten by an update that returned false")
	}
}

// The lock file sits beside the state, never on it: Save replaces pet.json by
// rename, so a lock held on the state file would guard an unlinked inode.
func TestTheLockIsItsOwnFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pet.json")
	Update(path, func(s *State) bool { s.XP = 1; return true })

	if _, err := os.Stat(lockfile.Path(path)); err != nil {
		t.Fatalf("no lock file beside the state: %v", err)
	}
	if lockfile.Path(path) == path {
		t.Fatal("the lock is the state file itself")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
