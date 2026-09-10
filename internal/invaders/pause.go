package invaders

import (
	"os"
	"path/filepath"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// The auto-pause. No build tags and nothing to do with a terminal in here, on
// purpose: internal/hook imports this file, and a Stop hook must not drag
// termios into the path of every tool call.

// PausePath is the file Claude touches when it finishes answering, and the one
// the game watches.
//
// One file rather than one per session. The game has no session id and cannot
// enumerate sessions, and "a Claude I am running has finished" is exactly the
// question it wants answered - not "that particular one has". It is one stat of
// one fixed path twice a second, which is unmeasurable.
func PausePath() string { return filepath.Join(config.Dir(), "ccpet-stop") }

// Touch marks the file as of now.
//
// The clock is handed in because the hook that calls this already has one and
// because a test needs to put a file in the past. Failures are ignored by the
// caller: a game that is not running is the common case, and a Stop hook must
// never be the reason a tool call reports an error.
func Touch(now time.Time) error {
	path := PausePath()
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

// pauseWatch remembers how new the pause file was when the run started, so a
// leftover from last week never pauses a fresh game.
//
// The file is never deleted here. Two games may be watching it, and a
// zero-length file in ~/.claude that means nothing to a new run is not worth the
// race.
type pauseWatch struct {
	seen time.Time
}

func watchPause() pauseWatch {
	at, _ := pauseAt()
	return pauseWatch{seen: at}
}

// fired says whether Claude has stopped since the last time this was asked.
func (w *pauseWatch) fired() bool {
	at, ok := pauseAt()
	if !ok || !at.After(w.seen) {
		return false
	}
	w.seen = at
	return true
}

func pauseAt() (time.Time, bool) {
	info, err := os.Stat(PausePath())
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}
