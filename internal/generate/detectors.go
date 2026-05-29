package generate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/oneconfig/oneconfig/internal/config"
	"gopkg.in/yaml.v3"
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

// ---------------------------------------------------------------------------
// PythonDetector — detects Python projects from requirements.txt, pyproject.toml, etc.
// ---------------------------------------------------------------------------

// PythonDetector detects Python projects.
type PythonDetector struct{}

func (d *PythonDetector) Detect(projectDir string, result *ScanResult) error {
	d.detectAt(projectDir, ".", result)
	return nil
}

func (d *PythonDetector) detectAt(projectDir, relPath string, result *ScanResult) {
	dir := filepath.Join(projectDir, relPath)

	// Try pyproject.toml first
	if d.detectPyproject(dir, relPath, result) {
		return
	}

	// Then requirements.txt
	if d.detectRequirements(dir, relPath, result) {
		return
	}

	// Then Pipfile
	if _, err := os.Stat(filepath.Join(dir, "Pipfile")); err == nil {
		result.Findings = append(result.Findings, fmt.Sprintf("Found Pipfile in %s → Python project (pipenv)", d.descPath(relPath)))
		result.addRuntime("python", d.detectPythonVersion(dir))
		result.addPackageManager("pip", relPath) // pipenv wraps pip
		return
	}

	// Then setup.py
	if _, err := os.Stat(filepath.Join(dir, "setup.py")); err == nil {
		result.Findings = append(result.Findings, fmt.Sprintf("Found setup.py in %s → Python project", d.descPath(relPath)))
		result.addRuntime("python", d.detectPythonVersion(dir))
		result.addPackageManager("pip", relPath)
		return
	}
}

func (d *PythonDetector) detectPyproject(dir, relPath string, result *ScanResult) bool {
	path := filepath.Join(dir, "pyproject.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	content := string(data)
	desc := d.descPath(relPath)

	// Determine the package manager / build system
	pmType := "pip"
	buildTool := "setuptools"

	if strings.Contains(content, "[tool.poetry]") {
		pmType = "poetry"
		buildTool = "poetry"
	} else if strings.Contains(content, "[tool.pdm]") {
		pmType = "pip" // PDM uses pip-compatible installs
		buildTool = "pdm"
	}

	// Check for uv.lock alongside
	if _, err := os.Stat(filepath.Join(dir, "uv.lock")); err == nil {
		pmType = "uv"
		result.Findings = append(result.Findings, fmt.Sprintf("  Found uv.lock → using uv"))
	}

	result.Findings = append(result.Findings, fmt.Sprintf("Found pyproject.toml in %s → Python project (%s)", desc, buildTool))

	// Try to extract Python version from pyproject.toml
	version := d.extractPythonVersionFromPyproject(content)
	if version == "" {
		version = d.detectPythonVersion(dir)
	}
	result.addRuntime("python", version)
	result.addPackageManager(pmType, relPath)

	d.processContent(dir, content, relPath, result)

	return true
}

func (d *PythonDetector) detectRequirements(dir, relPath string, result *ScanResult) bool {
	path := filepath.Join(dir, "requirements.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	desc := d.descPath(relPath)
	result.Findings = append(result.Findings, fmt.Sprintf("Found requirements.txt in %s → Python project (pip)", desc))

	version := d.detectPythonVersion(dir)
	result.addRuntime("python", version)

	// Check for uv.lock
	pmType := "pip"
	if _, err := os.Stat(filepath.Join(dir, "uv.lock")); err == nil {
		pmType = "uv"
		result.Findings = append(result.Findings, fmt.Sprintf("  Found uv.lock → using uv"))
	}
	result.addPackageManager(pmType, relPath)

	d.processContent(dir, string(data), relPath, result)
	return true
}

func (d *PythonDetector) processContent(dir, content, relPath string, result *ScanResult) {
	// Detect frameworks from content (requirements.txt, pyproject.toml, Pipfile)
	svc, np, nm, nr, nmo := d.inferServiceFromRequirements(dir, content, relPath)
	d.inferDBFromRequirements(result, np, nm, nr, nmo)

	if svc != nil {
		result.addService(*svc)
		result.Findings = append(result.Findings, fmt.Sprintf("  Inferred service %q on port %d", svc.Name, svc.Port))

		// Setup step for Django
		if strings.Contains(svc.StartCommand, "manage.py") {
			result.Findings = append(result.Findings, "  Detected Django → adding migration setup step")
			result.Config.SetupSteps = append(result.Config.SetupSteps, config.SetupStep{
				Name:       fmt.Sprintf("migrate-%s", svc.Name),
				Command:    "python manage.py migrate",
				WorkingDir: relPath,
			})
		}
	}
}

func (d *PythonDetector) findPythonApp(dir string, importStr string) string {
	var candidate string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(dir, entry.Name()))
			if err == nil {
				for _, subEntry := range subEntries {
					if !subEntry.IsDir() && filepath.Ext(subEntry.Name()) == ".py" {
						path := filepath.Join(dir, entry.Name(), subEntry.Name())
						data, err := os.ReadFile(path)
						if err == nil && strings.Contains(string(data), importStr) {
							return filepath.Join(entry.Name(), subEntry.Name())
						}
					}
				}
			}
			continue
		}

		if filepath.Ext(entry.Name()) != ".py" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		if strings.Contains(content, importStr) {
			return entry.Name()
		}

		if entry.Name() == "app.py" || entry.Name() == "main.py" {
			candidate = entry.Name()
		}
	}
	return candidate
}

// inferServiceFromRequirements tries to determine a service from Python dependencies.
func (d *PythonDetector) inferServiceFromRequirements(dir, content, relPath string) (*config.Service, bool, bool, bool, bool) {
	lines := strings.Split(strings.ToLower(content), "\n")

	hasFastAPI := false
	hasUvicorn := false
	hasFlask := false
	hasDjango := false
	hasStreamlit := false
	hasGradio := false
	needsPostgres := false
	needsMySQL := false
	needsRedis := false
	needsMongo := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") || line == "" {
			continue
		}
		pkg := strings.Split(line, "==")[0]
		pkg = strings.Split(pkg, ">=")[0]
		pkg = strings.Split(pkg, "<=")[0]
		pkg = strings.Split(pkg, "~=")[0]
		pkg = strings.Split(pkg, "!=")[0]
		pkg = strings.Split(pkg, "=")[0]
		pkg = strings.TrimSpace(pkg)
		// strip quotes if any
		pkg = strings.Trim(pkg, `"'`)

		switch pkg {
		case "fastapi":
			hasFastAPI = true
		case "uvicorn":
			hasUvicorn = true
		case "flask":
			hasFlask = true
		case "django":
			hasDjango = true
		case "streamlit":
			hasStreamlit = true
		case "gradio":
			hasGradio = true
		case "psycopg2", "psycopg2-binary", "asyncpg":
			needsPostgres = true
		case "mysqlclient", "pymysql":
			needsMySQL = true
		case "redis":
			needsRedis = true
		case "pymongo":
			needsMongo = true
		}
	}

	name := filepath.Base(relPath)
	if relPath == "." {
		name = "api"
	}

	var svc *config.Service

	if hasFastAPI || hasUvicorn {
		cmd := "python -m uvicorn main:app --port 8000"
		if hasFastAPI && !hasUvicorn {
			cmd = "python -m fastapi run --port 8000"
		}
		svc = &config.Service{
			Name:         name,
			StartCommand: cmd,
			Port:         8000,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:8000/health",
			},
		}
	} else if hasFlask {
		svc = &config.Service{
			Name:         name,
			StartCommand: "python -m flask run --port 5000",
			Port:         5000,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:5000",
			},
		}
	} else if hasDjango {
		svc = &config.Service{
			Name:         name,
			StartCommand: "python manage.py runserver 8000",
			Port:         8000,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:8000",
			},
		}
	} else if hasStreamlit {
		script := d.findPythonApp(dir, "import streamlit")
		if script == "" {
			script = "app.py"
		}
		svc = &config.Service{
			Name:         name,
			StartCommand: fmt.Sprintf("streamlit run %s", script),
			Port:         8501,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:8501",
			},
		}
	} else if hasGradio {
		script := d.findPythonApp(dir, "import gradio")
		if script == "" {
			script = "app.py"
		}
		svc = &config.Service{
			Name:         name,
			StartCommand: fmt.Sprintf("python %s", script),
			Port:         7860,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:7860",
			},
		}
	}

	if svc != nil && relPath != "." {
		svc.WorkingDir = relPath
	}

	return svc, needsPostgres, needsMySQL, needsRedis, needsMongo
}

// inferDBFromRequirements sets database finding flags based on parsed python dependencies
func (d *PythonDetector) inferDBFromRequirements(result *ScanResult, needsPostgres, needsMySQL, needsRedis, needsMongo bool) {
	if needsPostgres {
		result.Findings = append(result.Findings, "[DB:postgres]")
	}
	if needsMySQL {
		result.Findings = append(result.Findings, "[DB:mysql]")
	}
	if needsRedis {
		result.Findings = append(result.Findings, "[DB:redis]")
	}
	if needsMongo {
		result.Findings = append(result.Findings, "[DB:mongo]")
	}
}

// extractPythonVersionFromPyproject looks for python version constraints in pyproject.toml.
func (d *PythonDetector) extractPythonVersionFromPyproject(content string) string {
	// Look for requires-python = ">=3.11" or python = "^3.11"
	patterns := []string{
		`requires-python\s*=\s*"([^"]*)"`,
		`python\s*=\s*"([^"]*)"`,
	}
	for _, pat := range patterns {
		re := regexp.MustCompile(pat)
		m := re.FindStringSubmatch(content)
		if len(m) >= 2 {
			return cleanPythonVersion(m[1])
		}
	}
	return ""
}

// cleanPythonVersion extracts a version number from a version constraint.
func cleanPythonVersion(constraint string) string {
	// Strip operators like ^, ~, >=, <=, ==
	v := strings.TrimLeft(constraint, "^~><=!")
	v = strings.TrimSpace(v)
	// Take just major.minor
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

// detectPythonVersion looks for .python-version file.
func (d *PythonDetector) detectPythonVersion(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, ".python-version")); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v
		}
	}
	return "3.12" // reasonable default
}

func (d *PythonDetector) descPath(relPath string) string {
	if relPath == "." {
		return "root"
	}
	return relPath
}

// ---------------------------------------------------------------------------
// GoDetector — detects Go projects from go.mod
// ---------------------------------------------------------------------------

// GoDetector detects Go projects.
type GoDetector struct{}

func (d *GoDetector) Detect(projectDir string, result *ScanResult) error {
	goModPath := filepath.Join(projectDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	content := string(data)
	result.Findings = append(result.Findings, "Found go.mod → Go project")

	// Extract Go version
	version := "1.22"
	re := regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
	if m := re.FindStringSubmatch(content); len(m) >= 2 {
		version = m[1]
		result.Findings = append(result.Findings, fmt.Sprintf("  Go version: %s", version))
	}

	// Check for main.go → executable project
	if _, err := os.Stat(filepath.Join(projectDir, "main.go")); err == nil {
		result.Findings = append(result.Findings, "  Found main.go → executable project")
	}

	return nil
}

// ---------------------------------------------------------------------------
// EnvFileDetector — parses .env template files and extracts variables
// ---------------------------------------------------------------------------

// EnvFileDetector detects and parses environment template files.
type EnvFileDetector struct{}

func (d *EnvFileDetector) Detect(projectDir string, result *ScanResult) error {
	envFiles := []string{".env.example", ".env.sample", ".env.template", ".env.development"}

	for _, f := range envFiles {
		path := filepath.Join(projectDir, f)
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		result.Findings = append(result.Findings, fmt.Sprintf("Found %s → parsing environment variables", f))

		count := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Parse KEY=value
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes from value
			value = strings.Trim(value, `"'`)

			if key != "" {
				result.addEnvVar(key, value)
				count++
			}
		}
		file.Close()

		result.Findings = append(result.Findings, fmt.Sprintf("  Extracted %d environment variables from %s", count, f))
		break // only process the first env file found
	}

	return nil
}

// ---------------------------------------------------------------------------
// DockerComposeDetector — parses docker-compose.yml to extract services
// ---------------------------------------------------------------------------

// DockerComposeDetector detects and parses Docker Compose files.
type DockerComposeDetector struct{}

// composeFile is a minimal representation of a docker-compose.yml.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string   `yaml:"image"`
	Build       any      `yaml:"build"`
	Ports       []string `yaml:"ports"`
	Environment any      `yaml:"environment"`
	DependsOn   any      `yaml:"depends_on"`
	Command     string   `yaml:"command"`
}

func (d *DockerComposeDetector) Detect(projectDir string, result *ScanResult) error {
	composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

	for _, f := range composeFiles {
		path := filepath.Join(projectDir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		result.Findings = append(result.Findings, fmt.Sprintf("Found %s → extracting services", f))

		var cf composeFile
		if err := yaml.Unmarshal(data, &cf); err != nil {
			result.Findings = append(result.Findings, fmt.Sprintf("  Failed to parse %s: %s (manual review needed)", f, err))
			return nil
		}

		for svcName, svcDef := range cf.Services {
			svc := d.convertService(svcName, svcDef)
			result.addService(svc)
			result.Findings = append(result.Findings, fmt.Sprintf("  Extracted service %q (image: %s, port: %d)", svc.Name, svcDef.Image, svc.Port))
		}

		return nil // only process first compose file
	}

	return nil
}

func (d *DockerComposeDetector) convertService(name string, cs composeService) config.Service {
	svc := config.Service{
		Name: name,
		Type: "docker",
	}

	// Build start command
	if cs.Image != "" {
		cmd := fmt.Sprintf("docker run --rm")

		// Add port mappings
		for _, p := range cs.Ports {
			cmd += fmt.Sprintf(" -p %s", p)
		}

		// Add environment variables
		envVars := d.parseEnvironment(cs.Environment)
		for k, v := range envVars {
			cmd += fmt.Sprintf(" -e %s=%s", k, v)
		}

		cmd += fmt.Sprintf(" %s", cs.Image)

		if cs.Command != "" {
			cmd += fmt.Sprintf(" %s", cs.Command)
		}

		svc.StartCommand = cmd
	} else if cs.Build != nil {
		// Build-based service — generate a placeholder
		svc.StartCommand = fmt.Sprintf("docker compose up %s", name)
		svc.Type = "process"
	}

	// Extract first exposed port
	if len(cs.Ports) > 0 {
		port := d.parsePort(cs.Ports[0])
		svc.Port = port

		// Add a health check for the port
		svc.HealthCheck = &config.ServiceHealth{
			Type:   "tcp",
			Target: fmt.Sprintf("%d", port),
		}
	}

	// Extract depends_on
	svc.DependsOn = d.parseDependsOn(cs.DependsOn)

	return svc
}

// parseEnvironment handles both list and map formats of docker-compose environment.
func (d *DockerComposeDetector) parseEnvironment(env any) map[string]string {
	result := make(map[string]string)
	if env == nil {
		return result
	}

	switch e := env.(type) {
	case map[string]any:
		for k, v := range e {
			result[k] = fmt.Sprintf("%v", v)
		}
	case []any:
		for _, item := range e {
			s, ok := item.(string)
			if !ok {
				continue
			}
			parts := strings.SplitN(s, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
	}
	return result
}

// parsePort extracts the host port from a docker-compose port mapping like "5432:5432".
func (d *DockerComposeDetector) parsePort(portMapping string) int {
	// Handle "host:container" and "host:container/proto"
	parts := strings.Split(portMapping, ":")
	portStr := parts[0]
	if len(parts) > 1 {
		portStr = parts[0]
	}
	// Strip protocol suffix
	portStr = strings.Split(portStr, "/")[0]

	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		return 0
	}
	return port
}

// parseDependsOn handles both list and map formats.
func (d *DockerComposeDetector) parseDependsOn(dep any) []string {
	if dep == nil {
		return nil
	}

	switch d := dep.(type) {
	case []any:
		result := make([]string, 0, len(d))
		for _, item := range d {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case map[string]any:
		result := make([]string, 0, len(d))
		for k := range d {
			result = append(result, k)
		}
		return result
	}
	return nil
}

// ---------------------------------------------------------------------------
// MonorepoDetector — walks first-level subdirectories for project indicators
// ---------------------------------------------------------------------------

// MonorepoDetector detects components in subdirectories.
type MonorepoDetector struct{}

// projectIndicators maps indicator files to detector functions.
var projectIndicators = []string{
	"package.json",
	"requirements.txt",
	"pyproject.toml",
	"Pipfile",
	"go.mod",
	"Cargo.toml",
	"Gemfile",
	"pom.xml",
	"build.gradle",
}

func (d *MonorepoDetector) Detect(projectDir string, result *ScanResult) error {
	d.scanDirectory(projectDir, projectDir, result, 0)
	return nil
}

func (d *MonorepoDetector) scanDirectory(rootDir, currentDir string, result *ScanResult, depth int) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	nodeDetector := &NodeDetector{}
	pythonDetector := &PythonDetector{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden directories and common non-project dirs
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if name == "node_modules" || name == "vendor" || name == "__pycache__" || name == "dist" || name == "build" {
			continue
		}

		subdir := filepath.Join(currentDir, name)
		relPath, _ := filepath.Rel(rootDir, subdir)
		// Clean up windows paths to standard forward slashes for the config
		relPath = filepath.ToSlash(relPath)

		// Check if this is a known monorepo grouping directory (like 'packages' or 'apps')
		// We only want to dive into these if we're at the root level to avoid unbounded recursion
		if depth == 0 && (name == "packages" || name == "apps" || name == "services" || name == "projects") {
			d.scanDirectory(rootDir, subdir, result, depth+1)
			continue
		}

		// Check if this subdirectory has any project indicators
		for _, indicator := range projectIndicators {
			indicatorPath := filepath.Join(subdir, indicator)
			if _, err := os.Stat(indicatorPath); err != nil {
				continue
			}

			// Delegate to the right detector
			switch indicator {
			case "package.json":
				nodeDetector.detectAt(rootDir, relPath, result)
			case "requirements.txt", "pyproject.toml", "Pipfile":
				pythonDetector.detectAt(rootDir, relPath, result)
			default:
				result.Findings = append(result.Findings, fmt.Sprintf("Found %s in %s/ → detected sub-project (manual config needed)", indicator, relPath))
			}
			break // only process first indicator per subdirectory
		}
	}
}
