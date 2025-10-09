package services_kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
)

// CleanKubeconfig limpia el archivo ~/.kube/config
func CleanKubeconfig(kubeconfigPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	// Verificar si el archivo existe
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		// El archivo no existe, no hay nada que limpiar
		fmt.Println("~/.kube/config does not exist, nothing to clean")
		return nil
	}

	// Crear backup del archivo antes de eliminarlo (opcional pero recomendado)
	backupPath := kubeconfigPath + ".backup"
	if err := os.Rename(kubeconfigPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup kubeconfig: %w", err)
	}
	fmt.Printf("Backup created at: %s\n", backupPath)

	// Crear directorio ~/.kube si no existe
	kubeDir := filepath.Join(homeDir, ".kube")
	if err := os.MkdirAll(kubeDir, 0700); err != nil {
		return fmt.Errorf("failed to create .kube directory: %w", err)
	}

	// Crear archivo vacío
	if err := os.WriteFile(kubeconfigPath, []byte(""), 0600); err != nil {
		return fmt.Errorf("failed to create empty kubeconfig: %w", err)
	}

	fmt.Println("✓ Kubeconfig cleaned successfully")
	return nil
}
