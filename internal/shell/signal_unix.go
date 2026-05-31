//go:build !windows

package shell

import (
	"os"
	"syscall"
)

// GracefulStop sends SIGTERM to a process, giving it a chance to clean up
// open resources, flush buffers, and shut down gracefully.
func GracefulStop(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
