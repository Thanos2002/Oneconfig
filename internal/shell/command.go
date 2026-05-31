// Package shell provides cross-platform shell command utilities,
// centralizing the OS-specific logic for spawning shell processes.
package shell

import (
	"os/exec"
	"runtime"
)

// Command creates an exec.Cmd that executes a command string via the
// platform's native shell interpreter: sh -c on Unix, cmd /c on Windows.
func Command(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", command)
	}
	return exec.Command("sh", "-c", command)
}
