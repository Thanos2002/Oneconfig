package config

import (
	"testing"
)

func TestParse_ValidConfig(t *testing.T) {
	yaml := `
project_name: test-app
runtimes:
  - name: node
    version: "20.x"
package_managers:
  - type: npm
    path: .
env_vars:
  NODE_ENV: development
services:
  - name: api
    start_command: npm run dev
    port: 3000
    health_check:
      type: http
      target: http://localhost:3000/health
setup_steps:
  - name: migrate
    command: npm run migrate
    depends_on: [api]
health_checks:
  - name: app
    url: http://localhost:3000/health
    timeout: 30s
post_start_commands:
  - echo "ready"
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ProjectName != "test-app" {
		t.Errorf("expected project_name 'test-app', got %q", cfg.ProjectName)
	}
	if len(cfg.Runtimes) != 1 {
		t.Errorf("expected 1 runtime, got %d", len(cfg.Runtimes))
	}
	if cfg.Runtimes[0].Name != "node" {
		t.Errorf("expected runtime name 'node', got %q", cfg.Runtimes[0].Name)
	}
	if len(cfg.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(cfg.Services))
	}
	if cfg.Services[0].Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Services[0].Port)
	}
}

func TestParse_MissingProjectName(t *testing.T) {
	yaml := `
runtimes:
  - name: node
    version: "20.x"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing project_name")
	}
}

func TestParse_DuplicateServiceNames(t *testing.T) {
	yaml := `
project_name: test
services:
  - name: api
    start_command: npm start
  - name: api
    start_command: npm start
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for duplicate service names")
	}
}

func TestParse_InvalidDependsOn(t *testing.T) {
	yaml := `
project_name: test
services:
  - name: api
    start_command: npm start
    depends_on: [nonexistent]
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid depends_on reference")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	yaml := `
project_name: [invalid yaml
  this: is broken
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParse_MultipleHealthCheckTargets(t *testing.T) {
	yaml := `
project_name: test
health_checks:
  - name: bad
    url: http://localhost:3000
    port: 3000
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for multiple health check targets")
	}
}

func TestDefaults(t *testing.T) {
	cfg := &Config{
		ProjectName: "test",
		PackageManagers: []PackageManager{
			{Type: "npm"},
		},
		Services: []Service{
			{
				Name:         "api",
				StartCommand: "npm start",
				HealthCheck: &ServiceHealth{
					Type: "http",
				},
			},
		},
		SetupSteps: []SetupStep{
			{Name: "migrate", Command: "npm run migrate"},
		},
		HealthChecks: []HealthCheck{
			{URL: "http://localhost:3000"},
		},
	}

	cfg.Defaults()

	if cfg.PackageManagers[0].Path != "." {
		t.Errorf("expected default path '.', got %q", cfg.PackageManagers[0].Path)
	}
	if cfg.Services[0].Type != "process" {
		t.Errorf("expected default type 'process', got %q", cfg.Services[0].Type)
	}
	if cfg.Services[0].HealthCheck.Timeout != "30s" {
		t.Errorf("expected default timeout '30s', got %q", cfg.Services[0].HealthCheck.Timeout)
	}
	if cfg.SetupSteps[0].WorkingDir != "." {
		t.Errorf("expected default working_dir '.', got %q", cfg.SetupSteps[0].WorkingDir)
	}
	if cfg.HealthChecks[0].Interval != "2s" {
		t.Errorf("expected default interval '2s', got %q", cfg.HealthChecks[0].Interval)
	}
}
