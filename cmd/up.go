package cmd

import (
	"fmt"
	"os"

	"github.com/Thanos2002/Oneconfig/internal/config"
	"github.com/Thanos2002/Oneconfig/internal/envvar"
	"github.com/Thanos2002/Oneconfig/internal/health"
	"github.com/Thanos2002/Oneconfig/internal/orchestrator"
	"github.com/Thanos2002/Oneconfig/internal/pkgmanager"
	"github.com/Thanos2002/Oneconfig/internal/runtime"
	"github.com/Thanos2002/Oneconfig/internal/services"
	"github.com/Thanos2002/Oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

var dryRun bool

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Set up and start the development environment",
	Long: `Reads oneconfig.yml and executes the full setup pipeline:

  1. Validate config
  2. Install required runtimes
  3. Run package managers
  4. Set environment variables
  5. Start services
  6. Run setup steps
  7. Verify health checks
  8. Execute post-start commands

Use --dry-run to preview all commands without executing them.
The environment is ready when all health checks pass.`,
	RunE: runUp,
}

func init() {
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview all commands without executing them")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) (retErr error) {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Step 1: Load and validate config (FR-1, FR-2, FR-3)
	ui.Header("Loading configuration")
	cfg, err := config.LoadFromPath(dir, ConfigFilePath())
	if err != nil {
		ui.Error("Failed to load configuration", "")
		return err
	}
	ui.Success(fmt.Sprintf("Loaded config for %q", cfg.ProjectName))

	// --- Dry-run mode: show what would be executed without making changes ---
	if dryRun {
		return printDryRun(cfg)
	}

	// Step 2: Install runtimes (FR-4)
	if len(cfg.Runtimes) > 0 {
		ui.Header("Installing runtimes")
		rm := runtime.NewManager(verbose)
		for _, rt := range cfg.Runtimes {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err()
			}
			spinner, _ := ui.Spinner(fmt.Sprintf("Setting up %s %s", rt.Name, rt.Version))
			if err := rm.Ensure(cmd.Context(), rt.Name, rt.Version); err != nil {
				if spinner != nil {
					spinner.Fail(fmt.Sprintf("Failed to set up %s %s", rt.Name, rt.Version))
				}
				return fmt.Errorf("runtime %s %s: %w", rt.Name, rt.Version, err)
			}
			if spinner != nil {
				spinner.Success(fmt.Sprintf("%s %s ready", rt.Name, rt.Version))
			}
		}
	}

	// Step 3: Install packages (FR-5)
	if len(cfg.PackageManagers) > 0 {
		ui.Header("Installing dependencies")
		pm := pkgmanager.NewRunner(dir, verbose)
		for _, pkg := range cfg.PackageManagers {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err()
			}
			spinner, _ := ui.Spinner(fmt.Sprintf("Running %s install in %s", pkg.Type, pkg.Path))
			if err := pm.Install(cmd.Context(), pkg); err != nil {
				if spinner != nil {
					spinner.Fail(fmt.Sprintf("%s install failed in %s", pkg.Type, pkg.Path))
				}
				return fmt.Errorf("package manager %s: %w", pkg.Type, err)
			}
			if spinner != nil {
				spinner.Success(fmt.Sprintf("%s dependencies installed in %s", pkg.Type, pkg.Path))
			}
		}
	}

	// Step 4: Set environment variables (FR-6)
	if len(cfg.EnvVars) > 0 {
		if cmd.Context().Err() != nil {
			return cmd.Context().Err()
		}
		ui.Header("Configuring environment variables")
		ew := envvar.NewWriter(dir)
		if err := ew.Write(cfg.EnvVars); err != nil {
			return fmt.Errorf("writing env vars: %w", err)
		}
		ui.Success(fmt.Sprintf("Set %d environment variables", len(cfg.EnvVars)))
	}

	// Step 5: Start services (FR-7) with dependency ordering (FR-8, FR-9)
	var svcMgr *services.Manager
	if len(cfg.Services) > 0 {
		ui.Header("Starting services")
		svcMgr = services.NewManager(dir, verbose)

		// Cleanup: if anything fails after services start, stop them
		defer func() {
			if retErr != nil {
				fmt.Println()
				ui.Warning("Failure detected — stopping services that were started...")
				stopped, _ := svcMgr.StopAll()
				if stopped > 0 {
					ui.Info(fmt.Sprintf("Cleaned up %d service(s)", stopped))
				}
			}
		}()

		// Topological sort for dependency ordering
		order, err := orchestrator.SortServices(cfg.Services)
		if err != nil {
			return fmt.Errorf("resolving service dependencies: %w", err)
		}

		for _, svc := range order {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err()
			}
			spinner, _ := ui.Spinner(fmt.Sprintf("Starting %s", svc.Name))
			if err := svcMgr.Start(svc); err != nil {
				if spinner != nil {
					spinner.Fail(fmt.Sprintf("Failed to start %s", svc.Name))
				}
				return fmt.Errorf("starting service %s: %w", svc.Name, err)
			}

			// Wait for service health check (FR-10)
			if svc.HealthCheck != nil {
				if err := health.WaitForService(cmd.Context(), svc); err != nil {
					if spinner != nil {
						spinner.Fail(fmt.Sprintf("%s is not healthy", svc.Name))
					}
					return fmt.Errorf("health check for %s: %w", svc.Name, err)
				}
			}

			if spinner != nil {
				spinner.Success(fmt.Sprintf("%s is running on port %d", svc.Name, svc.Port))
			}
		}
	}

	// Step 6: Run setup steps (FR-8, FR-9)
	if len(cfg.SetupSteps) > 0 {
		if cmd.Context().Err() != nil {
			return cmd.Context().Err()
		}
		ui.Header("Running setup steps")
		orch := orchestrator.NewEngine(dir, verbose)
		if err := orch.RunSteps(cmd.Context(), cfg.SetupSteps); err != nil {
			return fmt.Errorf("setup steps: %w", err)
		}
	}

	// Step 7: Final health checks (FR-10)
	if len(cfg.HealthChecks) > 0 {
		ui.Header("Verifying environment health")
		for _, hc := range cfg.HealthChecks {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err()
			}
			spinner, _ := ui.Spinner(fmt.Sprintf("Checking %s", hc.Name))
			if err := health.Check(cmd.Context(), hc); err != nil {
				if spinner != nil {
					spinner.Fail(fmt.Sprintf("%s health check failed", hc.Name))
				}
				return fmt.Errorf("health check %s: %w", hc.Name, err)
			}
			if spinner != nil {
				spinner.Success(fmt.Sprintf("%s is healthy", hc.Name))
			}
		}
	}

	// Step 8: Post-start commands
	if len(cfg.PostStartCommands) > 0 {
		ui.Header("Running post-start commands")
		orch := orchestrator.NewEngine(dir, verbose)
		for _, postCmd := range cfg.PostStartCommands {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err()
			}
			if err := orch.RunCommand(cmd.Context(), postCmd, dir); err != nil {
				ui.Warning(fmt.Sprintf("Post-start command failed: %s", postCmd))
			}
		}
	}

	// Display service URLs for easy access
	var webServices []config.Service
	for _, svc := range cfg.Services {
		if svc.Port > 0 {
			webServices = append(webServices, svc)
		}
	}

	if len(webServices) > 0 {
		ui.Header("Services available at:")
		for _, svc := range webServices {
			fmt.Printf("  • %s: http://localhost:%d\n", svc.Name, svc.Port)
		}
	}

	// Done!
	fmt.Println()
	ui.Success("🚀 Environment is ready!")
	fmt.Println()

	return nil
}

// printDryRun displays all actions that would be taken without executing them.
func printDryRun(cfg *config.Config) error {
	ui.Header("Dry run — previewing actions (nothing will be executed)")

	if len(cfg.Runtimes) > 0 {
		ui.Step("Runtimes to install:")
		for _, rt := range cfg.Runtimes {
			fmt.Printf("    • %s %s\n", rt.Name, rt.Version)
		}
		fmt.Println()
	}

	if len(cfg.PackageManagers) > 0 {
		ui.Step("Package managers to run:")
		for _, pm := range cfg.PackageManagers {
			cmdStr := pm.InstallCommand
			if cmdStr == "" {
				cmdStr = fmt.Sprintf("%s install", pm.Type)
			}
			fmt.Printf("    • %s (in %s)\n", cmdStr, pm.Path)
		}
		fmt.Println()
	}

	if len(cfg.EnvVars) > 0 {
		ui.Step(fmt.Sprintf("Environment variables to set: %d", len(cfg.EnvVars)))
		for k := range cfg.EnvVars {
			fmt.Printf("    • %s\n", k)
		}
		fmt.Println()
	}

	if len(cfg.Services) > 0 {
		ui.Step("Services to start:")
		for _, svc := range cfg.Services {
			deps := ""
			if len(svc.DependsOn) > 0 {
				deps = fmt.Sprintf(" (depends on: %s)", fmt.Sprintf("%v", svc.DependsOn))
			}
			fmt.Printf("    • %s: %s%s\n", svc.Name, svc.StartCommand, deps)
		}
		fmt.Println()
	}

	if len(cfg.SetupSteps) > 0 {
		ui.Step("Setup steps to run:")
		for _, step := range cfg.SetupSteps {
			fmt.Printf("    • %s: %s\n", step.Name, step.Command)
		}
		fmt.Println()
	}

	if len(cfg.PostStartCommands) > 0 {
		ui.Step("Post-start commands:")
		for _, c := range cfg.PostStartCommands {
			fmt.Printf("    • %s\n", c)
		}
		fmt.Println()
	}

	ui.Info("No changes were made. Remove --dry-run to execute.")
	return nil
}
