package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oneconfig/oneconfig/internal/config"
)

// ---------------------------------------------------------------------------
// RubyDetector — detects Ruby projects from Gemfile
// ---------------------------------------------------------------------------

type RubyDetector struct{}

func (d *RubyDetector) Detect(projectDir string, result *ScanResult) error {
	gemfilePath := filepath.Join(projectDir, "Gemfile")
	data, err := os.ReadFile(gemfilePath)
	if err != nil {
		return nil
	}

	result.Findings = append(result.Findings, "Found Gemfile → Ruby project")

	version := "3.3"
	re := regexp.MustCompile(`ruby\s+['"]([^'"]+)['"]`)
	if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
		version = m[1]
		result.Findings = append(result.Findings, fmt.Sprintf("  Ruby version: %s", version))
	}
	result.addRuntime("ruby", version)
	result.addPackageManager("bundler", ".")

	content := string(data)
	if strings.Contains(content, "rails") {
		result.addService(config.Service{
			Name:         "web",
			StartCommand: "bundle exec rails server -p 3000",
			Port:         3000,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:3000",
			},
		})
		result.Findings = append(result.Findings, "  Detected Rails framework")
	} else if strings.Contains(content, "sinatra") {
		result.addService(config.Service{
			Name:         "web",
			StartCommand: "bundle exec ruby app.rb",
			Port:         4567,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:4567",
			},
		})
		result.Findings = append(result.Findings, "  Detected Sinatra framework")
	}

	return nil
}
