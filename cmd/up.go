package cmd

import (
	"fmt"
	"os"

	"github.com/oneconfig/oneconfig/internal/config"
	"github.com/oneconfig/oneconfig/internal/envvar"
	"github.com/oneconfig/oneconfig/internal/health"
	"github.com/oneconfig/oneconfig/internal/orchestrator"
	"github.com/oneconfig/oneconfig/internal/pkgmanager"
	"github.com/oneconfig/oneconfig/internal/runtime"
	"github.com/oneconfig/oneconfig/internal/services"
	"github.com/oneconfig/oneconfig/internal/ui"
	"github.com/spf13/cobra"
)

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

The environment is ready when all health checks pass.`,
	RunE: runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Step 1: Load and validate config (FR-1, FR-2, FR-3)
	ui.Header("Loading configuration")
	cfg, err := config.Load(dir)
	if err != nil {
		ui.Error("Failed to load configuration", "")
		return err
	}
	ui.Success(fmt.Sprintf("Loaded config for %q", cfg.ProjectName))

	// Step 2: Install runtimes (FR-4)
	if len(cfg.Runtimes) > 0 {
		ui.Header("Installing runtimes")
		rm := runtime.NewManager(verbose)
		for _, rt := range cfg.Runtimes {
			spinner, _ := ui.Spinner(fmt.Sprintf("Setting up %s %s", rt.Name, rt.Version))
			if err := rm.Ensure(rt.Name, rt.Version); err != nil {
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
			spinner, _ := ui.Spinner(fmt.Sprintf("Running %s install in %s", pkg.Type, pkg.Path))
			if err := pm.Install(pkg); err != nil {
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

		// Topological sort for dependency ordering
		order, err := orchestrator.SortServices(cfg.Services)
		if err != nil {
			return fmt.Errorf("resolving service dependencies: %w", err)
		}

		for _, svc := range order {
			spinner, _ := ui.Spinner(fmt.Sprintf("Starting %s", svc.Name))
			if err := svcMgr.Start(svc); err != nil {
				if spinner != nil {
					spinner.Fail(fmt.Sprintf("Failed to start %s", svc.Name))
				}
				return fmt.Errorf("starting service %s: %w", svc.Name, err)
			}

			// Wait for service health check (FR-10)
			if svc.HealthCheck != nil {
				if err := health.WaitForService(svc); err != nil {
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
		ui.Header("Running setup steps")
		orch := orchestrator.NewEngine(dir, verbose)
		if err := orch.RunSteps(cfg.SetupSteps); err != nil {
			return fmt.Errorf("setup steps: %w", err)
		}
	}

	// Step 7: Final health checks (FR-10)
	if len(cfg.HealthChecks) > 0 {
		ui.Header("Verifying environment health")
		for _, hc := range cfg.HealthChecks {
			spinner, _ := ui.Spinner(fmt.Sprintf("Checking %s", hc.Name))
			if err := health.Check(hc); err != nil {
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
		for _, cmd := range cfg.PostStartCommands {
			if err := orch.RunCommand(cmd, dir); err != nil {
				ui.Warning(fmt.Sprintf("Post-start command failed: %s", cmd))
			}
		}
	}

	// Done!
	fmt.Println()
	ui.Success("🚀 Environment is ready!")
	fmt.Println()

	return nil
}
