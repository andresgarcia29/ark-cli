package services_aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
)

// RegionResult represents the result of processing a specific region
type RegionResult struct {
	// Region identifies which region was processed
	Region string
	// AccountID identifies which account this region belongs to
	AccountID string
	// Clusters contains the clusters found in this region
	Clusters []EKSCluster
	// Error contains any error that occurred during processing
	Error error
}

// ProcessRegionsInParallel processes multiple regions in parallel for a specific account
func ProcessRegionsInParallel(
	ctx context.Context,
	profile, accountID string,
	regions []string,
	config lib.ParallelConfig,
) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	var wg sync.WaitGroup
	// Channel to receive results from each region
	resultChan := make(chan RegionResult, len(regions))

	workerPool := lib.NewWorkerPool(config.MaxWorkers)

	logger.Infow("Scanning regions in parallel",
		"total_regions", len(regions),
		"account_id", accountID)

	// Launch a goroutine for each region
	for _, region := range regions {
		wg.Add(1)
		currentRegion := region // Capture variable for closure

		go func() {
			defer wg.Done()

			logger.Debugw("Searching for clusters in region",
				"region", currentRegion,
				"account_id", accountID)

			err := workerPool.Execute(timeoutCtx, func() error {
				// Get clusters for this specific region
				clusters, err := GetClustersForAccountRegion(timeoutCtx, profile, accountID, currentRegion)

				// Send the result to the channel
				select {
				case resultChan <- RegionResult{
					Region:    currentRegion,
					AccountID: accountID,
					Clusters:  clusters,
					Error:     err,
				}:
					if err != nil {
						logger.Errorw("Error scanning region",
							"region", currentRegion,
							"account_id", accountID,
							"error", err)
					} else {
						logger.Infow("Region scanned successfully",
							"region", currentRegion,
							"account_id", accountID,
							"clusters_found", len(clusters))
					}
				case <-timeoutCtx.Done():
					return timeoutCtx.Err()
				}
				return nil
			})

			// Handle worker pool errors
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

	// Goroutine to close the channel when all regions finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allClusters []EKSCluster
	var hasErrors bool

	for result := range resultChan {
		if result.Error != nil {
			logger.Warnw("Region failed during scan",
				"region", result.Region,
				"account_id", accountID,
				"error", result.Error)
			hasErrors = true
		} else {
			// Add all clusters from this region
			allClusters = append(allClusters, result.Clusters...)
		}
	}

	// If all regions failed, return error
	if hasErrors && len(allClusters) == 0 {
		logger.Errorw("All regions failed",
			"account_id", accountID)
		return nil, fmt.Errorf("all regions failed for account %s", accountID)
	}

	logger.Infow("Region scan completed",
		"account_id", accountID,
		"total_clusters", len(allClusters))

	return allClusters, nil
}
