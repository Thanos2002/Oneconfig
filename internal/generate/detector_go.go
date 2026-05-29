package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	version := "1.22"
	re := regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
	if m := re.FindStringSubmatch(content); len(m) >= 2 {
		version = m[1]
		result.Findings = append(result.Findings, fmt.Sprintf("  Go version: %s", version))
	}

	// Check for main.go → executable project
	if _, err := os.Stat(filepath.Join(projectDir, "main.go")); err == nil {
		result.Findings = append(result.Findings, "  Found main.go → executable project")
	}

	return nil
}
