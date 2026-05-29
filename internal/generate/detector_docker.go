package generate

import (
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
// DockerfileDetector — fallback to extract exposed ports from Dockerfile
// ---------------------------------------------------------------------------

type DockerfileDetector struct{}

func (d *DockerfileDetector) Detect(projectDir string, result *ScanResult) error {
	// Only add Dockerfile if we haven't found any other services
	if len(result.Config.Services) > 0 {
		return nil
	}

	dockerfile := filepath.Join(projectDir, "Dockerfile")
	data, err := os.ReadFile(dockerfile)
	if err != nil {
		return nil
	}

	result.Findings = append(result.Findings, "Found Dockerfile → no other services detected, using as fallback")

	port := 8080
	re := regexp.MustCompile(`(?m)^EXPOSE\s+(\d+)`)
	if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
		if p, err := strconv.Atoi(m[1]); err == nil {
			port = p
			result.Findings = append(result.Findings, fmt.Sprintf("  Extracted EXPOSE port %d", port))
		}
	}

	result.addService(config.Service{
		Name:         "app",
		Type:         "docker",
		StartCommand: fmt.Sprintf("docker build -t app . && docker run --rm -p %[1]d:%[1]d app", port),
		Port:         port,
		HealthCheck: &config.ServiceHealth{
			Type:   "tcp",
			Target: fmt.Sprintf("%d", port),
		},
	})

	return nil
}
