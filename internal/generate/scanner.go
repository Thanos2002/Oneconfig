// Package generate scans an existing project to auto-detect its stack and
// generate a draft oneconfig.yml configuration file.
package generate

import (
	"path/filepath"

	"github.com/oneconfig/oneconfig/internal/config"
)

// Scanner analyzes a project directory to detect its technology stack.
type Scanner struct {
	projectDir string
	detectors  []Detector
}

// ScanResult holds the findings from a project scan.
type ScanResult struct {
	Config   config.Config // the generated configuration
	Findings []string      // human-readable findings for display
}

// Detector is the interface for individual project analysis strategies.
// Each detector is responsible for one concern (e.g. Node.js, Python, Docker Compose).
type Detector interface {
	// Detect analyzes the project directory and mutates result with its findings.
	Detect(projectDir string, result *ScanResult) error
}

// NewScanner creates a new project scanner with the default set of detectors.
func NewScanner(projectDir string) *Scanner {
	return &Scanner{
		projectDir: projectDir,
		detectors: []Detector{
			&NodeDetector{},
			&PythonDetector{},
			&GoDetector{},
			&RustDetector{},
			&RubyDetector{},
			&JavaDetector{},
			&MonorepoDetector{},
			&TaskRunnerDetector{},
			&EnvFileDetector{},
			&DockerComposeDetector{},
			&DockerfileDetector{},
			&DatabaseResolver{},
		},
	}
}

// Scan analyzes the project and returns detected configuration.
func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{
		Config: config.Config{
			ProjectName: filepath.Base(s.projectDir),
		},
	}

	for _, d := range s.detectors {
		if err := d.Detect(s.projectDir, result); err != nil {
			return nil, err
		}
	}

	if len(result.Findings) == 0 {
		result.Findings = append(result.Findings, "No project signals detected. Creating a minimal config.")
	}

	return result, nil
}

// addRuntime adds a runtime to the result if it's not already present.
func (r *ScanResult) addRuntime(name, version string) {
	for _, rt := range r.Config.Runtimes {
		if rt.Name == name {
			return
		}
	}
	r.Config.Runtimes = append(r.Config.Runtimes, config.Runtime{
		Name:    name,
		Version: version,
	})
}

// addPackageManager adds a package manager to the result, avoiding duplicates
// based on type+path.
func (r *ScanResult) addPackageManager(pmType, path string) {
	for _, pm := range r.Config.PackageManagers {
		if pm.Type == pmType && pm.Path == path {
			return
		}
	}
	r.Config.PackageManagers = append(r.Config.PackageManagers, config.PackageManager{
		Type: pmType,
		Path: path,
	})
}

// addService adds a service to the result, avoiding duplicates by name.
func (r *ScanResult) addService(svc config.Service) {
	for _, existing := range r.Config.Services {
		if existing.Name == svc.Name {
			return
		}
	}
	r.Config.Services = append(r.Config.Services, svc)
}

// addEnvVar adds an environment variable to the result.
func (r *ScanResult) addEnvVar(key, value string) {
	if r.Config.EnvVars == nil {
		r.Config.EnvVars = make(map[string]string)
	}
	// Don't overwrite if already set
	if _, exists := r.Config.EnvVars[key]; !exists {
		r.Config.EnvVars[key] = value
	}
}
