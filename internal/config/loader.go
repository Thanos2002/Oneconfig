package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed oneconfig.schema.json
var schemaBytes []byte

const (
	// DefaultConfigFile is the default config file name.
	DefaultConfigFile = "oneconfig.yml"
)

// Load reads and parses a oneconfig.yml file from the given directory.
// It validates the config against the JSON schema and applies defaults.
func Load(dir string) (*Config, error) {
	path := filepath.Join(dir, DefaultConfigFile)
	return LoadFile(path)
}

// LoadFile reads and parses a oneconfig.yml file from the given path.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigError{
				Type:    ErrNotFound,
				Message: fmt.Sprintf("No %s found in %s", DefaultConfigFile, filepath.Dir(path)),
				Hint:    "Run 'oneconfig init' to create one, or 'oneconfig generate' to auto-detect your project setup.",
			}
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return Parse(data)
}

// Parse parses raw YAML bytes into a validated Config.
func Parse(data []byte) (*Config, error) {
	// Step 1: Parse YAML into raw map for schema validation
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, &ConfigError{
			Type:    ErrInvalidYAML,
			Message: fmt.Sprintf("Invalid YAML syntax: %s", err),
			Hint:    "Check your oneconfig.yml for YAML formatting issues (indentation, colons, quotes).",
		}
	}

	// Step 2: Validate against JSON Schema
	if err := validateSchema(raw); err != nil {
		return nil, err
	}

	// Step 3: Unmarshal into Go struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, &ConfigError{
			Type:    ErrInvalidConfig,
			Message: fmt.Sprintf("Failed to parse config: %s", err),
			Hint:    "Ensure your oneconfig.yml matches the expected structure. See https://oneconfig.dev/docs/reference",
		}
	}

	// Step 4: Apply defaults
	cfg.Defaults()

	// Step 5: Semantic validation (beyond what JSON Schema can express)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validNamePattern restricts service/step names to safe characters,
// preventing path traversal via crafted names used in log file paths.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Validate performs semantic validation on the config that goes beyond schema checks.
func (c *Config) Validate() error {
	if c.ProjectName == "" {
		return &ConfigError{
			Type:    ErrInvalidConfig,
			Message: "project_name is required",
			Hint:    "Add 'project_name: my-app' to your oneconfig.yml.",
		}
	}

	// Validate service names against safe pattern (prevents path traversal)
	for _, svc := range c.Services {
		if !validNamePattern.MatchString(svc.Name) {
			return &ConfigError{
				Type:    ErrInvalidConfig,
				Message: fmt.Sprintf("Invalid service name: %q", svc.Name),
				Hint:    "Service names must start with a letter or digit and contain only letters, digits, dots, hyphens, and underscores.",
			}
		}
	}

	// Validate setup step names against safe pattern
	for _, step := range c.SetupSteps {
		if !validNamePattern.MatchString(step.Name) {
			return &ConfigError{
				Type:    ErrInvalidConfig,
				Message: fmt.Sprintf("Invalid setup step name: %q", step.Name),
				Hint:    "Step names must start with a letter or digit and contain only letters, digits, dots, hyphens, and underscores.",
			}
		}
	}

	// Check for duplicate service names
	serviceNames := make(map[string]bool)
	for _, svc := range c.Services {
		if serviceNames[svc.Name] {
			return &ConfigError{
				Type:    ErrInvalidConfig,
				Message: fmt.Sprintf("Duplicate service name: %q", svc.Name),
				Hint:    "Each service must have a unique name.",
			}
		}
		serviceNames[svc.Name] = true
	}

	// Check for duplicate setup step names
	stepNames := make(map[string]bool)
	for _, step := range c.SetupSteps {
		if stepNames[step.Name] {
			return &ConfigError{
				Type:    ErrInvalidConfig,
				Message: fmt.Sprintf("Duplicate setup step name: %q", step.Name),
				Hint:    "Each setup step must have a unique name.",
			}
		}
		stepNames[step.Name] = true
	}

	// Validate depends_on references exist
	allNames := make(map[string]bool)
	for k := range serviceNames {
		allNames[k] = true
	}
	for k := range stepNames {
		allNames[k] = true
	}

	for _, svc := range c.Services {
		for _, dep := range svc.DependsOn {
			if !allNames[dep] {
				return &ConfigError{
					Type:    ErrInvalidConfig,
					Message: fmt.Sprintf("Service %q depends on %q, which does not exist", svc.Name, dep),
					Hint:    fmt.Sprintf("Check the depends_on list for service %q. Available names: %s", svc.Name, joinNames(allNames)),
				}
			}
		}
	}

	for _, step := range c.SetupSteps {
		for _, dep := range step.DependsOn {
			if !allNames[dep] {
				return &ConfigError{
					Type:    ErrInvalidConfig,
					Message: fmt.Sprintf("Setup step %q depends on %q, which does not exist", step.Name, dep),
					Hint:    fmt.Sprintf("Check the depends_on list for step %q. Available names: %s", step.Name, joinNames(allNames)),
				}
			}
		}
	}

	// Validate health checks have exactly one target
	for i, hc := range c.HealthChecks {
		targets := 0
		if hc.URL != "" {
			targets++
		}
		if hc.Port != 0 {
			targets++
		}
		if hc.Command != "" {
			targets++
		}
		if targets == 0 {
			return &ConfigError{
				Type:    ErrInvalidConfig,
				Message: fmt.Sprintf("Health check #%d must specify one of: url, port, or command", i+1),
				Hint:    "Each health check needs exactly one target. Example: url: http://localhost:3000/health",
			}
		}
		if targets > 1 {
			return &ConfigError{
				Type:    ErrInvalidConfig,
				Message: fmt.Sprintf("Health check #%d specifies multiple targets (url/port/command); pick one", i+1),
				Hint:    "Each health check should have exactly one of: url, port, or command.",
			}
		}
	}

	// Validate durations are parseable
	for _, svc := range c.Services {
		if svc.HealthCheck != nil {
			if _, err := ParseDuration(svc.HealthCheck.Timeout); err != nil {
				return &ConfigError{
					Type:    ErrInvalidConfig,
					Message: fmt.Sprintf("Invalid timeout %q for service %q health check", svc.HealthCheck.Timeout, svc.Name),
					Hint:    "Use Go duration format: '30s', '2m', '1m30s'.",
				}
			}
			if _, err := ParseDuration(svc.HealthCheck.Interval); err != nil {
				return &ConfigError{
					Type:    ErrInvalidConfig,
					Message: fmt.Sprintf("Invalid interval %q for service %q health check", svc.HealthCheck.Interval, svc.Name),
					Hint:    "Use Go duration format: '2s', '5s', '500ms'.",
				}
			}
		}
	}

	return nil
}

// validateSchema validates a raw YAML-parsed value against the embedded JSON Schema.
func validateSchema(raw any) error {
	// Convert YAML-parsed data to JSON-compatible types
	// (yaml.v3 uses map[string]any, but some values may need conversion)
	jsonCompatible := convertToJSONCompatible(raw)

	// Marshal and unmarshal through JSON to normalize types.
	// If the YAML data contains types not representable in JSON (extremely rare),
	// skip schema validation. The subsequent struct unmarshal + semantic validation
	// will still catch real configuration errors.
	jsonData, err := json.Marshal(jsonCompatible)
	if err != nil {
		return fmt.Errorf("internal error: failed to marshal jsonCompatible: %w", err)
	}

	var jsonValue any
	if err := json.Unmarshal(jsonData, &jsonValue); err != nil {
		return fmt.Errorf("internal error: failed to unmarshal jsonData: %w", err)
	}

	// Compile the embedded schema. Failures here indicate a bug in OneConfig
	// (the schema is compiled into the binary), so they surface as hard errors.
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("oneconfig.schema.json", strings.NewReader(string(schemaBytes))); err != nil {
		return fmt.Errorf("internal error: failed to load embedded JSON schema: %w", err)
	}

	schema, err := compiler.Compile("oneconfig.schema.json")
	if err != nil {
		return fmt.Errorf("internal error: failed to compile embedded JSON schema: %w", err)
	}

	if err := schema.Validate(jsonValue); err != nil {
		return &ConfigError{
			Type:    ErrSchemaValidation,
			Message: fmt.Sprintf("Config validation failed:\n  %s", err),
			Hint:    "Check your oneconfig.yml against the expected schema. Run 'oneconfig doctor' for diagnostics.",
		}
	}

	return nil
}

// convertToJSONCompatible recursively converts YAML-parsed types to JSON-compatible types.
func convertToJSONCompatible(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = convertToJSONCompatible(v)
		}
		return result
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = convertToJSONCompatible(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = convertToJSONCompatible(v)
		}
		return result
	default:
		return v
	}
}

func joinNames(names map[string]bool) string {
	parts := make([]string, 0, len(names))
	for k := range names {
		parts = append(parts, k)
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
