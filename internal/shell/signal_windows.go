//go:build windows

package shell

import "os"

// GracefulStop attempts to gracefully stop a process on Windows.
// Windows does not support SIGTERM natively, so this falls back to
// TerminateProcess via Kill(). For console applications started in their own
// process group, a CTRL_BREAK_EVENT could be sent, but Kill() is the
// most reliable cross-application approach.
func GracefulStop(proc *os.Process) error {
	return proc.Kill()
}
