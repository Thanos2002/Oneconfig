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
		resolvedPort := resolvePort(
			extractPortFromRegexFile(projectDir, []string{"config/puma.rb"}, `(?m)^\s*port\s+(?:ENV\.fetch\(['"]PORT['"]\)\s*\{\s*)?(\d+)`),
			extractPortFromEnv(projectDir),
			3000,
		)
		result.addService(config.Service{
			Name:         "web",
			StartCommand: fmt.Sprintf("bundle exec rails server -p %d", resolvedPort),
			Port:         resolvedPort,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: fmt.Sprintf("http://localhost:%d", resolvedPort),
			},
		})
		result.Findings = append(result.Findings, "  Detected Rails framework")
	} else if strings.Contains(content, "sinatra") {
		resolvedPort := resolvePort(
			extractPortFromEnv(projectDir),
			4567,
		)
		result.addService(config.Service{
			Name:         "web",
			StartCommand: "bundle exec ruby app.rb",
			Port:         resolvedPort,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: fmt.Sprintf("http://localhost:%d", resolvedPort),
			},
		})
		result.Findings = append(result.Findings, "  Detected Sinatra framework")
	}

	return nil
}
