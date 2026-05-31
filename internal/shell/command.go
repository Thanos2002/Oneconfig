// Package shell provides cross-platform shell command utilities,
// centralizing the OS-specific logic for spawning shell processes.
package shell

import (
	"context"
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

// CommandContext creates an exec.Cmd with a context.
func CommandContext(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}
