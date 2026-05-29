package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

// ---------------------------------------------------------------------------
// TaskRunnerDetector — detects Makefile and Procfile
// ---------------------------------------------------------------------------

type TaskRunnerDetector struct{}

func (d *TaskRunnerDetector) Detect(projectDir string, result *ScanResult) error {
	// Parse Makefile for simple run commands
	makefile := filepath.Join(projectDir, "Makefile")
	if data, err := os.ReadFile(makefile); err == nil {
		content := string(data)
		lines := strings.Split(content, "\n")
		hasRun := false
		for _, line := range lines {
			if strings.HasPrefix(line, "run:") || strings.HasPrefix(line, "start:") || strings.HasPrefix(line, "dev:") {
				hasRun = true
				break
			}
		}
		if hasRun {
			result.Findings = append(result.Findings, "Found Makefile with run/start/dev target")
			// We don't blindly add it as a service since a framework detector might have already added a better one
		}
	}

	// Parse Procfile
	procfile := filepath.Join(projectDir, "Procfile")
	if data, err := os.ReadFile(procfile); err == nil {
		result.Findings = append(result.Findings, "Found Procfile → extracting services")
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				cmd := strings.TrimSpace(parts[1])

				port := extractPort(cmd)
				if port == 0 && name == "web" {
					port = 8080
				}

				svc := config.Service{
					Name:         name,
					StartCommand: cmd,
				}
				if port > 0 {
					svc.Port = port
					svc.HealthCheck = &config.ServiceHealth{
						Type:   "http",
						Target: fmt.Sprintf("http://localhost:%d", port),
					}
				}
				result.addService(svc)
				result.Findings = append(result.Findings, fmt.Sprintf("  Extracted service %q from Procfile", name))
			}
		}
	}

	return nil
}
