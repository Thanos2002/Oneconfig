package cmd

import (
	"fmt"
	"os"

	"github.com/oneconfig/oneconfig/internal/config"
	"github.com/oneconfig/oneconfig/internal/services"
	"github.com/oneconfig/oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of configured services",
	Long:  `Displays the current state of all services defined in oneconfig.yml, including PID, port, and health status.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return err
	}

	ui.Header(fmt.Sprintf("Status: %s", cfg.ProjectName))

	if len(cfg.Services) == 0 {
		ui.Info("No services configured.")
		return nil
	}

	mgr := services.NewManager(dir, verbose)
	statuses := mgr.Status(cfg.Services)

	headers := []string{"Service", "Port", "PID", "Status"}
	var rows [][]string
	for _, s := range statuses {
		rows = append(rows, []string{
			s.Name,
			fmt.Sprintf("%d", s.Port),
			fmt.Sprintf("%d", s.PID),
			ui.StatusBadge(s.State),
		})
	}

	ui.Table(headers, rows)
	return nil
}
