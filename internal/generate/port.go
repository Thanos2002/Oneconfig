package generate

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// resolvePort returns the first non-zero port from the given layers, ordered by highest priority first.
func resolvePort(layers ...int) int {
	for _, p := range layers {
		if p > 0 {
			return p
		}
	}
	return 0
}

// extractPortFromEnv checks standard environment files for a PORT variable.
func extractPortFromEnv(projectDir string) int {
	envFiles := []string{".env", ".env.development", ".env.local", ".env.example", ".env.sample"}

	for _, f := range envFiles {
		path := filepath.Join(projectDir, f)
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			if key == "PORT" {
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, `"'`)
				if p, err := strconv.Atoi(value); err == nil {
					file.Close()
					return p
				}
			}
		}
		file.Close()
	}
	return 0
}

// extractPortFromScript tries to find a port in a command string using common patterns.
func extractPortFromScript(cmd string) int {
	patterns := []string{
		`(?:--port|--PORT|-p)\s+(\d+)`,
		`PORT=(\d+)`,
		`--listen\s+(\d+)`,
		`(?::|=)(\d+)(?:\s|$)`, // e.g. "localhost:3000" or "--port=3000"
	}

	for _, pat := range patterns {
		re := regexp.MustCompile(pat)
		m := re.FindStringSubmatch(cmd)
		if len(m) >= 2 {
			p, err := strconv.Atoi(m[1])
			if err == nil {
				return p
			}
		}
	}
	return 0
}

// extractPortFromViteConfig parses vite.config.* for server.port.
func extractPortFromViteConfig(projectDir string) int {
	return extractPortFromRegexFile(projectDir, []string{"vite.config.ts", "vite.config.js", "vite.config.mts", "vite.config.mjs"}, `(?m)port\s*:\s*(\d+)`)
}

// extractPortFromNextConfig parses next.config.* for experimental server port (if any) or similar.
func extractPortFromNextConfig(projectDir string) int {
	return extractPortFromRegexFile(projectDir, []string{"next.config.ts", "next.config.js", "next.config.mjs"}, `(?m)port\s*:\s*(\d+)`)
}

// extractPortFromSpringConfig parses application.properties or application.yml for server.port.
func extractPortFromSpringConfig(projectDir string) int {
	propertiesPath := filepath.Join(projectDir, "src", "main", "resources", "application.properties")
	yamlPath := filepath.Join(projectDir, "src", "main", "resources", "application.yml")

	// Check properties
	if data, err := os.ReadFile(propertiesPath); err == nil {
		re := regexp.MustCompile(`(?m)^server\.port\s*=\s*(\d+)`)
		if m := re.FindStringSubmatch(string(data)); len(m) >= 2 {
			if p, err := strconv.Atoi(m[1]); err == nil {
				return p
			}
		}
	}

	// Check yaml
	if data, err := os.ReadFile(yamlPath); err == nil {
		re := regexp.MustCompile(`(?m)^server:\s*\n(?:\s+.*)*\s+port:\s*(\d+)`)
		if m := re.FindStringSubmatch(string(data)); len(m) >= 2 {
			if p, err := strconv.Atoi(m[1]); err == nil {
				return p
			}
		}
		
		// simpler fallback if it's inline
		re2 := regexp.MustCompile(`(?m)^server\.port\s*:\s*(\d+)`)
		if m := re2.FindStringSubmatch(string(data)); len(m) >= 2 {
			if p, err := strconv.Atoi(m[1]); err == nil {
				return p
			}
		}
	}
	return 0
}

// extractPortFromRegexFile checks a list of files for a specific regex pattern to extract a port.
func extractPortFromRegexFile(projectDir string, files []string, pattern string) int {
	re := regexp.MustCompile(pattern)
	for _, f := range files {
		path := filepath.Join(projectDir, f)
		data, err := os.ReadFile(path)
		if err == nil {
			if m := re.FindStringSubmatch(string(data)); len(m) >= 2 {
				if p, err := strconv.Atoi(m[1]); err == nil {
					return p
				}
			}
		}
	}
	return 0
}
