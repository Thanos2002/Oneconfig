// Package orchestrator handles ordered and parallel execution of setup steps,
// including topological sorting of dependencies.
package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/oneconfig/oneconfig/internal/config"
	"github.com/oneconfig/oneconfig/internal/ui"
)

// Engine executes setup steps in dependency order with optional parallelism.
type Engine struct {
	projectDir string
	verbose    bool
}

// NewEngine creates a new orchestration engine.
func NewEngine(projectDir string, verbose bool) *Engine {
	return &Engine{projectDir: projectDir, verbose: verbose}
}

// RunSteps executes setup steps in topological order, parallelizing independent steps.
func (e *Engine) RunSteps(steps []config.SetupStep) error {
	// Build dependency graph
	order, layers, err := topologicalSort(steps)
	if err != nil {
		return err
	}

	if e.verbose {
		ui.Info(fmt.Sprintf("Execution order: %v (%d layers)", order, len(layers)))
	}

	// Execute layer by layer (steps within a layer run in parallel)
	for _, layer := range layers {
		if len(layer) == 1 {
			// Single step — run sequentially
			step := layer[0]
			ui.Step(fmt.Sprintf("Running: %s", step.Name))
			if err := e.RunCommand(step.Command, filepath.Join(e.projectDir, step.WorkingDir)); err != nil {
				return fmt.Errorf("step %q failed: %w", step.Name, err)
			}
			ui.Success(step.Name)
		} else {
			// Multiple independent steps — run in parallel (FR-9)
			ui.Step(fmt.Sprintf("Running %d steps in parallel", len(layer)))
			var wg sync.WaitGroup
			errCh := make(chan error, len(layer))

			for _, step := range layer {
				wg.Add(1)
				go func(s config.SetupStep) {
					defer wg.Done()
					if err := e.RunCommand(s.Command, filepath.Join(e.projectDir, s.WorkingDir)); err != nil {
						errCh <- fmt.Errorf("step %q failed: %w", s.Name, err)
					} else {
						ui.Success(s.Name)
					}
				}(step)
			}

			wg.Wait()
			close(errCh)

			// Collect errors
			for err := range errCh {
				return err // fail fast on first error
			}
		}
	}

	return nil
}

// RunCommand executes a single shell command in the given directory.
func (e *Engine) RunCommand(command, dir string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	cmd.Env = os.Environ()

	if e.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Capture output for error reporting
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w\n\nOutput:\n%s", err, string(output))
		}
		return nil
	}

	return cmd.Run()
}

// SortServices sorts services in dependency order for startup.
func SortServices(services []config.Service) ([]config.Service, error) {
	// Build adjacency map
	nameMap := make(map[string]config.Service)
	for _, svc := range services {
		nameMap[svc.Name] = svc
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	for _, svc := range services {
		if _, exists := inDegree[svc.Name]; !exists {
			inDegree[svc.Name] = 0
		}
		for _, dep := range svc.DependsOn {
			graph[dep] = append(graph[dep], svc.Name)
			inDegree[svc.Name]++
		}
	}

	// Start with nodes that have no dependencies
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []config.Service
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, nameMap[name])

		for _, dependent := range graph[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(services) {
		return nil, fmt.Errorf("circular dependency detected among services")
	}

	return sorted, nil
}

// topologicalSort orders setup steps by their dependencies and groups independent steps into layers.
func topologicalSort(steps []config.SetupStep) ([]string, [][]config.SetupStep, error) {
	nameMap := make(map[string]config.SetupStep)
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	for _, step := range steps {
		nameMap[step.Name] = step
		if _, exists := inDegree[step.Name]; !exists {
			inDegree[step.Name] = 0
		}
		for _, dep := range step.DependsOn {
			graph[dep] = append(graph[dep], step.Name)
			inDegree[step.Name]++
		}
	}

	// BFS layer by layer
	var layers [][]config.SetupStep
	var order []string

	// Initial queue: all steps with no dependencies
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	for len(queue) > 0 {
		var layer []config.SetupStep
		var nextQueue []string

		for _, name := range queue {
			layer = append(layer, nameMap[name])
			order = append(order, name)

			for _, dependent := range graph[name] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					nextQueue = append(nextQueue, dependent)
				}
			}
		}

		layers = append(layers, layer)
		queue = nextQueue
	}

	if len(order) != len(steps) {
		return nil, nil, fmt.Errorf("circular dependency detected among setup steps")
	}

	return order, layers, nil
}
