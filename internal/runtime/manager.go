// Package runtime manages the detection and installation of language runtimes
// such as Node.js and Python.
package runtime

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Thanos2002/Oneconfig/internal/shell"
)

// Manager handles runtime version detection and installation.
type Manager struct {
	verbose bool
}

// NewManager creates a new runtime manager.
func NewManager(verbose bool) *Manager {
	return &Manager{verbose: verbose}
}

// Ensure checks if the required runtime version is available, and attempts
// to install or switch to it if not.
func (m *Manager) Ensure(name, version string) error {
	switch name {
	case "node":
		return m.ensureNode(version)
	case "python":
		return m.ensurePython(version)
	default:
		return fmt.Errorf("unsupported runtime: %s (supported: node, python)", name)
	}
}

// ensureNode checks for Node.js and tries to install the right version.
func (m *Manager) ensureNode(version string) error {
	// Check if node is already available
	currentVersion, err := m.getVersion("node", "--version")
	if err == nil && matchesVersion(currentVersion, version) {
		return nil
	}

	// Try version managers in order of preference
	managers := []struct {
		name    string
		check   string
		install string
	}{
		{"fnm", "fnm", fmt.Sprintf("fnm install %s && fnm use %s", version, version)},
		{"nvm", "nvm", fmt.Sprintf("nvm install %s && nvm use %s", version, version)},
		{"volta", "volta", fmt.Sprintf("volta install node@%s", version)},
	}

	for _, mgr := range managers {
		if _, err := exec.LookPath(mgr.check); err == nil {
			// Version manager found — use it
			cmd := shell.Command(mgr.install)
			if output, err := cmd.CombinedOutput(); err != nil {
				if m.verbose {
					fmt.Printf("  [%s] %s\n", mgr.name, string(output))
				}
				continue
			}
			return nil
		}
	}

	// No version manager found
	if currentVersion != "" {
		return fmt.Errorf(
			"Node.js %s found but %s required.\n\n  💡 Install a version manager (fnm, nvm, or volta) to manage Node.js versions:\n     https://github.com/Schniz/fnm",
			currentVersion, version,
		)
	}

	return fmt.Errorf(
		"Node.js is not installed.\n\n  💡 Install Node.js %s:\n     https://nodejs.org/ or use a version manager like fnm",
		version,
	)
}

// ensurePython checks for Python and tries to install the right version.
func (m *Manager) ensurePython(version string) error {
	// Check python3 first, then python
	for _, bin := range []string{"python3", "python"} {
		currentVersion, err := m.getVersion(bin, "--version")
		if err == nil && matchesVersion(currentVersion, version) {
			return nil
		}
	}

	// Try pyenv
	if _, err := exec.LookPath("pyenv"); err == nil {
		cmd := shell.Command(fmt.Sprintf("pyenv install -s %s && pyenv local %s", version, version))
		if _, err := cmd.CombinedOutput(); err == nil {
			return nil
		}
	}

	return fmt.Errorf(
		"Python %s is not available.\n\n  💡 Install Python %s:\n     https://www.python.org/downloads/ or use pyenv:\n     https://github.com/pyenv/pyenv",
		version, version,
	)
}

// getVersion runs a command to get the installed version string.
func (m *Manager) getVersion(binary string, args ...string) (string, error) {
	cmd := exec.Command(binary, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// matchesVersion checks if the installed version satisfies the required version spec.
// Supports exact match ("20.11.0"), major.x ("20.x"), and major.minor ("3.11").
func matchesVersion(installed, required string) bool {
	// Normalize: remove leading "v" or "Python " prefix
	installed = strings.TrimPrefix(installed, "v")
	installed = strings.TrimPrefix(installed, "Python ")
	installed = strings.TrimSpace(installed)

	// Handle "x" wildcard in required version (e.g. "20.x")
	if strings.HasSuffix(required, ".x") {
		prefix := strings.TrimSuffix(required, ".x")
		return strings.HasPrefix(installed, prefix+".")
	}

	// Handle partial version match (e.g. "3.11" matches "3.11.5")
	return strings.HasPrefix(installed, required)
}
