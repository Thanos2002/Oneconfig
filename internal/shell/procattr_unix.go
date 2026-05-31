//go:build !windows

package shell

import (
	"os/exec"
	"syscall"
)

// SetNewProcessGroup configures the command to start in a new process group,
// allowing the entire process tree to be signalled later via the group ID.
func SetNewProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}
