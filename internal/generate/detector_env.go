package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// EnvFileDetector — parses .env template files and extracts variables
// ---------------------------------------------------------------------------

// EnvFileDetector detects and parses environment template files.
type EnvFileDetector struct{}

func (d *EnvFileDetector) Detect(projectDir string, result *ScanResult) error {
	envFiles := []string{".env.example", ".env.sample", ".env.template", ".env.development"}

	for _, f := range envFiles {
		path := filepath.Join(projectDir, f)
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		result.Findings = append(result.Findings, fmt.Sprintf("Found %s → parsing environment variables", f))

		count := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Parse KEY=value
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes from value
			value = strings.Trim(value, `"'`)

			if key != "" {
				result.addEnvVar(key, value)
				count++
			}
		}
		file.Close()

		result.Findings = append(result.Findings, fmt.Sprintf("  Extracted %d environment variables from %s", count, f))
		break // only process the first env file found
	}

	return nil
}
