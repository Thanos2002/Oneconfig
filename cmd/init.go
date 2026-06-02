package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Thanos2002/Oneconfig/internal/config"
	"github.com/Thanos2002/Oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create or validate a oneconfig.yml file",
	Long: `Initialize a new OneConfig project by creating a oneconfig.yml file in the
current directory, or validate an existing one.

If a oneconfig.yml already exists, it will be validated and any issues reported.
If no config file exists, an interactive wizard will help you create one.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	configPath := filepath.Join(dir, config.DefaultConfigFile)
	if ConfigFilePath() != "" {
		if filepath.IsAbs(ConfigFilePath()) {
			configPath = ConfigFilePath()
		} else {
			configPath = filepath.Join(dir, ConfigFilePath())
		}
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists — validate it (FR-2)
		ui.Header("Validating existing config")
		cfg, err := config.LoadFile(configPath)
		if err != nil {
			// FR-3: Show clear error
			ui.Error("Configuration is invalid", "")
			return err
		}
		ui.Success(fmt.Sprintf("Config for %q is valid", cfg.ProjectName))
		printConfigSummary(cfg)
		return nil
	}

	// Config doesn't exist — create one (FR-1)
	ui.Header("Creating new oneconfig.yml")

	template := generateDefaultConfig(dir)

	if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	ui.Success(fmt.Sprintf("Created %s", configPath))
	ui.Info("Edit the file to match your project, then run 'oneconfig up' to get started.")

	return nil
}

// generateDefaultConfig creates a starter oneconfig.yml tailored to the detected project.
func generateDefaultConfig(dir string) string {
	projectName := filepath.Base(dir)

	tmpl := fmt.Sprintf(`# OneConfig — Local Development Environment
# Docs: https://github.com/Thanos2002/Oneconfig#-example-configuration

project_name: %s

# Language runtimes required by this project
runtimes:
  - name: node
    version: "20.x"
  # - name: python
  #   version: "3.11"

# Package managers to run during setup
package_managers:
  - type: npm
    path: .
  # - type: pip
  #   path: ./backend

# Environment variables (merged into .env)
env_vars:
  NODE_ENV: development
  # DATABASE_URL: postgres://localhost:5432/%s

# Local services to start
# services:
#   - name: database
#     type: docker
#     start_command: docker run -p 5432:5432 -e POSTGRES_PASSWORD=dev postgres:15
#     port: 5432
#     health_check:
#       type: tcp
#       target: ":5432"
#       timeout: 30s

# Setup steps to run after services are healthy
# setup_steps:
#   - name: migrate
#     command: npm run db:migrate
#     depends_on: [database]

# Final readiness checks
# health_checks:
#   - name: app
#     url: http://localhost:3000/health
#     timeout: 60s

# Commands to run when everything is ready
post_start_commands:
  - echo "🚀 %s is ready for development!"
`, projectName, projectName, projectName)

	return tmpl
}

// printConfigSummary displays a quick overview of the loaded config.
func printConfigSummary(cfg *config.Config) {
	fmt.Println()
	ui.Info(fmt.Sprintf("Project:          %s", cfg.ProjectName))
	ui.Info(fmt.Sprintf("Runtimes:         %d configured", len(cfg.Runtimes)))
	ui.Info(fmt.Sprintf("Package Managers: %d configured", len(cfg.PackageManagers)))
	ui.Info(fmt.Sprintf("Services:         %d configured", len(cfg.Services)))
	ui.Info(fmt.Sprintf("Setup Steps:      %d configured", len(cfg.SetupSteps)))
	ui.Info(fmt.Sprintf("Health Checks:    %d configured", len(cfg.HealthChecks)))
}
