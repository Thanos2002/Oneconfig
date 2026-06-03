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
	re := regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
	if m := re.FindStringSubmatch(content); len(m) >= 2 {
		version := m[1]
		result.Findings = append(result.Findings, fmt.Sprintf("  Go version: %s", version))
	}

	// Add package manager
	result.addPackageManager("go", ".")

	// Check for main.go → executable project
	hasMain := false
	if _, err := os.Stat(filepath.Join(projectDir, "main.go")); err == nil {
		hasMain = true
		result.Findings = append(result.Findings, "  Found main.go → executable project")
	}

	// Detect web frameworks
	port := 0
	if hasMain {
		if strings.Contains(content, "github.com/gin-gonic/gin") {
			port = 8080
			result.Findings = append(result.Findings, "  Detected Gin framework")
		} else if strings.Contains(content, "github.com/labstack/echo") {
			port = 8080
			result.Findings = append(result.Findings, "  Detected Echo framework")
		} else if strings.Contains(content, "github.com/gorilla/mux") {
			port = 8080
			result.Findings = append(result.Findings, "  Detected Gorilla Mux")
		} else if strings.Contains(content, "github.com/go-chi/chi") {
			port = 8080
			result.Findings = append(result.Findings, "  Detected Chi framework")
		} else if strings.Contains(content, "github.com/gofiber/fiber") {
			port = 3000
			result.Findings = append(result.Findings, "  Detected Fiber framework")
		} else {
			// Check if net/http is imported in main.go
			mainContent, err := os.ReadFile(filepath.Join(projectDir, "main.go"))
			if err == nil && strings.Contains(string(mainContent), `"net/http"`) {
				port = 8080
				result.Findings = append(result.Findings, "  Detected net/http in main.go")
			}
		}
	}

	if port > 0 {
		resolvedPort := resolvePort(
			extractPortFromEnv(projectDir),
			port,
		)
		name := filepath.Base(projectDir)
		if name == "." || name == "" {
			name = "go-app"
		}
		
		result.addService(config.Service{
			Name:         name,
			StartCommand: "go run main.go",
			Port:         resolvedPort,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: fmt.Sprintf("http://localhost:%d", resolvedPort),
			},
		})
	}

	return nil
}
