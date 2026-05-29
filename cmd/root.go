// Package cmd contains all CLI command definitions for the OneConfig tool.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via ldflags.
	Version = "dev"

	// Global flags
	cfgFile string
	verbose bool
	noColor bool
)

// rootCmd is the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "oneconfig",
	Short: "Set up any dev environment with one command",
	Long: `OneConfig reads a single oneconfig.yml file at the root of your repository
and provisions a fully working local development environment.

No more reading READMEs, installing mismatched tool versions, or starting
services manually. Just clone and run:

  oneconfig up`,
	Version: Version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ./oneconfig.yml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
}
