package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

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
