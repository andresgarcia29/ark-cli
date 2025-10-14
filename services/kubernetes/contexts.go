package services_kubernetes

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ClusterContext represents a Kubernetes cluster context
type ClusterContext struct {
	Name    string
	Current bool
}

// GetClusterContexts retrieves all available cluster contexts from kubectl
func GetClusterContexts() ([]ClusterContext, error) {
	// Get all context names
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get cluster contexts: %w\nStderr: %s", err, stderr.String())
	}

	contextNames := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(contextNames) == 1 && contextNames[0] == "" {
		return []ClusterContext{}, nil
	}

	// Get current context
	currentContext, err := getCurrentContext()
	if err != nil {
		// If we can't get current context, continue without marking any as current
		currentContext = ""
	}

	contexts := make([]ClusterContext, 0, len(contextNames))
	for _, name := range contextNames {
		if name != "" {
			contexts = append(contexts, ClusterContext{
				Name:    name,
				Current: name == currentContext,
			})
		}
	}

	return contexts, nil
}

// getCurrentContext gets the currently active context
func getCurrentContext() (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current context: %w\nStderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// SwitchToContext switches to the specified cluster context
func SwitchToContext(contextName string) error {
	cmd := exec.Command("kubectl", "config", "use-context", contextName)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to context %s: %w\nStderr: %s", contextName, err, stderr.String())
	}

	return nil
}
