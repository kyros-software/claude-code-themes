//go:build !windows

package voice

import (
	"os/exec"
	"syscall"
)

// detach puts the process in a session of its own, so it outlives the hook
// that started it and whatever the CLI does to the hook's group afterwards.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
