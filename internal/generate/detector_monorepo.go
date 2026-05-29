package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

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

// ---------------------------------------------------------------------------
// MonorepoDetector — walks first-level subdirectories for project indicators
// ---------------------------------------------------------------------------

// MonorepoDetector detects components in subdirectories.
type MonorepoDetector struct{}

// projectIndicators maps indicator files to detector functions.
var projectIndicators = []string{
	"package.json",
	"requirements.txt",
	"pyproject.toml",
	"Pipfile",
	"go.mod",
	"Cargo.toml",
	"Gemfile",
	"pom.xml",
	"build.gradle",
}

func (d *MonorepoDetector) Detect(projectDir string, result *ScanResult) error {
	d.scanDirectory(projectDir, projectDir, result, 0)
	return nil
}

func (d *MonorepoDetector) scanDirectory(rootDir, currentDir string, result *ScanResult, depth int) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return
	}

	nodeDetector := &NodeDetector{}
	pythonDetector := &PythonDetector{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden directories and common non-project dirs
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if name == "node_modules" || name == "vendor" || name == "__pycache__" || name == "dist" || name == "build" {
			continue
		}

		subdir := filepath.Join(currentDir, name)
		relPath, _ := filepath.Rel(rootDir, subdir)
		// Clean up windows paths to standard forward slashes for the config
		relPath = filepath.ToSlash(relPath)

		// Check if this is a known monorepo grouping directory (like 'packages' or 'apps')
		// We only want to dive into these if we're at the root level to avoid unbounded recursion
		if depth == 0 && (name == "packages" || name == "apps" || name == "services" || name == "projects") {
			d.scanDirectory(rootDir, subdir, result, depth+1)
			continue
		}

		// Check if this subdirectory has any project indicators
		for _, indicator := range projectIndicators {
			indicatorPath := filepath.Join(subdir, indicator)
			if _, err := os.Stat(indicatorPath); err != nil {
				continue
			}

			// Delegate to the right detector
			switch indicator {
			case "package.json":
				nodeDetector.detectAt(rootDir, relPath, result)
			case "requirements.txt", "pyproject.toml", "Pipfile":
				pythonDetector.detectAt(rootDir, relPath, result)
			default:
				result.Findings = append(result.Findings, fmt.Sprintf("Found %s in %s/ → detected sub-project (manual config needed)", indicator, relPath))
			}
			break // only process first indicator per subdirectory
		}
	}
}
