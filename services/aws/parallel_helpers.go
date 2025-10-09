package services_aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
)

// RegionResult representa el resultado de procesar una región específica
type RegionResult struct {
	// Region identifica qué región se procesó
	Region string
	// AccountID identifica a qué cuenta pertenece esta región
	AccountID string
	// Clusters contiene los clusters encontrados en esta región
	Clusters []EKSCluster
	// Error contiene cualquier error que ocurrió durante el procesamiento
	Error error
}

// ProcessRegionsInParallel procesa múltiples regiones en paralelo para una cuenta específica
func ProcessRegionsInParallel(
	ctx context.Context,
	profile, accountID string,
	regions []string,
	config lib.ParallelConfig,
) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	// Creamos contexto con timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	var wg sync.WaitGroup
	// Channel para recibir resultados de cada región
	resultChan := make(chan RegionResult, len(regions))

	workerPool := lib.NewWorkerPool(config.MaxWorkers)

	logger.Infow("Escaneando regiones en paralelo",
		"total_regions", len(regions),
		"account_id", accountID)

	// Lanzamos una goroutine para cada región
	for _, region := range regions {
		wg.Add(1)
		currentRegion := region // Capturar variable para closure

		go func() {
			defer wg.Done()

			logger.Debugw("Buscando clusters en región",
				"region", currentRegion,
				"account_id", accountID)

			err := workerPool.Execute(timeoutCtx, func() error {
				// Obtenemos clusters para esta región específica
				clusters, err := GetClustersForAccountRegion(timeoutCtx, profile, accountID, currentRegion)

				// Enviamos el resultado al channel
				select {
				case resultChan <- RegionResult{
					Region:    currentRegion,
					AccountID: accountID,
					Clusters:  clusters,
					Error:     err,
				}:
					if err != nil {
						logger.Errorw("Error escaneando región",
							"region", currentRegion,
							"account_id", accountID,
							"error", err)
					} else {
						logger.Infow("Región escaneada exitosamente",
							"region", currentRegion,
							"account_id", accountID,
							"clusters_found", len(clusters))
					}
				case <-timeoutCtx.Done():
					return timeoutCtx.Err()
				}
				return nil
			})

			// Manejar errores del worker pool
			if err != nil {
				select {
				case resultChan <- RegionResult{
					Region:    currentRegion,
					AccountID: accountID,
					Clusters:  nil,
					Error:     err,
				}:
				case <-timeoutCtx.Done():
				}
			}
		}()
	}

	// Goroutine para cerrar el channel cuando todas las regiones terminen
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Recolectamos resultados
	var allClusters []EKSCluster
	var hasErrors bool

	for result := range resultChan {
		if result.Error != nil {
			logger.Warnw("Región falló durante escaneo",
				"region", result.Region,
				"account_id", accountID,
				"error", result.Error)
			hasErrors = true
		} else {
			// Agregamos todos los clusters de esta región
			allClusters = append(allClusters, result.Clusters...)
		}
	}

	// Si todas las regiones fallaron, retornamos error
	if hasErrors && len(allClusters) == 0 {
		logger.Errorw("Todas las regiones fallaron",
			"account_id", accountID)
		return nil, fmt.Errorf("todas las regiones fallaron para la cuenta %s", accountID)
	}

	logger.Infow("Escaneo de regiones completado",
		"account_id", accountID,
		"total_clusters", len(allClusters))

	return allClusters, nil
}
