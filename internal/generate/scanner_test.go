package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

// helper to create a file within a temp directory
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("creating dir for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestScanner_EmptyProject(t *testing.T) {
	dir := t.TempDir()

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Config.ProjectName == "" {
		t.Error("expected non-empty project name")
	}
	if len(result.Findings) == 0 {
		t.Error("expected at least one finding for empty project")
	}
	if result.Findings[0] != "No project signals detected. Creating a minimal config." {
		t.Errorf("unexpected finding: %s", result.Findings[0])
	}
}

func TestNodeDetector_BasicProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "my-app",
		"scripts": { "start": "node server.js" },
		"engines": { "node": "20.x" }
	}`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect Node.js runtime
	if len(result.Config.Runtimes) == 0 {
		t.Fatal("expected at least one runtime")
	}
	rt := result.Config.Runtimes[0]
	if rt.Name != "node" {
		t.Errorf("expected runtime 'node', got %q", rt.Name)
	}
	if rt.Version != "20.x" {
		t.Errorf("expected version '20.x', got %q", rt.Version)
	}

	// Should detect npm package manager
	if len(result.Config.PackageManagers) == 0 {
		t.Fatal("expected at least one package manager")
	}
	pm := result.Config.PackageManagers[0]
	if pm.Type != "npm" {
		t.Errorf("expected package manager 'npm', got %q", pm.Type)
	}

	// Should detect a service from the "start" script
	if len(result.Config.Services) == 0 {
		t.Fatal("expected at least one service")
	}
	svc := result.Config.Services[0]
	if svc.Name != "my-app" {
		t.Errorf("expected service name 'my-app', got %q", svc.Name)
	}
	if svc.Port != 3000 {
		t.Errorf("expected port 3000, got %d", svc.Port)
	}
}

func TestNodeDetector_YarnLockfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "yarn-project"}`)
	writeFile(t, dir, "yarn.lock", "")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.PackageManagers) == 0 {
		t.Fatal("expected package manager")
	}
	if result.Config.PackageManagers[0].Type != "yarn" {
		t.Errorf("expected yarn, got %q", result.Config.PackageManagers[0].Type)
	}
}

func TestNodeDetector_NvmrcVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "nvmrc-project"}`)
	writeFile(t, dir, ".nvmrc", "18.17.0")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Runtimes) == 0 {
		t.Fatal("expected runtime")
	}
	if result.Config.Runtimes[0].Version != "18.17.0" {
		t.Errorf("expected version '18.17.0', got %q", result.Config.Runtimes[0].Version)
	}
}

func TestNodeDetector_VitePort(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "vite-app",
		"scripts": { "dev": "vite" },
		"devDependencies": { "vite": "^5.0.0" }
	}`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected service")
	}
	if result.Config.Services[0].Port != 5173 {
		t.Errorf("expected port 5173 for vite, got %d", result.Config.Services[0].Port)
	}
}

func TestNodeDetector_ViteConfigPort(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "vite-custom",
		"scripts": { "dev": "vite" },
		"devDependencies": { "vite": "^5.0.0" }
	}`)
	writeFile(t, dir, "vite.config.ts", `
export default defineConfig({
  server: {
    port: 3000
  }
})
`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected service")
	}
	if result.Config.Services[0].Port != 3000 {
		t.Errorf("expected port 3000 from vite.config.ts, got %d", result.Config.Services[0].Port)
	}
}

func TestNodeDetector_EnvPort(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "env-app",
		"scripts": { "start": "node server.js" }
	}`)
	writeFile(t, dir, ".env", "PORT=4000\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected service")
	}
	if result.Config.Services[0].Port != 4000 {
		t.Errorf("expected port 4000 from .env, got %d", result.Config.Services[0].Port)
	}
}

func TestNodeDetector_ScriptInlinePort(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "inline-app",
		"scripts": { "dev": "PORT=4000 next dev" },
		"dependencies": { "next": "^14.0.0" }
	}`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected service")
	}
	if result.Config.Services[0].Port != 4000 {
		t.Errorf("expected port 4000 from inline script assignment, got %d", result.Config.Services[0].Port)
	}
}

func TestPythonDetector_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi==0.103.1\nuvicorn==0.23.2\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect Python
	if len(result.Config.Runtimes) == 0 {
		t.Fatal("expected Python runtime")
	}
	if result.Config.Runtimes[0].Name != "python" {
		t.Errorf("expected runtime 'python', got %q", result.Config.Runtimes[0].Name)
	}

	// Should detect pip
	if len(result.Config.PackageManagers) == 0 {
		t.Fatal("expected package manager")
	}
	if result.Config.PackageManagers[0].Type != "pip" {
		t.Errorf("expected 'pip', got %q", result.Config.PackageManagers[0].Type)
	}

	// Should infer FastAPI service
	if len(result.Config.Services) == 0 {
		t.Fatal("expected service from FastAPI detection")
	}
	svc := result.Config.Services[0]
	if svc.Port != 8000 {
		t.Errorf("expected port 8000, got %d", svc.Port)
	}
	if svc.HealthCheck == nil {
		t.Error("expected health check")
	}
}

func TestPythonDetector_PyprojectPoetry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `
[tool.poetry]
name = "my-python-app"

[tool.poetry.dependencies]
python = "^3.11"
`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.PackageManagers) == 0 {
		t.Fatal("expected package manager")
	}
	if result.Config.PackageManagers[0].Type != "poetry" {
		t.Errorf("expected 'poetry', got %q", result.Config.PackageManagers[0].Type)
	}

	if len(result.Config.Runtimes) == 0 {
		t.Fatal("expected runtime")
	}
	if result.Config.Runtimes[0].Version != "3.11" {
		t.Errorf("expected version '3.11', got %q", result.Config.Runtimes[0].Version)
	}
}

func TestPythonDetector_UvLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0\n")
	writeFile(t, dir, "uv.lock", "")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.PackageManagers) == 0 {
		t.Fatal("expected package manager")
	}
	if result.Config.PackageManagers[0].Type != "uv" {
		t.Errorf("expected 'uv', got %q", result.Config.PackageManagers[0].Type)
	}
}

func TestPythonDetector_FlaskService(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected Flask service")
	}
	svc := result.Config.Services[0]
	if svc.Port != 5000 {
		t.Errorf("expected port 5000 for Flask, got %d", svc.Port)
	}
}

func TestPythonDetector_DjangoService(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "django==5.0\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected Django service")
	}
	svc := result.Config.Services[0]
	if svc.Port != 8000 {
		t.Errorf("expected port 8000 for Django, got %d", svc.Port)
	}
}

func TestGoDetector(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module github.com/example/myapp

go 1.23

require (
	github.com/spf13/cobra v1.8.0
)
`)
	writeFile(t, dir, "main.go", `package main

func main() {}
`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have findings about Go
	hasGoFinding := false
	hasMainFinding := false
	for _, f := range result.Findings {
		if f == "Found go.mod → Go project" {
			hasGoFinding = true
		}
		if f == "  Found main.go → executable project" {
			hasMainFinding = true
		}
	}
	if !hasGoFinding {
		t.Error("expected 'Found go.mod' finding")
	}
	if !hasMainFinding {
		t.Error("expected 'Found main.go' finding")
	}
}

func TestGoDetector_GinFramework(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module github.com/example/myapp

go 1.23

require (
	github.com/gin-gonic/gin v1.9.1
)
`)
	writeFile(t, dir, "main.go", `package main
import "github.com/gin-gonic/gin"
func main() {}
`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected Go web service")
	}
	if result.Config.Services[0].Port != 8080 {
		t.Errorf("expected port 8080 for Gin, got %d", result.Config.Services[0].Port)
	}
}

func TestEnvFileDetector(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", `# Database config
DATABASE_URL=postgres://localhost:5432/mydb
API_SECRET="super-secret"
EMPTY_VAR=
# Another comment
PORT=3000
`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := result.Config.EnvVars
	if env == nil {
		t.Fatal("expected env vars to be populated")
	}

	if env["DATABASE_URL"] != "postgres://localhost:5432/mydb" {
		t.Errorf("expected DATABASE_URL, got %q", env["DATABASE_URL"])
	}
	if env["API_SECRET"] != "super-secret" {
		t.Errorf("expected API_SECRET without quotes, got %q", env["API_SECRET"])
	}
	if env["PORT"] != "3000" {
		t.Errorf("expected PORT=3000, got %q", env["PORT"])
	}
}

func TestDockerComposeDetector(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yml", `
services:
  db:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: dev
      POSTGRES_DB: myapp
  redis:
    image: redis:7
    ports:
      - "6379:6379"
`)

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) < 2 {
		t.Fatalf("expected at least 2 services, got %d", len(result.Config.Services))
	}

	// Find db service
	var dbSvc, redisSvc *config.Service
	for i := range result.Config.Services {
		if result.Config.Services[i].Name == "db" {
			dbSvc = &result.Config.Services[i]
		}
		if result.Config.Services[i].Name == "redis" {
			redisSvc = &result.Config.Services[i]
		}
	}

	_ = dbSvc
	_ = redisSvc

	// Verify services exist by name
	names := make(map[string]bool)
	for _, svc := range result.Config.Services {
		names[svc.Name] = true
		if svc.Type != "docker" {
			t.Errorf("expected service type 'docker', got %q for %s", svc.Type, svc.Name)
		}
	}
	if !names["db"] {
		t.Error("expected 'db' service")
	}
	if !names["redis"] {
		t.Error("expected 'redis' service")
	}
}

func TestMonorepoDetector(t *testing.T) {
	dir := t.TempDir()

	// Root has no package.json
	// Subdirectories have project indicators
	writeFile(t, dir, "frontend/package.json", `{
		"name": "frontend",
		"scripts": { "dev": "next dev" },
		"dependencies": { "next": "^14.0.0" }
	}`)
	writeFile(t, dir, "api/requirements.txt", "fastapi==0.103.1\nuvicorn==0.23.2\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect both runtimes
	runtimes := make(map[string]bool)
	for _, rt := range result.Config.Runtimes {
		runtimes[rt.Name] = true
	}
	if !runtimes["node"] {
		t.Error("expected node runtime")
	}
	if !runtimes["python"] {
		t.Error("expected python runtime")
	}

	// Should detect both package managers
	if len(result.Config.PackageManagers) < 2 {
		t.Errorf("expected at least 2 package managers, got %d", len(result.Config.PackageManagers))
	}

	// Should detect services
	if len(result.Config.Services) < 2 {
		t.Errorf("expected at least 2 services, got %d", len(result.Config.Services))
	}
}

func TestJavaDetector_SpringPortOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", `<project><dependencies><dependency><groupId>org.springframework.boot</groupId></dependency></dependencies></project>`)
	writeFile(t, dir, "src/main/resources/application.properties", "server.port=9090\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Services) == 0 {
		t.Fatal("expected Java service")
	}
	if result.Config.Services[0].Port != 9090 {
		t.Errorf("expected port 9090 from application.properties, got %d", result.Config.Services[0].Port)
	}
}

func TestEmitYAML_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"name": "roundtrip-app",
		"scripts": { "start": "node server.js" },
		"engines": { "node": "20.x" }
	}`)
	writeFile(t, dir, ".env.example", "PORT=3000\nNODE_ENV=development\n")

	scanner := NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}

	yamlBytes, err := EmitYAML(&result.Config)
	if err != nil {
		t.Fatalf("emit error: %v", err)
	}

	yamlStr := string(yamlBytes)
	if yamlStr == "" {
		t.Fatal("expected non-empty YAML output")
	}

	// Verify the output contains expected fields
	for _, expected := range []string{"project_name:", "runtimes:", "package_managers:", "env_vars:", "post_start_commands:"} {
		if !contains(yamlStr, expected) {
			t.Errorf("expected YAML to contain %q", expected)
		}
	}
}

func TestExtractPort_Enhanced(t *testing.T) {
	tests := []struct {
		cmd  string
		want int
	}{
		{"node server.js --port 8080", 8080},
		{"uvicorn main:app --port 8000", 8000},
		{"next dev -p 4000", 4000},
		{"npm start", 0},
		{"", 0},
		{"PORT=4000 next dev", 4000},
		{"fastapi run --port=8080", 8080},
		{"rails s -p 3000", 3000},
		{"server --listen 5050", 5050},
	}

	for _, tt := range tests {
		got := extractPortFromScript(tt.cmd)
		if got != tt.want {
			t.Errorf("extractPortFromScript(%q) = %d, want %d", tt.cmd, got, tt.want)
		}
	}
}

func TestCleanPythonVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"^3.11", "3.11"},
		{">=3.10", "3.10"},
		{"~=3.9.1", "3.9"},
		{"3.12", "3.12"},
		{"==3.11.5", "3.11"},
	}

	for _, tt := range tests {
		got := cleanPythonVersion(tt.input)
		if got != tt.want {
			t.Errorf("cleanPythonVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
