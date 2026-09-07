package statusline

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/session"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// What the band needs from git is two things with very different prices.
//
// The BRANCH is written down in .git/HEAD and costs a read. The DIRTY flag
// needs the whole working tree walked, which is a fork of git: measured here,
// 16 ms against 3.9 µs for the rest of a refresh - the statusline was 99.98%
// git. At refreshInterval: 1 that is one fork per second per open session.
//
// So they are split. The branch is read straight from the file and is always
// exact. The dirty flag is cached for dirtyTTL, which is the only thing in the
// footer that can lag, and it lags by at most three seconds.

// dirtyTTL is how long a working-tree reading is reused. Three seconds is two
// refreshes' worth: long enough to cut the forks to a third, short enough that
// the mark appears while your hand is still on the keyboard.
const dirtyTTL = 3 * time.Second

// gitDir resolves the .git of a repo root, following the one-line "gitdir:"
// pointer that worktrees and submodules leave in place of the directory.
func gitDir(root string) string {
	path := filepath.Join(root, ".git")
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return path
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(raw))
	target, ok := strings.CutPrefix(line, "gitdir:")
	if !ok {
		return ""
	}
	target = strings.TrimSpace(target)
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	return target
}

// branchOf is the checked-out branch, read from HEAD rather than asked of git.
//
// HEAD holds either "ref: refs/heads/<name>" or a bare sha when detached. A
// repo with no commits yet still names its branch here, which is what the old
// code had to recognise the sentence "No commits yet on main" to find out.
func branchOf(root string) string {
	dir := gitDir(root)
	if dir == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(dir, "HEAD"))
	if err != nil {
		return ""
	}
	// Flattened like every other name from outside: git refuses control
	// characters in a ref, but nothing refuses them in a HEAD written by hand.
	head := theme.OneLine(strings.TrimSpace(string(raw)))
	ref, ok := strings.CutPrefix(head, "ref:")
	if !ok {
		// Detached: a bare sha, and no name to give.
		return "HEAD"
	}
	ref = strings.TrimSpace(ref)
	if name, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		return name
	}
	// Anything else HEAD can point at - a remote-tracking ref, mostly - keeps
	// its last segment. A "ref:" with nothing after it is not a name at all:
	// filepath.Base("") is ".", and a lone dot in the band is worse than
	// nothing, which is what an unreadable HEAD already gives.
	if ref == "" {
		return ""
	}
	return filepath.Base(ref)
}

// dirtyCache is what one repo's working-tree reading looks like on disk.
type dirtyCache struct {
	Root  string  `json:"root"`
	Dirty bool    `json:"dirty"`
	At    float64 `json:"at"`
}

// dirtyCachePath is per session, and shares the prefix so session.Sweep
// collects it along with everything else the statusline leaves in $TMPDIR.
func dirtyCachePath(sessionID string) string { return session.PathFor(sessionID, "git") }

// dirtyOf says whether the working tree has changes, reusing a reading younger
// than dirtyTTL taken in the SAME repo. A cwd that moves between repos - a
// subagent working elsewhere, a cd - misses the cache and pays for git, which
// is correct: the two trees have nothing to say about each other.
func dirtyOf(root, sessionID string, now time.Time) bool {
	path := dirtyCachePath(sessionID)
	if cached, ok := readDirtyCache(path, root, now); ok {
		return cached
	}
	dirty := gitDirty(root)
	if path != "" {
		if raw, err := json.Marshal(dirtyCache{
			Root: root, Dirty: dirty, At: float64(now.UnixNano()) / 1e9,
		}); err == nil {
			session.WriteAtomic(path, raw)
		}
	}
	return dirty
}

func readDirtyCache(path, root string, now time.Time) (bool, bool) {
	if path == "" {
		return false, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, false
	}
	var c dirtyCache
	if json.Unmarshal(raw, &c) != nil || c.Root != root {
		return false, false
	}
	age := float64(now.UnixNano())/1e9 - c.At
	if age < 0 || age > dirtyTTL.Seconds() {
		return false, false
	}
	return c.Dirty, true
}

// gitDirty is the fork. --untracked-files=normal is git's own default and is
// kept explicit here: a brand new file counts as a change, which is the reading
// this footer has always given.
func gitDirty(root string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "-C", root,
		"status", "--porcelain=v1", "--untracked-files=normal")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
