package services_kubernetes

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/andresgarcia29/ark-cli/logs"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

// UpdateKubeconfigForCluster ejecuta aws eks update-kubeconfig para un cluster específico
func UpdateKubeconfigForCluster(cluster services_aws.EKSCluster, replaceProfile string) error {
	if replaceProfile != "" {
		cluster.Profile = replaceProfile
	}

	cmd := exec.Command(
		"aws",
		"eks",
		"update-kubeconfig",
		"--name", cluster.Name,
		"--region", cluster.Region,
		"--profile", cluster.Profile,
		"--alias", cluster.Name,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update kubeconfig for cluster %s: %w\nStderr: %s", cluster.Name, err, stderr.String())
	}

	return nil
}

// UpdateKubeconfigForAllClusters actualiza kubeconfig para todos los clusters
func UpdateKubeconfigForAllClusters(clusters []services_aws.EKSCluster, replaceProfile string) error {
	logger := logs.GetLogger()

	if len(clusters) == 0 {
		logger.Info("No hay clusters para configurar")
		return nil
	}

	logger.Infof("Configurando %d cluster(s)", len(clusters))

	var errors []error
	successCount := 0

	for _, cluster := range clusters {
		logger.Infof("Configurando cluster: %s (cuenta: %s, región: %s)",
			cluster.Name, cluster.AccountID, cluster.Region)

		if err := UpdateKubeconfigForCluster(cluster, replaceProfile); err != nil {
			logger.Errorw("Error configurando cluster",
				"cluster", cluster.Name,
				"account", cluster.AccountID,
				"region", cluster.Region,
				"error", err)
			errors = append(errors, fmt.Errorf("cluster %s: %w", cluster.Name, err))
		} else {
			logger.Infow("Cluster configurado exitosamente",
				"cluster", cluster.Name,
				"account", cluster.AccountID,
				"region", cluster.Region)
			successCount++
		}
	}

	// Reportamos estadísticas finales
	logger.Infow("Configuración completada",
		"exitosos", successCount,
		"fallidos", len(errors),
		"total", len(clusters))

	if len(errors) > 0 {
		logger.Warn("Se encontraron errores durante la configuración:")
		for _, err := range errors {
			logger.Warnf("  - %v", err)
		}
	}

	// Solo consideramos la operación como fallida si TODOS los clusters fallaron
	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("la configuración falló para todos los %d clusters", len(errors))
	}

	return nil
}
