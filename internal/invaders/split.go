package invaders

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
)

// split opens the game beside Claude in a tmux pane, when there is a tmux to
// open it in.
//
// The game owns a terminal and Claude Code owns another, so switching between
// them is the emulator's own tab key and nothing ccpet can reach. Inside tmux it
// can be one keystroke instead, and that is worth ten lines - guarded, because
// tmux is a convenience here and never a dependency. Its absence is a sentence,
// not an error the user has to look up.
//
// It insists on being inside tmux already: splitting from outside would create a
// detached session that nobody ever sees, which looks exactly like the command
// having silently done nothing.
func split(stdout, stderr io.Writer) int {
	if _, err := exec.LookPath("tmux"); err != nil || os.Getenv("TMUX") == "" {
		fmt.Fprintln(stderr, "ccpet:", i18n.G().NoTmux)
		return 1
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(stderr, "ccpet:", err)
		return 1
	}
	if err := exec.Command("tmux", "split-window", "-h", exe+" invade").Run(); err != nil {
		fmt.Fprintln(stderr, "ccpet:", err)
		return 1
	}
	return 0
}
