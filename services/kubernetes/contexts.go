package services_kubernetes

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ClusterContext represents a Kubernetes cluster context
type ClusterContext struct {
	Name        string
	Current     bool
	Profile     string
	Region      string
	ClusterName string
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
			// Get detailed context information including profile
			profile, region, clusterName, err := getContextDetails(name)
			if err != nil {
				// If we can't get context details, continue with empty values
				profile = ""
				region = ""
				clusterName = ""
			}

			contexts = append(contexts, ClusterContext{
				Name:        name,
				Current:     name == currentContext,
				Profile:     profile,
				Region:      region,
				ClusterName: clusterName,
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

// getContextDetails extracts profile, region, and cluster name from a specific context
func getContextDetails(contextName string) (profile, region, clusterName string, err error) {
	// Get the full context configuration
	cmd := exec.Command("kubectl", "config", "view", "--context", contextName, "--minify", "--flatten")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", "", fmt.Errorf("failed to get context details: %w\nStderr: %s", err, stderr.String())
	}

	config := stdout.String()
	lines := strings.Split(config, "\n")

	// Parse the configuration to extract profile, region, and cluster name
	inArgs := false
	inEnv := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Track if we're in the args or env section
		if strings.Contains(line, "args:") {
			inArgs = true
			inEnv = false
			continue
		}
		if strings.Contains(line, "env:") {
			inArgs = false
			inEnv = true
			continue
		}
		if strings.Contains(line, ":") && !strings.Contains(line, "- ") {
			// We've moved to a different section
			inArgs = false
			inEnv = false
		}

		// Extract AWS_PROFILE from env section
		if inEnv && strings.Contains(line, "AWS_PROFILE") {
			// Look for the value in the next line
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.Contains(nextLine, "value:") {
					parts := strings.Split(nextLine, "value:")
					if len(parts) == 2 {
						profile = strings.TrimSpace(parts[1])
					}
				}
			}
		}

		// Extract region and cluster name from args section
		if inArgs {
			// Look for --region followed by the region value
			if strings.Contains(line, "--region") && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if !strings.HasPrefix(nextLine, "-") {
					region = nextLine
				}
			}

			// Look for --cluster-name followed by the cluster name
			if strings.Contains(line, "--cluster-name") && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if !strings.HasPrefix(nextLine, "-") {
					clusterName = nextLine
				}
			}
		}
	}

	return profile, region, clusterName, nil
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
