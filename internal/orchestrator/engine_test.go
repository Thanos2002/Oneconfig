package orchestrator

import (
	"testing"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

func TestSortServices_NoDependencies(t *testing.T) {
	services := []config.Service{
		{Name: "api", StartCommand: "npm start"},
		{Name: "worker", StartCommand: "npm run worker"},
	}

	sorted, err := SortServices(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 2 {
		t.Fatalf("expected 2 services, got %d", len(sorted))
	}
}

func TestSortServices_WithDependencies(t *testing.T) {
	services := []config.Service{
		{Name: "api", StartCommand: "npm start", DependsOn: []string{"db"}},
		{Name: "db", StartCommand: "docker run postgres"},
		{Name: "worker", StartCommand: "npm run worker", DependsOn: []string{"api", "db"}},
	}

	sorted, err := SortServices(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 services, got %d", len(sorted))
	}

	// db should come before api and worker
	indexMap := make(map[string]int)
	for i, s := range sorted {
		indexMap[s.Name] = i
	}

	if indexMap["db"] > indexMap["api"] {
		t.Error("db should come before api")
	}
	if indexMap["db"] > indexMap["worker"] {
		t.Error("db should come before worker")
	}
	if indexMap["api"] > indexMap["worker"] {
		t.Error("api should come before worker")
	}
}

func TestSortServices_CircularDependency(t *testing.T) {
	services := []config.Service{
		{Name: "a", StartCommand: "a", DependsOn: []string{"b"}},
		{Name: "b", StartCommand: "b", DependsOn: []string{"a"}},
	}

	_, err := SortServices(services)
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
}

func TestTopologicalSort_Layers(t *testing.T) {
	steps := []config.SetupStep{
		{Name: "a", Command: "a"},
		{Name: "b", Command: "b"},
		{Name: "c", Command: "c", DependsOn: []string{"a", "b"}},
		{Name: "d", Command: "d", DependsOn: []string{"c"}},
	}

	order, layers, err := topologicalSort(steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 steps in order, got %d", len(order))
	}

	// Layer 0 should have a and b (independent)
	if len(layers) < 2 {
		t.Fatalf("expected at least 2 layers, got %d", len(layers))
	}
	if len(layers[0]) != 2 {
		t.Errorf("expected 2 steps in first layer, got %d", len(layers[0]))
	}
}
