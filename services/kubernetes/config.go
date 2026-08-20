package services_kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExpandPath resolves a leading ~ against the user's home directory. Flags
// carry paths as typed, and os.Stat does not expand them.
func ExpandPath(path string) (string, error) {
	if path == "" {
		path = "~/.kube/config"
	}
	if !strings.HasPrefix(path, "~") {
		return filepath.Clean(path), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

// BackupKubeconfig moves the current kubeconfig aside under a timestamped
// name and returns that path. It returns "" when there was nothing to back up.
func BackupKubeconfig(kubeconfigPath string) (string, error) {
	path, err := ExpandPath(kubeconfigPath)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to inspect %s: %w", path, err)
	}

	// Two runs in the same second must not overwrite each other's backup.
	stamp := time.Now().Format("20060102-150405")
	backup := fmt.Sprintf("%s.backup-%s", path, stamp)
	for n := 2; ; n++ {
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			break
		}
		backup = fmt.Sprintf("%s.backup-%s.%d", path, stamp, n)
	}

	if err := os.Rename(path, backup); err != nil {
		return "", fmt.Errorf("failed to back up %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return backup, fmt.Errorf("failed to create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		return backup, fmt.Errorf("failed to create empty kubeconfig: %w", err)
	}
	return backup, nil
}
