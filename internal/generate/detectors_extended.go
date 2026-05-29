package generate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/oneconfig/oneconfig/internal/config"
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

// ---------------------------------------------------------------------------
// DockerfileDetector — fallback to extract exposed ports from Dockerfile
// ---------------------------------------------------------------------------

type DockerfileDetector struct{}

func (d *DockerfileDetector) Detect(projectDir string, result *ScanResult) error {
	// Only add Dockerfile if we haven't found any other services
	if len(result.Config.Services) > 0 {
		return nil
	}

	dockerfile := filepath.Join(projectDir, "Dockerfile")
	data, err := os.ReadFile(dockerfile)
	if err != nil {
		return nil
	}

	result.Findings = append(result.Findings, "Found Dockerfile → no other services detected, using as fallback")

	port := 8080
	re := regexp.MustCompile(`(?m)^EXPOSE\s+(\d+)`)
	if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
		if p, err := strconv.Atoi(m[1]); err == nil {
			port = p
			result.Findings = append(result.Findings, fmt.Sprintf("  Extracted EXPOSE port %d", port))
		}
	}

	result.addService(config.Service{
		Name:         "app",
		Type:         "docker",
		StartCommand: fmt.Sprintf("docker build -t app . && docker run --rm -p %[1]d:%[1]d app", port),
		Port:         port,
		HealthCheck: &config.ServiceHealth{
			Type:   "tcp",
			Target: fmt.Sprintf("%d", port),
		},
	})

	return nil
}

// ---------------------------------------------------------------------------
// DatabaseResolver — analyzes findings and injects database services
// ---------------------------------------------------------------------------

type DatabaseResolver struct{}

func (d *DatabaseResolver) Detect(projectDir string, result *ScanResult) error {
	needsPostgres := false
	needsRedis := false
	needsMongo := false
	needsMySQL := false

	// Determine DB requirements from Findings
	// This is a simple heuristic based on the flags we will set in NodeDetector and PythonDetector
	for _, f := range result.Findings {
		if strings.Contains(f, "[DB:postgres]") {
			needsPostgres = true
		}
		if strings.Contains(f, "[DB:redis]") {
			needsRedis = true
		}
		if strings.Contains(f, "[DB:mongo]") {
			needsMongo = true
		}
		if strings.Contains(f, "[DB:mysql]") {
			needsMySQL = true
		}
	}

	// Also infer based on existing env vars if they were extracted from .env
	if result.Config.EnvVars != nil {
		if url, ok := result.Config.EnvVars["DATABASE_URL"]; ok {
			if strings.HasPrefix(url, "postgres") {
				needsPostgres = true
			} else if strings.HasPrefix(url, "mysql") {
				needsMySQL = true
			}
		}
		if _, ok := result.Config.EnvVars["REDIS_URL"]; ok {
			needsRedis = true
		}
	}

	// Add database services
	if needsPostgres {
		result.Findings = append(result.Findings, "Database inference → injected PostgreSQL service")
		result.addService(config.Service{
			Name:         "postgres",
			Type:         "docker",
			StartCommand: "docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=app postgres:15",
			Port:         5432,
			HealthCheck: &config.ServiceHealth{
				Type:   "tcp",
				Target: "5432",
			},
		})
		result.addEnvVar("DATABASE_URL", "postgres://postgres:dev@localhost:5432/app")
	}

	if needsMySQL {
		result.Findings = append(result.Findings, "Database inference → injected MySQL service")
		result.addService(config.Service{
			Name:         "mysql",
			Type:         "docker",
			StartCommand: "docker run --rm -p 3306:3306 -e MYSQL_ROOT_PASSWORD=dev -e MYSQL_DATABASE=app mysql:8",
			Port:         3306,
			HealthCheck: &config.ServiceHealth{
				Type:   "tcp",
				Target: "3306",
			},
		})
		result.addEnvVar("DATABASE_URL", "mysql://root:dev@localhost:3306/app")
	}

	if needsRedis {
		result.Findings = append(result.Findings, "Database inference → injected Redis service")
		result.addService(config.Service{
			Name:         "redis",
			Type:         "docker",
			StartCommand: "docker run --rm -p 6379:6379 redis:7",
			Port:         6379,
			HealthCheck: &config.ServiceHealth{
				Type:   "tcp",
				Target: "6379",
			},
		})
		result.addEnvVar("REDIS_URL", "redis://localhost:6379")
	}

	if needsMongo {
		result.Findings = append(result.Findings, "Database inference → injected MongoDB service")
		result.addService(config.Service{
			Name:         "mongodb",
			Type:         "docker",
			StartCommand: "docker run --rm -p 27017:27017 mongo:6",
			Port:         27017,
			HealthCheck: &config.ServiceHealth{
				Type:   "tcp",
				Target: "27017",
			},
		})
		result.addEnvVar("MONGO_URL", "mongodb://localhost:27017/app")
	}

	// Link existing app services to the databases
	dbDeps := []string{}
	if needsPostgres {
		dbDeps = append(dbDeps, "postgres")
	}
	if needsMySQL {
		dbDeps = append(dbDeps, "mysql")
	}
	if needsRedis {
		dbDeps = append(dbDeps, "redis")
	}
	if needsMongo {
		dbDeps = append(dbDeps, "mongodb")
	}

	if len(dbDeps) > 0 {
		for i, svc := range result.Config.Services {
			if svc.Type != "docker" && svc.Type != "script" && svc.Name != "postgres" && svc.Name != "mysql" && svc.Name != "redis" && svc.Name != "mongodb" {
				// Prevent duplicate dependencies
				existingDeps := make(map[string]bool)
				for _, dep := range svc.DependsOn {
					existingDeps[dep] = true
				}
				for _, newDep := range dbDeps {
					if !existingDeps[newDep] {
						result.Config.Services[i].DependsOn = append(result.Config.Services[i].DependsOn, newDep)
					}
				}
			}
		}
	}

	return nil
}
