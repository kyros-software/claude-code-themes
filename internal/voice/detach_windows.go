package voice

import (
	"os/exec"
	"syscall"
)

// DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP: no console, and its own group.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200}
}
