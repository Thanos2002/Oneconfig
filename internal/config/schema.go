// Package config defines the Go types for the oneconfig.yml data model
// and provides loading, parsing, and validation functionality.
package config

import "time"

// Config is the top-level structure of a oneconfig.yml file.
type Config struct {
	ProjectName       string            `yaml:"project_name" json:"project_name"`
	Runtimes          []Runtime         `yaml:"runtimes,omitempty" json:"runtimes,omitempty"`
	PackageManagers   []PackageManager  `yaml:"package_managers,omitempty" json:"package_managers,omitempty"`
	EnvVars           map[string]string `yaml:"env_vars,omitempty" json:"env_vars,omitempty"`
	Services          []Service         `yaml:"services,omitempty" json:"services,omitempty"`
	SetupSteps        []SetupStep       `yaml:"setup_steps,omitempty" json:"setup_steps,omitempty"`
	HealthChecks      []HealthCheck     `yaml:"health_checks,omitempty" json:"health_checks,omitempty"`
	PostStartCommands []string          `yaml:"post_start_commands,omitempty" json:"post_start_commands,omitempty"`
}

// Runtime describes a language runtime required by the project.
type Runtime struct {
	Name    string `yaml:"name" json:"name"`       // e.g. "node", "python"
	Version string `yaml:"version" json:"version"` // e.g. "20.x", "3.11"
}

// PackageManager describes a package manager invocation.
type PackageManager struct {
	Type           string `yaml:"type" json:"type"`                                           // e.g. "npm", "yarn", "pip", "poetry"
	Path           string `yaml:"path,omitempty" json:"path,omitempty"`                       // relative path to manifest dir (default ".")
	InstallCommand string `yaml:"install_command,omitempty" json:"install_command,omitempty"` // override default install cmd
}

// Service describes a local service to start as part of the dev environment.
type Service struct {
	Name         string            `yaml:"name" json:"name"`
	Type         string            `yaml:"type,omitempty" json:"type,omitempty"` // "process", "docker", "script" (default "process")
	StartCommand string            `yaml:"start_command" json:"start_command"`
	StopCommand  string            `yaml:"stop_command,omitempty" json:"stop_command,omitempty"`
	Port         int               `yaml:"port,omitempty" json:"port,omitempty"`
	HealthCheck  *ServiceHealth    `yaml:"health_check,omitempty" json:"health_check,omitempty"`
	DependsOn    []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	WorkingDir   string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	Env          map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// ServiceHealth defines a health check attached to a service.
type ServiceHealth struct {
	Type     string `yaml:"type" json:"type"`                             // "http", "tcp", "cmd"
	Target   string `yaml:"target,omitempty" json:"target,omitempty"`     // URL, port string, or command
	Timeout  string `yaml:"timeout,omitempty" json:"timeout,omitempty"`   // e.g. "30s"
	Interval string `yaml:"interval,omitempty" json:"interval,omitempty"` // e.g. "2s"
}

// SetupStep describes an ordered command to run during environment setup.
type SetupStep struct {
	Name       string   `yaml:"name" json:"name"`
	Command    string   `yaml:"command" json:"command"`
	DependsOn  []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	WorkingDir string   `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
}

// HealthCheck describes a final readiness check for the environment.
type HealthCheck struct {
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	URL      string `yaml:"url,omitempty" json:"url,omitempty"`         // HTTP GET, expect 2xx
	Port     int    `yaml:"port,omitempty" json:"port,omitempty"`       // TCP dial
	Command  string `yaml:"command,omitempty" json:"command,omitempty"` // shell cmd, expect exit 0
	Timeout  string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Interval string `yaml:"interval,omitempty" json:"interval,omitempty"`
}

// Defaults applies default values to the config where fields are omitted.
func (c *Config) Defaults() {
	for i := range c.PackageManagers {
		if c.PackageManagers[i].Path == "" {
			c.PackageManagers[i].Path = "."
		}
	}
	for i := range c.Services {
		if c.Services[i].Type == "" {
			c.Services[i].Type = "process"
		}
		if c.Services[i].WorkingDir == "" {
			c.Services[i].WorkingDir = "."
		}
		if c.Services[i].HealthCheck != nil {
			if c.Services[i].HealthCheck.Timeout == "" {
				c.Services[i].HealthCheck.Timeout = "30s"
			}
			if c.Services[i].HealthCheck.Interval == "" {
				c.Services[i].HealthCheck.Interval = "2s"
			}
		}
	}
	for i := range c.SetupSteps {
		if c.SetupSteps[i].WorkingDir == "" {
			c.SetupSteps[i].WorkingDir = "."
		}
	}
	for i := range c.HealthChecks {
		if c.HealthChecks[i].Timeout == "" {
			c.HealthChecks[i].Timeout = "30s"
		}
		if c.HealthChecks[i].Interval == "" {
			c.HealthChecks[i].Interval = "2s"
		}
	}
}

// ParseDuration parses a timeout/interval string like "30s" or "2m" into a time.Duration.
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
