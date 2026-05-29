package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oneconfig/oneconfig/internal/config"
	"gopkg.in/yaml.v3"
)

// EmitYAML generates a well-formatted oneconfig.yml from a Config struct.
// It uses yaml.Node to attach inline TODO comments on fields that were
// inferred heuristically and may need user review.
func EmitYAML(cfg *config.Config) ([]byte, error) {
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	root := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	root.HeadComment = "OneConfig — Auto-generated configuration\nReview this file and look for # TODO markers"

	// project_name
	addScalar(root, "project_name", cfg.ProjectName, "")

	// runtimes
	if len(cfg.Runtimes) > 0 {
		runtimesKey := scalarNode("runtimes", "")
		runtimesVal := &yaml.Node{Kind: yaml.SequenceNode}

		for _, rt := range cfg.Runtimes {
			entry := &yaml.Node{Kind: yaml.MappingNode}
			addScalar(entry, "name", rt.Name, "")
			addScalar(entry, "version", rt.Version, "TODO: verify this version")
			runtimesVal.Content = append(runtimesVal.Content, entry)
		}

		root.Content = append(root.Content, runtimesKey, runtimesVal)
	}

	// package_managers
	if len(cfg.PackageManagers) > 0 {
		pmKey := scalarNode("package_managers", "")
		pmVal := &yaml.Node{Kind: yaml.SequenceNode}

		for _, pm := range cfg.PackageManagers {
			entry := &yaml.Node{Kind: yaml.MappingNode}
			addScalar(entry, "type", pm.Type, "")
			addScalar(entry, "path", pm.Path, "")
			if pm.InstallCommand != "" {
				addScalar(entry, "install_command", pm.InstallCommand, "")
			}
			pmVal.Content = append(pmVal.Content, entry)
		}

		root.Content = append(root.Content, pmKey, pmVal)
	}

	// env_vars
	if len(cfg.EnvVars) > 0 {
		envKey := scalarNode("env_vars", "")
		envVal := &yaml.Node{Kind: yaml.MappingNode}

		// Sort keys for stable output
		keys := make([]string, 0, len(cfg.EnvVars))
		for k := range cfg.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := cfg.EnvVars[k]
			comment := ""
			if looksLikeSecret(k) {
				comment = "TODO: set real value (secret)"
			}
			addScalar(envVal, k, v, comment)
		}

		root.Content = append(root.Content, envKey, envVal)
	}

	// services
	if len(cfg.Services) > 0 {
		svcKey := scalarNode("services", "")
		svcVal := &yaml.Node{Kind: yaml.SequenceNode}

		for _, svc := range cfg.Services {
			entry := emitService(svc)
			svcVal.Content = append(svcVal.Content, entry)
		}

		root.Content = append(root.Content, svcKey, svcVal)
	}

	// setup_steps
	if len(cfg.SetupSteps) > 0 {
		stepsKey := scalarNode("setup_steps", "")
		stepsVal := &yaml.Node{Kind: yaml.SequenceNode}

		for _, step := range cfg.SetupSteps {
			entry := &yaml.Node{Kind: yaml.MappingNode}
			addScalar(entry, "name", step.Name, "")
			addScalar(entry, "command", step.Command, "")
			if len(step.DependsOn) > 0 {
				addStringList(entry, "depends_on", step.DependsOn)
			}
			if step.WorkingDir != "" && step.WorkingDir != "." {
				addScalar(entry, "working_dir", step.WorkingDir, "")
			}
			stepsVal.Content = append(stepsVal.Content, entry)
		}

		root.Content = append(root.Content, stepsKey, stepsVal)
	}

	// health_checks
	if len(cfg.HealthChecks) > 0 {
		hcKey := scalarNode("health_checks", "")
		hcVal := &yaml.Node{Kind: yaml.SequenceNode}

		for _, hc := range cfg.HealthChecks {
			entry := &yaml.Node{Kind: yaml.MappingNode}
			if hc.Name != "" {
				addScalar(entry, "name", hc.Name, "")
			}
			if hc.URL != "" {
				addScalar(entry, "url", hc.URL, "")
			}
			if hc.Port != 0 {
				addIntScalar(entry, "port", hc.Port, "")
			}
			if hc.Command != "" {
				addScalar(entry, "command", hc.Command, "")
			}
			hcVal.Content = append(hcVal.Content, entry)
		}

		root.Content = append(root.Content, hcKey, hcVal)
	}

	// post_start_commands
	if len(cfg.PostStartCommands) > 0 {
		pscKey := scalarNode("post_start_commands", "")
		pscVal := &yaml.Node{Kind: yaml.SequenceNode}

		for _, cmd := range cfg.PostStartCommands {
			pscVal.Content = append(pscVal.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: cmd,
				Tag:   "!!str",
			})
		}

		root.Content = append(root.Content, pscKey, pscVal)
	} else {
		// Always add a post_start section with a welcome message
		pscKey := scalarNode("post_start_commands", "")
		pscVal := &yaml.Node{Kind: yaml.SequenceNode}
		pscVal.Content = append(pscVal.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("echo \"🚀 %s is ready!\"", cfg.ProjectName),
			Tag:   "!!str",
		})
		root.Content = append(root.Content, pscKey, pscVal)
	}

	doc.Content = append(doc.Content, root)

	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encoding YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}

	return []byte(b.String()), nil
}

// emitService creates a yaml.Node tree for a service entry.
func emitService(svc config.Service) *yaml.Node {
	entry := &yaml.Node{Kind: yaml.MappingNode}

	addScalar(entry, "name", svc.Name, "")

	if svc.Type != "" && svc.Type != "process" {
		addScalar(entry, "type", svc.Type, "")
	}

	if svc.WorkingDir != "" && svc.WorkingDir != "." {
		addScalar(entry, "working_dir", svc.WorkingDir, "")
	}

	addScalar(entry, "start_command", svc.StartCommand, "")

	if svc.Port > 0 {
		addIntScalar(entry, "port", svc.Port, "")
	}

	if len(svc.DependsOn) > 0 {
		addStringList(entry, "depends_on", svc.DependsOn)
	}

	if svc.HealthCheck != nil {
		hcKey := scalarNode("health_check", "")
		hcVal := &yaml.Node{Kind: yaml.MappingNode}

		addScalar(hcVal, "type", svc.HealthCheck.Type, "")
		if svc.HealthCheck.Target != "" {
			addScalar(hcVal, "target", svc.HealthCheck.Target, "")
		}

		entry.Content = append(entry.Content, hcKey, hcVal)
	}

	if len(svc.Env) > 0 {
		envKey := scalarNode("env", "")
		envVal := &yaml.Node{Kind: yaml.MappingNode}

		keys := make([]string, 0, len(svc.Env))
		for k := range svc.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			addScalar(envVal, k, svc.Env[k], "")
		}
		entry.Content = append(entry.Content, envKey, envVal)
	}

	return entry
}

// --- yaml.Node helpers ---

func scalarNode(value, lineComment string) *yaml.Node {
	n := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
		Tag:   "!!str",
	}
	if lineComment != "" {
		n.LineComment = lineComment
	}
	return n
}

func addScalar(parent *yaml.Node, key, value, lineComment string) {
	keyNode := scalarNode(key, "")
	valNode := scalarNode(value, lineComment)
	parent.Content = append(parent.Content, keyNode, valNode)
}

func addIntScalar(parent *yaml.Node, key string, value int, lineComment string) {
	keyNode := scalarNode(key, "")
	valNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: fmt.Sprintf("%d", value),
		Tag:   "!!int",
	}
	if lineComment != "" {
		valNode.LineComment = lineComment
	}
	parent.Content = append(parent.Content, keyNode, valNode)
}

func addStringList(parent *yaml.Node, key string, values []string) {
	keyNode := scalarNode(key, "")
	listNode := &yaml.Node{
		Kind:  yaml.SequenceNode,
		Style: yaml.FlowStyle,
	}
	for _, v := range values {
		listNode.Content = append(listNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: v,
			Tag:   "!!str",
		})
	}
	parent.Content = append(parent.Content, keyNode, listNode)
}

// looksLikeSecret checks if an env var name suggests a secret value.
func looksLikeSecret(key string) bool {
	lower := strings.ToLower(key)
	secretPatterns := []string{"password", "secret", "token", "key", "api_key", "apikey", "auth"}
	for _, p := range secretPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
