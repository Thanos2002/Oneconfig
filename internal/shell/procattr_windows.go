//go:build windows

package shell

import (
	"os/exec"
	"syscall"
)

// SetNewProcessGroup configures the command to start in a new process group
// on Windows using CREATE_NEW_PROCESS_GROUP, enabling clean teardown.
func SetNewProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}
