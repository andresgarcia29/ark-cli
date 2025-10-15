package services_kubernetes

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/andresgarcia29/ark-cli/logs"
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
	logger := logs.GetLogger()
	logger.Debug("Starting to retrieve cluster contexts from kubectl")

	// Get all context names
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.Debug("Executing kubectl config get-contexts command")
	if err := cmd.Run(); err != nil {
		logger.Errorw("Failed to get cluster contexts", "error", err, "stderr", stderr.String())
		return nil, fmt.Errorf("failed to get cluster contexts: %w\nStderr: %s", err, stderr.String())
	}

	contextNames := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	logger.Debugw("Retrieved context names", "count", len(contextNames), "contexts", contextNames)

	if len(contextNames) == 1 && contextNames[0] == "" {
		logger.Info("No cluster contexts found")
		return []ClusterContext{}, nil
	}

	// Get current context
	logger.Debug("Getting current context")
	currentContext, err := getCurrentContext()
	if err != nil {
		logger.Warnw("Failed to get current context, continuing without marking any as current", "error", err)
		// If we can't get current context, continue without marking any as current
		currentContext = ""
	} else {
		logger.Debugw("Current context retrieved", "context", currentContext)
	}

	contexts := make([]ClusterContext, 0, len(contextNames))
	for _, name := range contextNames {
		if name != "" {
			logger.Debugw("Processing context", "name", name)
			// Get detailed context information including profile
			// profile, region, clusterName, err := getContextDetails(name)
			// if err != nil {
			// 	logger.Warnw("Failed to get context details, using empty values", "context", name, "error", err)
			// 	// If we can't get context details, continue with empty values
			// 	profile = ""
			// 	region = ""
			// 	clusterName = ""
			// } else {
			// 	logger.Debugw("Context details retrieved", "context", name, "profile", profile, "region", region, "cluster", clusterName)
			// }

			context := ClusterContext{
				Name:    name,
				Current: name == currentContext,
				// Profile:     profile,
				// Region:      region,
				// ClusterName: clusterName,
			}
			contexts = append(contexts, context)
			logger.Debugw("Context added to results", "context", context)
		}
	}

	logger.Infow("Successfully retrieved cluster contexts", "count", len(contexts))
	return contexts, nil
}

// getCurrentContext gets the currently active context
func getCurrentContext() (string, error) {
	logger := logs.GetLogger()
	logger.Debug("Executing kubectl config current-context command")

	cmd := exec.Command("kubectl", "config", "current-context")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Errorw("Failed to get current context", "error", err, "stderr", stderr.String())
		return "", fmt.Errorf("failed to get current context: %w\nStderr: %s", err, stderr.String())
	}

	currentContext := strings.TrimSpace(stdout.String())
	logger.Debugw("Current context retrieved", "context", currentContext)
	return currentContext, nil
}

// getContextDetails extracts profile, region, and cluster name from a specific context
func getContextDetails(contextName string) (profile, region, clusterName string, err error) {
	logger := logs.GetLogger()
	logger.Debugw("Getting context details", "context", contextName)

	// Get the full context configuration
	cmd := exec.Command("kubectl", "config", "view", "--context", contextName, "--minify", "--flatten")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.Debugw("Executing kubectl config view command", "context", contextName)
	if err := cmd.Run(); err != nil {
		logger.Errorw("Failed to get context details", "context", contextName, "error", err, "stderr", stderr.String())
		return "", "", "", fmt.Errorf("failed to get context details: %w\nStderr: %s", err, stderr.String())
	}

	config := stdout.String()
	lines := strings.Split(config, "\n")
	logger.Debugw("Parsing context configuration", "context", contextName, "lines", len(lines))

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
			logger.Debugw("Found AWS_PROFILE in env section", "context", contextName, "line", line)
			// Look for the value in the next line
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.Contains(nextLine, "value:") {
					parts := strings.Split(nextLine, "value:")
					if len(parts) == 2 {
						profile = strings.TrimSpace(parts[1])
						logger.Debugw("Extracted AWS profile", "context", contextName, "profile", profile)
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
					logger.Debugw("Extracted region", "context", contextName, "region", region)
				}
			}

			// Look for --cluster-name followed by the cluster name
			if strings.Contains(line, "--cluster-name") && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if !strings.HasPrefix(nextLine, "-") {
					clusterName = nextLine
					logger.Debugw("Extracted cluster name", "context", contextName, "cluster", clusterName)
				}
			}
		}
	}

	logger.Debugw("Context details parsing completed", "context", contextName, "profile", profile, "region", region, "cluster", clusterName)
	return profile, region, clusterName, nil
}

// SwitchToContext switches to the specified cluster context
func SwitchToContext(contextName string) error {
	logger := logs.GetLogger()
	logger.Infow("Switching to cluster context", "context", contextName)

	cmd := exec.Command("kubectl", "config", "use-context", contextName)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.Debugw("Executing kubectl config use-context command", "context", contextName)
	if err := cmd.Run(); err != nil {
		logger.Errorw("Failed to switch to context", "context", contextName, "error", err, "stderr", stderr.String())
		return fmt.Errorf("failed to switch to context %s: %w\nStderr: %s", contextName, err, stderr.String())
	}

	logger.Infow("Successfully switched to context", "context", contextName)
	return nil
}
