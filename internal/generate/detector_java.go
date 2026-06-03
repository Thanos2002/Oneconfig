package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

// ---------------------------------------------------------------------------
// JavaDetector — detects Java projects from pom.xml or build.gradle
// ---------------------------------------------------------------------------

type JavaDetector struct{}

func (d *JavaDetector) Detect(projectDir string, result *ScanResult) error {
	if _, err := os.Stat(filepath.Join(projectDir, "pom.xml")); err == nil {
		result.Findings = append(result.Findings, "Found pom.xml → Java project (Maven)")
		result.addRuntime("java", "21")
		result.addPackageManager("maven", ".")
		resolvedPort := resolvePort(
			extractPortFromSpringConfig(projectDir),
			extractPortFromEnv(projectDir),
			8080,
		)
		result.addService(config.Service{
			Name:         "app",
			StartCommand: "mvn spring-boot:run",
			Port:         resolvedPort,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: fmt.Sprintf("http://localhost:%d", resolvedPort),
			},
		})
		return nil
	}

	if _, err := os.Stat(filepath.Join(projectDir, "build.gradle")); err == nil {
		result.Findings = append(result.Findings, "Found build.gradle → Java project (Gradle)")
		result.addRuntime("java", "21")
		result.addPackageManager("gradle", ".")
		resolvedPort := resolvePort(
			extractPortFromSpringConfig(projectDir),
			extractPortFromEnv(projectDir),
			8080,
		)
		result.addService(config.Service{
			Name:         "app",
			StartCommand: "./gradlew bootRun",
			Port:         resolvedPort,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: fmt.Sprintf("http://localhost:%d", resolvedPort),
			},
		})
		return nil
	}

	return nil
}
