// Package cmd contains all CLI command definitions for the OneConfig tool.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Thanos2002/Oneconfig/internal/ui"
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

// ConfigFilePath returns the path specified by the --config flag, or empty if not set.
func ConfigFilePath() string {
	return cfgFile
}

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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			ui.DisableColor()
		}
	},
}

// Execute runs the root command.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nReceived interrupt signal, initiating graceful shutdown...")
		cancel()
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ./oneconfig.yml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
}
