package invaders

import (
	"os"
	"path/filepath"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// The two files a turn talks to the game through. No build tags and nothing to
// do with a terminal in here, on purpose: internal/hook imports this file, and
// neither a Stop nor a prompt must drag termios into the path of every tool
// call.
//
// Two files and not one with a word in it: the signal is the mtime, and a mtime
// is something both sides can compare without reading, parsing or locking
// anything. Which of the two moved last is the whole protocol.

// PausePath is the file Claude touches when it finishes answering, and the one
// the game watches to stop and let you go and read the answer.
//
// One file rather than one per session. The game has no session id and cannot
// enumerate sessions, and "a Claude I am running has finished" is exactly the
// question it wants answered - not "that particular one has". It is one stat of
// one fixed path four times a second, which is unmeasurable.
func PausePath() string { return filepath.Join(config.Dir(), "ccpet-stop") }

// ResumePath is the other half: touched when a turn STARTS, and what takes the
// game back out of the pause the last Stop left it in. Without it the arena
// would open a window showing a paused game and wait for you to press p, every
// single turn.
func ResumePath() string { return filepath.Join(config.Dir(), "ccpet-play") }

// Touch marks the pause file as of now.
//
// The clock is handed in because the hook that calls this already has one and
// because a test needs to put a file in the past. Failures are ignored by the
// caller: a game that is not running is the common case, and a hook must never
// be the reason a tool call reports an error.
func Touch(now time.Time) error { return touch(PausePath(), now) }

// Resume marks the resume file, which is a prompt saying the wait is back on.
func Resume(now time.Time) error { return touch(ResumePath(), now) }

func touch(path string, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chtimes(path, now, now)
}

// fileWatch remembers how new a file was when the run started, so a leftover
// from last week never pauses - or wakes - a fresh game.
//
// The files are never deleted here. Two games may be watching them, and a
// zero-length file in ~/.claude that means nothing to a new run is not worth the
// race.
type fileWatch struct {
	path string
	seen time.Time
}

func watchFile(path string) fileWatch {
	at, _ := mtime(path)
	return fileWatch{path: path, seen: at}
}

// fired says whether the file has been touched since the last time this was
// asked.
func (w *fileWatch) fired() bool {
	at, ok := mtime(w.path)
	if !ok || !at.After(w.seen) {
		return false
	}
	w.seen = at
	return true
}

func mtime(path string) (time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}
