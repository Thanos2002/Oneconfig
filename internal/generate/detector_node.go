package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

// ---------------------------------------------------------------------------
// NodeDetector — detects Node.js projects from package.json and lockfiles
// ---------------------------------------------------------------------------

// NodeDetector detects Node.js projects.
type NodeDetector struct{}

// packageJSON is a minimal representation of a package.json file.
type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Engines         struct {
		Node string `json:"node"`
	} `json:"engines"`
}

func (d *NodeDetector) Detect(projectDir string, result *ScanResult) error {
	d.detectAt(projectDir, ".", result)
	return nil
}

func (d *NodeDetector) detectAt(projectDir, relPath string, result *ScanResult) {
	dir := filepath.Join(projectDir, relPath)
	pkgJSONPath := filepath.Join(dir, "package.json")

	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		return
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		result.Findings = append(result.Findings, fmt.Sprintf("Found package.json in %s but failed to parse: %s", relPath, err))
		return
	}

	desc := relPath
	if relPath == "." {
		desc = "root"
	}
	result.Findings = append(result.Findings, fmt.Sprintf("Found package.json in %s → Node.js project", desc))

	// Detect Node.js version
	version := "lts"
	if pkg.Engines.Node != "" {
		version = pkg.Engines.Node
		result.Findings = append(result.Findings, fmt.Sprintf("  engines.node = %q", version))
	} else {
		// Check for .nvmrc or .node-version
		for _, vFile := range []string{".nvmrc", ".node-version"} {
			vPath := filepath.Join(dir, vFile)
			if vData, err := os.ReadFile(vPath); err == nil {
				v := strings.TrimSpace(string(vData))
				if v != "" {
					version = v
					result.Findings = append(result.Findings, fmt.Sprintf("  Found %s → version %s", vFile, v))
					break
				}
			}
		}
	}
	result.addRuntime("node", version)

	// Detect package manager
	pmType := "npm"
	hasLocalLockfile := false

	// Check local directory first
	if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
		pmType = "yarn"
		hasLocalLockfile = true
		result.Findings = append(result.Findings, fmt.Sprintf("  Found yarn.lock → using yarn"))
	} else if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
		pmType = "pnpm"
		hasLocalLockfile = true
		result.Findings = append(result.Findings, fmt.Sprintf("  Found pnpm-lock.yaml → using pnpm"))
	} else if _, err := os.Stat(filepath.Join(dir, "bun.lockb")); err == nil {
		pmType = "bun"
		hasLocalLockfile = true
		result.Findings = append(result.Findings, fmt.Sprintf("  Found bun.lockb → using bun"))
	} else if _, err := os.Stat(filepath.Join(dir, "bun.lock")); err == nil {
		pmType = "bun"
		hasLocalLockfile = true
		result.Findings = append(result.Findings, fmt.Sprintf("  Found bun.lock → using bun"))
	} else if _, err := os.Stat(filepath.Join(dir, "package-lock.json")); err == nil {
		pmType = "npm"
		hasLocalLockfile = true
		result.Findings = append(result.Findings, fmt.Sprintf("  Found package-lock.json → using npm"))
	}

	// If no local lockfile, check root directory for workspace lockfiles
	if !hasLocalLockfile && relPath != "." {
		if _, err := os.Stat(filepath.Join(projectDir, "yarn.lock")); err == nil {
			pmType = "yarn"
			result.Findings = append(result.Findings, fmt.Sprintf("  No local lockfile, but root has yarn.lock → workspace using yarn"))
		} else if _, err := os.Stat(filepath.Join(projectDir, "pnpm-lock.yaml")); err == nil {
			pmType = "pnpm"
			result.Findings = append(result.Findings, fmt.Sprintf("  No local lockfile, but root has pnpm-lock.yaml → workspace using pnpm"))
		} else if _, err := os.Stat(filepath.Join(projectDir, "bun.lockb")); err == nil {
			pmType = "bun"
			result.Findings = append(result.Findings, fmt.Sprintf("  No local lockfile, but root has bun.lockb → workspace using bun"))
		} else if _, err := os.Stat(filepath.Join(projectDir, "bun.lock")); err == nil {
			pmType = "bun"
			result.Findings = append(result.Findings, fmt.Sprintf("  No local lockfile, but root has bun.lock → workspace using bun"))
		} else if _, err := os.Stat(filepath.Join(projectDir, "package-lock.json")); err == nil {
			pmType = "npm"
			result.Findings = append(result.Findings, fmt.Sprintf("  No local lockfile, but root has package-lock.json → workspace using npm"))
		} else {
			result.Findings = append(result.Findings, fmt.Sprintf("  No lockfile detected → defaulting to npm"))
			hasLocalLockfile = true // Force adding it since we don't have a workspace
		}
	} else if !hasLocalLockfile && relPath == "." {
		result.Findings = append(result.Findings, fmt.Sprintf("  No lockfile detected → defaulting to npm"))
		hasLocalLockfile = true
	}

	// Only add a package manager step if it's the root or it has its own lockfile (not part of a workspace)
	if hasLocalLockfile || relPath == "." {
		result.addPackageManager(pmType, relPath)
	}

	// Detect service from scripts
	svc := d.inferService(pkg, relPath, pmType, result)
	if svc != nil {
		result.addService(*svc)
		result.Findings = append(result.Findings, fmt.Sprintf("  Inferred service %q on port %d", svc.Name, svc.Port))
	}
}

// inferService tries to determine a runnable service from package.json scripts.
func (d *NodeDetector) inferService(pkg packageJSON, relPath string, pmType string, result *ScanResult) *config.Service {
	// Determine service name
	name := pkg.Name
	if name == "" {
		if relPath == "." {
			name = "app"
		} else {
			name = filepath.Base(relPath)
		}
	}

	// Determine start command and port
	var startCmd string
	port := 3000

	// Check for framework-specific ports and commands
	if _, ok := pkg.Dependencies["next"]; ok {
		port = 3000
	} else if _, ok := pkg.DevDependencies["next"]; ok {
		port = 3000
	} else if _, ok := pkg.DevDependencies["vite"]; ok {
		port = 5173
	} else if _, ok := pkg.Dependencies["vite"]; ok {
		port = 5173
	}

	// Prefer "dev" script for development, fall back to "start"
	if script, ok := pkg.Scripts["dev"]; ok {
		startCmd = fmt.Sprintf("%s run dev", pmType)
		// Try to extract port from the script
		if p := extractPort(script); p > 0 {
			port = p
		}
	} else if script, ok := pkg.Scripts["start"]; ok {
		startCmd = fmt.Sprintf("%s start", pmType)
		if p := extractPort(script); p > 0 {
			port = p
		}
	} else {
		return nil // no runnable script
	}

	svc := &config.Service{
		Name:         name,
		StartCommand: startCmd,
		Port:         port,
		HealthCheck: &config.ServiceHealth{
			Type:   "http",
			Target: fmt.Sprintf("http://localhost:%d", port),
		},
	}
	if relPath != "." {
		svc.WorkingDir = relPath
	}

	// Database heuristics
	if _, ok := pkg.Dependencies["pg"]; ok {
		result.Findings = append(result.Findings, "[DB:postgres]")
	}
	if _, ok := pkg.Dependencies["mysql2"]; ok {
		result.Findings = append(result.Findings, "[DB:mysql]")
	}
	if _, ok := pkg.Dependencies["mongoose"]; ok {
		result.Findings = append(result.Findings, "[DB:mongo]")
	}
	if _, ok := pkg.Dependencies["redis"]; ok {
		result.Findings = append(result.Findings, "[DB:redis]")
	}

	hasPrisma := false
	if _, ok := pkg.Dependencies["prisma"]; ok {
		hasPrisma = true
	}
	if _, ok := pkg.DevDependencies["prisma"]; ok {
		hasPrisma = true
	}
	if hasPrisma {
		result.Findings = append(result.Findings, "[DB:postgres]") // Prisma defaults to Postgres commonly, but user can change
		result.Findings = append(result.Findings, "  Detected Prisma → adding setup steps")
		result.Config.SetupSteps = append(result.Config.SetupSteps, config.SetupStep{
			Name:       fmt.Sprintf("prisma-generate-%s", name),
			Command:    fmt.Sprintf("%s exec prisma generate", pmType),
			WorkingDir: relPath,
		})
		result.Config.SetupSteps = append(result.Config.SetupSteps, config.SetupStep{
			Name:       fmt.Sprintf("prisma-db-push-%s", name),
			Command:    fmt.Sprintf("%s exec prisma db push", pmType),
			WorkingDir: relPath,
			DependsOn:  []string{fmt.Sprintf("prisma-generate-%s", name)},
		})
	}

	return svc
}

// extractPort tries to find a --port or -p flag value in a command string.
var portRe = regexp.MustCompile(`(?:--port|--PORT|-p)\s+(\d+)`)

func extractPort(cmd string) int {
	m := portRe.FindStringSubmatch(cmd)
	if len(m) >= 2 {
		p, err := strconv.Atoi(m[1])
		if err == nil {
			return p
		}
	}
	return 0
}
