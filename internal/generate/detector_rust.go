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
// RustDetector — detects Rust projects from Cargo.toml
// ---------------------------------------------------------------------------

type RustDetector struct{}

func (d *RustDetector) Detect(projectDir string, result *ScanResult) error {
	cargoPath := filepath.Join(projectDir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); err != nil {
		return nil
	}

	result.Findings = append(result.Findings, "Found Cargo.toml → Rust project")
	result.addRuntime("rust", "stable")
	result.addPackageManager("cargo", ".")

	// Check if it has a binary target
	data, err := os.ReadFile(cargoPath)
	if err == nil {
		content := string(data)
		if strings.Contains(content, "[[bin]]") || !strings.Contains(content, "[lib]") {
			name := "rust-app"
			re := regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
			if m := re.FindStringSubmatch(content); len(m) > 1 {
				name = m[1]
			}
			result.addService(config.Service{
				Name:         name,
				StartCommand: "cargo run",
				Port:         8080,
				HealthCheck: &config.ServiceHealth{
					Type:   "http",
					Target: "http://localhost:8080",
				},
			})
			result.Findings = append(result.Findings, fmt.Sprintf("  Inferred service %q", name))
		}
	}
	return nil
}
