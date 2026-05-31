// Package pkgmanager runs package manager install commands for supported ecosystems.
package pkgmanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Thanos2002/Oneconfig/internal/config"
	"github.com/Thanos2002/Oneconfig/internal/shell"
)

// Runner executes package manager install commands.
type Runner struct {
	projectDir string
	verbose    bool
}

// NewRunner creates a new package manager runner.
func NewRunner(projectDir string, verbose bool) *Runner {
	return &Runner{projectDir: projectDir, verbose: verbose}
}

// Install runs the appropriate install command for the given package manager config.
func (r *Runner) Install(ctx context.Context, pm config.PackageManager) error {
	dir := filepath.Join(r.projectDir, pm.Path)

	// Verify the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory %s does not exist", dir)
	}

	// Determine the install command
	command := pm.InstallCommand
	if command == "" {
		command = defaultInstallCommand(pm.Type)
	}

	if command == "" {
		return fmt.Errorf("unsupported package manager type: %s", pm.Type)
	}

	// Run the command
	cmd := shell.CommandContext(ctx, command)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CI=true") // suppress interactive prompts

	if r.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s install failed in %s: %w\n\n  💡 Try running '%s' manually in %s to see the full error", pm.Type, dir, err, command, dir)
	}

	return nil
}

// defaultInstallCommand returns the default install command for a package manager type.
func defaultInstallCommand(pmType string) string {
	switch pmType {
	case "npm":
		return "npm install"
	case "yarn":
		return "yarn install --frozen-lockfile"
	case "pnpm":
		return "pnpm install --frozen-lockfile"
	case "pip":
		return "pip install -r requirements.txt"
	case "poetry":
		return "poetry install"
	case "uv":
		return "uv sync"
	default:
		return ""
	}
}
