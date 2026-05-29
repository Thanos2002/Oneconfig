package cmd

import (
	"fmt"
	"os"

	"github.com/oneconfig/oneconfig/internal/services"
	"github.com/oneconfig/oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all running services",
	Long:  `Stops all services that were started by 'oneconfig up' and cleans up the local state.`,
	RunE:  runDown,
}

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	ui.Header("Stopping services")

	mgr := services.NewManager(dir, verbose)
	stopped, err := mgr.StopAll()
	if err != nil {
		ui.Warning(fmt.Sprintf("Some services may not have stopped cleanly: %s", err))
	}

	if stopped == 0 {
		ui.Info("No services were running.")
	} else {
		ui.Success(fmt.Sprintf("Stopped %d service(s)", stopped))
	}

	return nil
}
