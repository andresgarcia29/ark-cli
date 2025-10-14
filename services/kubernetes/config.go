package services_kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andresgarcia29/ark-cli/logs"
)

// CleanKubeconfig limpia el archivo ~/.kube/config
func CleanKubeconfig(kubeconfigPath string) error {
	logger := logs.GetLogger()
	logger.Infow("Starting kubeconfig cleanup", "path", kubeconfigPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorw("Failed to get home directory", "error", err)
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		logger.Debugw("Using default kubeconfig path", "path", kubeconfigPath)
	}

	// Verificar si el archivo existe
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		// El archivo no existe, no hay nada que limpiar
		logger.Infow("Kubeconfig file does not exist, nothing to clean", "path", kubeconfigPath)
		fmt.Println("~/.kube/config does not exist, nothing to clean")
		return nil
	}

	logger.Debugw("Kubeconfig file exists, proceeding with cleanup", "path", kubeconfigPath)

	// Crear backup del archivo antes de eliminarlo (opcional pero recomendado)
	backupPath := kubeconfigPath + ".backup"
	logger.Debugw("Creating backup of kubeconfig", "original", kubeconfigPath, "backup", backupPath)

	if err := os.Rename(kubeconfigPath, backupPath); err != nil {
		logger.Errorw("Failed to backup kubeconfig", "original", kubeconfigPath, "backup", backupPath, "error", err)
		return fmt.Errorf("failed to backup kubeconfig: %w", err)
	}

	logger.Infow("Backup created successfully", "backup", backupPath)
	fmt.Printf("Backup created at: %s\n", backupPath)

	// Crear directorio ~/.kube si no existe
	kubeDir := filepath.Join(homeDir, ".kube")
	logger.Debugw("Ensuring .kube directory exists", "path", kubeDir)

	if err := os.MkdirAll(kubeDir, 0700); err != nil {
		logger.Errorw("Failed to create .kube directory", "path", kubeDir, "error", err)
		return fmt.Errorf("failed to create .kube directory: %w", err)
	}

	logger.Debugw(".kube directory ensured", "path", kubeDir)

	// Crear archivo vacío
	logger.Debugw("Creating empty kubeconfig file", "path", kubeconfigPath)

	if err := os.WriteFile(kubeconfigPath, []byte(""), 0600); err != nil {
		logger.Errorw("Failed to create empty kubeconfig", "path", kubeconfigPath, "error", err)
		return fmt.Errorf("failed to create empty kubeconfig: %w", err)
	}

	logger.Infow("Kubeconfig cleaned successfully", "path", kubeconfigPath)
	fmt.Println("✓ Kubeconfig cleaned successfully")
	return nil
}
