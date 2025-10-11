package controllers

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/andresgarcia29/ark-cli/logs"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

// UpdateKubeconfigForCluster executes aws eks update-kubeconfig for a specific cluster
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

// UpdateKubeconfigForAllClusters updates kubeconfig for all clusters
func UpdateKubeconfigForAllClusters(clusters []services_aws.EKSCluster, replaceProfile string) error {
	logger := logs.GetLogger()

	if len(clusters) == 0 {
		logger.Info("No clusters to configure")
		return nil
	}

	logger.Infof("Configuring %d cluster(s)", len(clusters))

	var errors []error
	successCount := 0

	for _, cluster := range clusters {
		logger.Infof("Configuring cluster: %s (account: %s, region: %s)",
			cluster.Name, cluster.AccountID, cluster.Region)

		if err := UpdateKubeconfigForCluster(cluster, replaceProfile); err != nil {
			logger.Errorw("Error configuring cluster",
				"cluster", cluster.Name,
				"account", cluster.AccountID,
				"region", cluster.Region,
				"error", err)
			errors = append(errors, fmt.Errorf("cluster %s: %w", cluster.Name, err))
		} else {
			logger.Infow("Cluster configured successfully",
				"cluster", cluster.Name,
				"account", cluster.AccountID,
				"region", cluster.Region)
			successCount++
		}
	}

	// Report final statistics
	logger.Infow("Configuration completed",
		"successful", successCount,
		"failed", len(errors),
		"total", len(clusters))

	if len(errors) > 0 {
		logger.Warn("Errors found during configuration:")
		for _, err := range errors {
			logger.Warnf("  - %v", err)
		}
	}

	// We only consider the operation as failed if ALL clusters failed
	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("configuration failed for all %d clusters", len(errors))
	}

	return nil
}

// UpdateKubeconfigWithProgress updates kubeconfig for all clusters with a progress bar
func UpdateKubeconfigWithProgress(clusters []services_aws.EKSCluster, replaceProfile string) error {
	if len(clusters) == 0 {
		fmt.Println("No clusters to configure")
		return nil
	}

	// Variable para almacenar errores
	var finalError error

	// Usar la barra de progreso
	err := animation.ShowProgressBar(len(clusters), func(update func(item string, err error)) error {
		var errors []error

		for _, cluster := range clusters {
			// Configurar el cluster
			clusterName := fmt.Sprintf("%s (%s)", cluster.Name, cluster.Region)
			err := UpdateKubeconfigForCluster(cluster, replaceProfile)

			// Actualizar el progreso
			update(clusterName, err)

			// Guardar error si existe
			if err != nil {
				errors = append(errors, fmt.Errorf("cluster %s: %w", cluster.Name, err))
			}
		}

		// Si hay errores pero no todos fallaron, no retornar error
		// Solo retornar error si TODOS fallaron
		if len(errors) > 0 && len(errors) == len(clusters) {
			finalError = fmt.Errorf("configuration failed for all %d clusters", len(errors))
			return finalError
		}

		if len(errors) > 0 {
			finalError = fmt.Errorf("some clusters failed to configure (%d/%d)", len(errors), len(clusters))
		}

		return nil
	})

	if err != nil {
		return err
	}

	return finalError
}
