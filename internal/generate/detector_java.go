package generate

import (
	"os"
	"path/filepath"

	"github.com/oneconfig/oneconfig/internal/config"
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
		result.addService(config.Service{
			Name:         "app",
			StartCommand: "mvn spring-boot:run",
			Port:         8080,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:8080",
			},
		})
		return nil
	}

	if _, err := os.Stat(filepath.Join(projectDir, "build.gradle")); err == nil {
		result.Findings = append(result.Findings, "Found build.gradle → Java project (Gradle)")
		result.addRuntime("java", "21")
		result.addPackageManager("gradle", ".")
		result.addService(config.Service{
			Name:         "app",
			StartCommand: "./gradlew bootRun",
			Port:         8080,
			HealthCheck: &config.ServiceHealth{
				Type:   "http",
				Target: "http://localhost:8080",
			},
		})
		return nil
	}

	return nil
}
