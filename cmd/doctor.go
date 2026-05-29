package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/Thanos2002/Oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system for missing tools and configuration issues",
	Long: `Runs a series of pre-flight checks to verify that your system has all the
tools needed to run OneConfig projects. Reports missing dependencies with
install instructions for your OS.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// toolCheck describes a tool to verify.
type toolCheck struct {
	Name        string
	Command     string // binary to look up on PATH
	Required    bool   // false = optional
	InstallHint string // how to install if missing
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ui.Header("System Doctor")
	ui.Info(fmt.Sprintf("OS: %s/%s", runtime.GOOS, runtime.GOARCH))
	fmt.Println()

	checks := []toolCheck{
		{Name: "Git", Command: "git", Required: true, InstallHint: installHint("git")},
		{Name: "Node.js", Command: "node", Required: false, InstallHint: installHint("node")},
		{Name: "npm", Command: "npm", Required: false, InstallHint: "Included with Node.js"},
		{Name: "yarn", Command: "yarn", Required: false, InstallHint: "npm install -g yarn"},
		{Name: "pnpm", Command: "pnpm", Required: false, InstallHint: "npm install -g pnpm"},
		{Name: "Python", Command: "python3", Required: false, InstallHint: installHint("python")},
		{Name: "pip", Command: "pip3", Required: false, InstallHint: "Included with Python"},
		{Name: "poetry", Command: "poetry", Required: false, InstallHint: "pip install poetry"},
		{Name: "Docker", Command: "docker", Required: false, InstallHint: installHint("docker")},
		{Name: "nvm", Command: "nvm", Required: false, InstallHint: "https://github.com/nvm-sh/nvm"},
		{Name: "pyenv", Command: "pyenv", Required: false, InstallHint: "https://github.com/pyenv/pyenv"},
	}

	var issues int
	headers := []string{"Tool", "Status", ""}
	var rows [][]string

	for _, check := range checks {
		path, err := exec.LookPath(check.Command)
		if err != nil {
			status := "missing"
			if check.Required {
				status = "REQUIRED"
				issues++
			}
			rows = append(rows, []string{
				check.Name,
				ui.StatusBadge("error"),
				fmt.Sprintf("%s — %s", status, check.InstallHint),
			})
		} else {
			rows = append(rows, []string{
				check.Name,
				ui.StatusBadge("ready"),
				path,
			})
		}
	}

	ui.Table(headers, rows)
	fmt.Println()

	if issues > 0 {
		ui.Warning(fmt.Sprintf("%d required tool(s) missing. Install them and run 'oneconfig doctor' again.", issues))
		return fmt.Errorf("%d required tools missing", issues)
	}

	ui.Success("All required tools are installed!")
	return nil
}

func installHint(tool string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("brew install %s", tool)
	case "linux":
		return fmt.Sprintf("Check your package manager or https://oneconfig.dev/docs/install#%s", tool)
	case "windows":
		return fmt.Sprintf("winget install %s or https://oneconfig.dev/docs/install#%s", tool, tool)
	default:
		return fmt.Sprintf("https://oneconfig.dev/docs/install#%s", tool)
	}
}
