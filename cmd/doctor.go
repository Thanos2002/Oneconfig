package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Thanos2002/Oneconfig/internal/config"
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

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.LoadFromPath(dir, ConfigFilePath())
	var checks []toolCheck

	if err == nil {
		ui.Info(fmt.Sprintf("Found configuration for project %q. Checking required tools...", cfg.ProjectName))
		checks = generateContextAwareChecks(cfg)
	} else {
		ui.Info("No valid oneconfig.yml found. Running generic system check...")
		checks = []toolCheck{
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

func generateContextAwareChecks(cfg *config.Config) []toolCheck {
	var checks []toolCheck

	// Git is always useful
	checks = append(checks, toolCheck{Name: "Git", Command: "git", Required: true, InstallHint: installHint("git")})

	// Add runtimes
	for _, rt := range cfg.Runtimes {
		switch rt.Name {
		case "node":
			checks = append(checks, toolCheck{Name: "Node.js", Command: "node", Required: true, InstallHint: installHint("node")})
		case "python":
			cmd := "python3"
			if runtime.GOOS == "windows" {
				cmd = "python"
			}
			checks = append(checks, toolCheck{Name: "Python", Command: cmd, Required: true, InstallHint: installHint("python")})
		case "go":
			checks = append(checks, toolCheck{Name: "Go", Command: "go", Required: true, InstallHint: installHint("go")})
		case "rust":
			checks = append(checks, toolCheck{Name: "Rust", Command: "cargo", Required: true, InstallHint: installHint("rust")})
		case "ruby":
			checks = append(checks, toolCheck{Name: "Ruby", Command: "ruby", Required: true, InstallHint: installHint("ruby")})
		case "java":
			checks = append(checks, toolCheck{Name: "Java", Command: "java", Required: true, InstallHint: installHint("java")})
		}
	}

	// Add package managers
	for _, pm := range cfg.PackageManagers {
		switch pm.Type {
		case "npm":
			checks = append(checks, toolCheck{Name: "npm", Command: "npm", Required: true, InstallHint: "Included with Node.js"})
		case "yarn":
			checks = append(checks, toolCheck{Name: "yarn", Command: "yarn", Required: true, InstallHint: "npm install -g yarn"})
		case "pnpm":
			checks = append(checks, toolCheck{Name: "pnpm", Command: "pnpm", Required: true, InstallHint: "npm install -g pnpm"})
		case "bun":
			checks = append(checks, toolCheck{Name: "bun", Command: "bun", Required: true, InstallHint: "npm install -g bun"})
		case "pip":
			cmd := "pip3"
			if runtime.GOOS == "windows" {
				cmd = "pip"
			}
			checks = append(checks, toolCheck{Name: "pip", Command: cmd, Required: true, InstallHint: "Included with Python"})
		case "poetry":
			checks = append(checks, toolCheck{Name: "poetry", Command: "poetry", Required: true, InstallHint: "pip install poetry"})
		case "uv":
			checks = append(checks, toolCheck{Name: "uv", Command: "uv", Required: true, InstallHint: "curl -LsSf https://astral.sh/uv/install.sh | sh"})
		case "gradle":
			checks = append(checks, toolCheck{Name: "Gradle", Command: "gradle", Required: true, InstallHint: installHint("gradle")})
		case "maven":
			checks = append(checks, toolCheck{Name: "Maven", Command: "mvn", Required: true, InstallHint: installHint("maven")})
		case "bundler":
			checks = append(checks, toolCheck{Name: "Bundler", Command: "bundle", Required: true, InstallHint: "gem install bundler"})
		case "cargo":
			checks = append(checks, toolCheck{Name: "Cargo", Command: "cargo", Required: true, InstallHint: installHint("cargo")})
		case "go":
			checks = append(checks, toolCheck{Name: "Go", Command: "go", Required: true, InstallHint: installHint("go")})
		}
	}

	// Check for docker in services or setup steps
	needsDocker := false
	for _, svc := range cfg.Services {
		if svc.Type == "docker" || strings.Contains(svc.StartCommand, "docker ") {
			needsDocker = true
			break
		}
	}
	if !needsDocker {
		for _, step := range cfg.SetupSteps {
			if strings.Contains(step.Command, "docker ") {
				needsDocker = true
				break
			}
		}
	}

	if needsDocker {
		checks = append(checks, toolCheck{Name: "Docker", Command: "docker", Required: true, InstallHint: installHint("docker")})
	}

	// Deduplicate checks (e.g. if pm is cargo and runtime is rust)
	var uniqueChecks []toolCheck
	seen := make(map[string]bool)
	for _, c := range checks {
		if !seen[c.Name] {
			seen[c.Name] = true
			uniqueChecks = append(uniqueChecks, c)
		}
	}

	return uniqueChecks
}

func installHint(tool string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("brew install %s", tool)
	case "linux":
		return "Check your package manager or https://github.com/Thanos2002/Oneconfig#-quick-start"
	case "windows":
		return fmt.Sprintf("winget install %s or https://github.com/Thanos2002/Oneconfig#-quick-start", tool)
	default:
		return "https://github.com/Thanos2002/Oneconfig#-quick-start"
	}
}
