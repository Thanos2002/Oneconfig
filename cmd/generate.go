package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oneconfig/oneconfig/internal/config"
	"github.com/oneconfig/oneconfig/internal/generate"
	"github.com/oneconfig/oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

var (
	generateForce  bool
	generateStdout bool
	generateOutput string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Scan an existing project and generate a draft oneconfig.yml",
	Long: `Analyzes the current repository to detect languages, package managers,
environment files, and services, then generates a draft oneconfig.yml.

The generated config includes TODO markers where manual review is needed.

Examples:
  oneconfig generate                 # write to ./oneconfig.yml
  oneconfig generate --stdout        # print to stdout instead of writing a file
  oneconfig generate -o custom.yml   # write to a custom path
  oneconfig generate --force         # overwrite an existing oneconfig.yml`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().BoolVarP(&generateForce, "force", "f", false, "overwrite existing oneconfig.yml")
	generateCmd.Flags().BoolVar(&generateStdout, "stdout", false, "print generated config to stdout instead of writing a file")
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "output file path (default: ./oneconfig.yml)")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Determine output path
	configPath := filepath.Join(dir, config.DefaultConfigFile)
	if generateOutput != "" {
		configPath = generateOutput
		if !filepath.IsAbs(configPath) {
			configPath = filepath.Join(dir, configPath)
		}
	}

	// Check if config already exists (unless --stdout or --force)
	if !generateStdout {
		if _, err := os.Stat(configPath); err == nil && !generateForce {
			ui.Warning(fmt.Sprintf("%s already exists. Use --force to overwrite.", filepath.Base(configPath)))
			return nil
		}
	}

	ui.Header("Scanning project")

	scanner := generate.NewScanner(dir)
	result, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("scanning project: %w", err)
	}

	// Print what was detected
	for _, finding := range result.Findings {
		ui.Step(finding)
	}
	fmt.Println()

	// Generate the YAML
	yamlContent, err := generate.EmitYAML(&result.Config)
	if err != nil {
		return fmt.Errorf("generating config YAML: %w", err)
	}

	if generateStdout {
		fmt.Print(string(yamlContent))
		return nil
	}

	if err := os.WriteFile(configPath, yamlContent, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	ui.Success(fmt.Sprintf("Generated %s", configPath))
	ui.Info("Review the file and look for # TODO markers, then run 'oneconfig up'.")

	return nil
}
