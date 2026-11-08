package setup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Keeping ~/.claude/ccpet pointed at the plugin.
//
// WHY THIS EXISTS. A plugin is installed under
//
//	~/.claude/plugins/cache/<marketplace>/<plugin>/<version>/
//
// and THE VERSION IS IN THE PATH, so it changes on every plugin update. Hooks
// and MCP servers do not care - the CLI re-resolves ${CLAUDE_PLUGIN_ROOT} each
// time it loads them - but statusLine.command in settings.json is a plain
// string that nobody re-resolves. Pointed straight at the plugin, it would
// break the first time the plugin updated.
//
// So the statusline is pointed at a stable path we own, and this keeps that
// path aimed at the current plugin.

// RuntimeLink is the stable path to the runtime directory.
func RuntimeLink() string { return filepath.Join(ConfigDir(), "ccpet") }

// StatusLinePath is the stable path the statusLine key points at.
//
// It is a symlink to this machine's binary named "ccpet-statusline", and the
// binary reads its own argv[0] to know what to do. That buys two things: the
// settings.json entry is a bare path with no arguments - the shape the Python
// this replaces used, and the only shape guaranteed to work whether or not the
// host runs the command through a shell - and there is no shell in the way of
// something that runs once a second.
func StatusLinePath() string { return filepath.Join(ConfigDir(), "ccpet-statusline") }

// RunLinkPath is the stable path to THIS MACHINE's binary.
//
// hooks.json needs one fixed path, and picking a build out of five is what
// bin/ccpet does - but that is a bash process in front of every tool call, and
// the shell Claude Code already spawned can exec this directly instead. Worth
// about 1.3 ms on the most frequent path there is.
func RunLinkPath() string { return filepath.Join(ConfigDir(), "ccpet-run") }

// hostBinary is the build for this machine inside root.
func hostBinary(root string) string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return filepath.Join(root, "bin",
		fmt.Sprintf("ccpet-%s-%s%s", runtime.GOOS, runtime.GOARCH, suffix))
}

// symlink points link at target, atomically, and says nothing if it cannot.
func symlink(link, target string) {
	if current, err := os.Readlink(link); err == nil && current == target {
		return
	}
	tmp := link + ".tmp"
	os.Remove(tmp)
	if os.Symlink(target, tmp) != nil {
		return
	}
	if os.Rename(tmp, link) != nil {
		os.Remove(tmp)
	}
}

// linkRun points the two stable entry points at root's binary for this machine.
func linkRun(root string) {
	target := hostBinary(root)
	if info, err := os.Stat(target); err != nil || info.IsDir() {
		return
	}
	symlink(RunLinkPath(), target)
	symlink(StatusLinePath(), target)
}

// PluginRoot is where this build is installed from, when it came from a plugin.
func PluginRoot() string { return os.Getenv("CLAUDE_PLUGIN_ROOT") }

// LinkState says what the stable path currently is.
type LinkState int

// The three things the stable path can be.
const (
	LinkMissing LinkState = iota
	LinkSymlink
	LinkRealDir
)

func linkState(path string) (LinkState, string) {
	info, err := os.Lstat(path)
	if err != nil {
		return LinkMissing, ""
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, _ := os.Readlink(path)
		return LinkSymlink, target
	}
	if info.IsDir() {
		return LinkRealDir, ""
	}
	return LinkMissing, ""
}

// Link points the stable path at root. It reports whether anything changed.
//
// It never deletes a real directory: one there belongs to scripts/install.sh,
// and that copy wins. Two installers fighting over one path is worse than one
// of them stepping aside.
func Link(root string) (bool, error) {
	if root == "" {
		return false, nil
	}
	if _, err := os.Stat(filepath.Join(root, "bin")); err != nil {
		return false, nil
	}
	link := RuntimeLink()

	switch state, target := linkState(link); state {
	case LinkSymlink:
		if target == root {
			linkRun(root)
			return false, nil // the common case, and it costs one lstat
		}
		if newerInstall(target, root) {
			// A session opened before the update still runs the old hooks,
			// and its SessionStart must not take the pet back a version.
			linkRun(target)
			return false, nil
		}
	case LinkRealDir:
		linkRun(link) // the legacy installer owns the directory; still point at it
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return false, err
	}
	tmp := link + ".tmp"
	os.Remove(tmp)
	if err := os.Symlink(root, tmp); err != nil {
		// Windows without developer mode, or a filesystem with no symlinks.
		// Not fatal: scripts/install.sh copies instead.
		return false, err
	}
	if err := os.Rename(tmp, link); err != nil {
		os.Remove(tmp)
		return false, err
	}
	linkRun(root)
	return true, nil
}

// newerInstall says whether current is a later install of the same plugin than
// root: two version directories side by side in the plugin cache, current
// with the higher version, and still there.
//
// CLAVE: SessionStart runs the link of whichever version the session loaded,
// and a session stays on the version it was opened with. Before this, one
// still-open session that compacted after an update pointed every stable link
// back at its own version, and from then on every session ran the old binary
// over a state the new one had already written (measured 8-nov-2026, 3.0.0
// taken back to 2.2.0). Anything that does not look like that - a hand-made
// link, a name that is not a version - is not refused, as before.
func newerInstall(current, root string) bool {
	if filepath.Dir(filepath.Clean(current)) != filepath.Dir(filepath.Clean(root)) {
		return false
	}
	cur, ok1 := parseVersion(filepath.Base(current))
	old, ok2 := parseVersion(filepath.Base(root))
	if !ok1 || !ok2 {
		return false
	}
	if info, err := os.Stat(filepath.Join(current, "bin")); err != nil || !info.IsDir() {
		return false
	}
	for i := range cur {
		if cur[i] != old[i] {
			return cur[i] > old[i]
		}
	}
	return false
}

// parseVersion reads "3.0.1" as three numbers. Anything else is not a version.
func parseVersion(name string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(name, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

// linkRunFor is linkRun, exported inside the package for setup.Command.
func linkRunFor(root string) { linkRun(root) }

// Heal repairs the stable path when it has gone dangling, which is what a
// plugin update leaves behind until the next SessionStart. Called from the hook
// on every tool call, so it is one lstat on the normal path.
func Heal() {
	root := PluginRoot()
	if root == "" {
		return
	}
	if _, err := os.Stat(filepath.Join(RuntimeLink(), "bin")); err == nil {
		_, runErr := os.Stat(RunLinkPath())
		_, statusErr := os.Stat(StatusLinePath())
		if runErr == nil && statusErr == nil {
			return
		}
	}
	Link(root)
}

// RunLink is the SessionStart entry point.
func RunLink(out io.Writer, root string) error {
	if root == "" {
		root = PluginRoot()
	}
	changed, err := Link(root)
	if err != nil {
		fmt.Fprintf(out, "ccpet: could not link %s (%v)\n", RuntimeLink(), err)
		return nil // never block a session over this
	}
	if changed {
		fmt.Fprintf(out, "ccpet: %s -> %s\n", RuntimeLink(), root)
	}
	return nil
}
