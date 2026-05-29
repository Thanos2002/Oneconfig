// Package services manages the lifecycle of local development services
// (starting, stopping, health monitoring, and log capture).
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/oneconfig/oneconfig/internal/config"
)

const stateFileName = ".oneconfig.state.json"

// Manager handles starting, stopping, and monitoring services.
type Manager struct {
	projectDir string
	verbose    bool
	logsDir    string
}

// ServiceStatus holds the current state of a service.
type ServiceStatus struct {
	Name  string `json:"name"`
	Port  int    `json:"port"`
	PID   int    `json:"pid"`
	State string `json:"state"` // "running", "stopped", "error"
}

// stateFile persists PIDs between up/down invocations.
type stateFile struct {
	Services []ServiceStatus `json:"services"`
}

// NewManager creates a new service manager.
func NewManager(projectDir string, verbose bool) *Manager {
	logsDir := filepath.Join(projectDir, ".oneconfig", "logs")
	os.MkdirAll(logsDir, 0755)
	return &Manager{
		projectDir: projectDir,
		verbose:    verbose,
		logsDir:    logsDir,
	}
}

// Start launches a service and records its PID.
func (m *Manager) Start(svc config.Service) error {
	workDir := filepath.Join(m.projectDir, svc.WorkingDir)

	// Set up log files (FR-11)
	logFile, err := os.Create(filepath.Join(m.logsDir, svc.Name+".log"))
	if err != nil {
		return fmt.Errorf("creating log file for %s: %w", svc.Name, err)
	}

	// Build command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", svc.StartCommand)
	} else {
		cmd = exec.Command("sh", "-c", svc.StartCommand)
	}
	cmd.Dir = workDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Add service-specific env vars
	cmd.Env = os.Environ()
	for k, v := range svc.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Start the process in a new process group so we can kill the group later
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// CreationFlags for Windows; on Unix this would be Setpgid
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting %s: %w\n\n  💡 Check the command: %s\n     Logs: %s", svc.Name, err, svc.StartCommand, logFile.Name())
	}

	// Record PID in state file
	m.recordPID(svc.Name, svc.Port, cmd.Process.Pid)

	// Don't wait — the service runs in the background
	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	return nil
}

// StopAll stops all services recorded in the state file.
func (m *Manager) StopAll() (int, error) {
	state, err := m.loadState()
	if err != nil {
		return 0, nil // no state file = nothing to stop
	}

	stopped := 0
	var lastErr error

	for _, svc := range state.Services {
		if svc.PID <= 0 {
			continue
		}

		proc, err := os.FindProcess(svc.PID)
		if err != nil {
			continue
		}

		if err := proc.Kill(); err != nil {
			if !strings.Contains(err.Error(), "process already finished") {
				lastErr = fmt.Errorf("killing %s (PID %d): %w", svc.Name, svc.PID, err)
			}
		} else {
			stopped++
		}
	}

	// Clean up state file
	m.clearState()

	return stopped, lastErr
}

// Status returns the current status of all configured services.
func (m *Manager) Status(services []config.Service) []ServiceStatus {
	state, _ := m.loadState()
	pidMap := make(map[string]int)
	if state != nil {
		for _, s := range state.Services {
			pidMap[s.Name] = s.PID
		}
	}

	var statuses []ServiceStatus
	for _, svc := range services {
		pid := pidMap[svc.Name]
		st := ServiceStatus{
			Name:  svc.Name,
			Port:  svc.Port,
			PID:   pid,
			State: "stopped",
		}

		if pid > 0 {
			if isProcessRunning(pid) {
				st.State = "running"
			} else {
				st.State = "stopped"
			}
		}

		statuses = append(statuses, st)
	}

	return statuses
}

// recordPID adds a service PID to the state file.
func (m *Manager) recordPID(name string, port int, pid int) {
	state, _ := m.loadState()
	if state == nil {
		state = &stateFile{}
	}

	// Remove existing entry for this service
	var filtered []ServiceStatus
	for _, s := range state.Services {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}

	state.Services = append(filtered, ServiceStatus{
		Name:  name,
		Port:  port,
		PID:   pid,
		State: "running",
	})

	m.saveState(state)
}

func (m *Manager) loadState() (*stateFile, error) {
	path := filepath.Join(m.projectDir, ".oneconfig", stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state stateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *Manager) saveState(state *stateFile) error {
	dir := filepath.Join(m.projectDir, ".oneconfig")
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stateFileName), data, 0644)
}

func (m *Manager) clearState() {
	path := filepath.Join(m.projectDir, ".oneconfig", stateFileName)
	os.Remove(path)
}

// isProcessRunning checks if a process with the given PID is still alive.
func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, FindProcess actually returns an error if it doesn't exist.
		// So if err == nil, it's alive (or we lack permissions).
		return true
	}

	// On Unix, FindProcess always succeeds; we need to send signal 0 to check
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// Ensure the port is valid
func validatePort(port string) (int, error) {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("invalid port: %s", port)
	}
	return p, nil
}
