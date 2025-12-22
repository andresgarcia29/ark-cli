package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

// ListClusters lists all EKS clusters in the configured region
func (e *EKSClient) ListClusters(ctx context.Context) ([]string, error) {
	var clusters []string
	var nextToken *string

	for {
		input := &eks.ListClustersInput{
			MaxResults: aws.Int32(100),
			NextToken:  nextToken,
		}

		output, err := e.client.ListClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
		}

		clusters = append(clusters, output.Clusters...)

		// If there are no more pages, finish
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return clusters, nil
}

// GetClustersForAccountRegion gets all clusters for a specific account and region
func GetClustersForAccountRegion(ctx context.Context, profile, accountID, region string) ([]EKSCluster, error) {
	// Create EKS client
	eksClient, err := NewEKSClient(ctx, region, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS client: %w", err)
	}

	// List clusters
	clusterNames, err := eksClient.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	// Create EKSCluster objects
	var clusters []EKSCluster
	for _, name := range clusterNames {
		clusters = append(clusters, EKSCluster{
			Name:      name,
			Region:    region,
			AccountID: accountID,
			Profile:   profile,
		})
	}

	return clusters, nil
}

// GetClustersForAccountMultiRegion gets all clusters for an account in multiple regions
// OPTIMIZED VERSION: Parallelizes the search across multiple regions simultaneously
func GetClustersForAccountMultiRegion(ctx context.Context, profile, accountID string, regions []string) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	// If there are no regions, return empty list
	if len(regions) == 0 {
		return []EKSCluster{}, nil
	}

	// If there's only one region, we don't need parallelization
	if len(regions) == 1 {
		return GetClustersForAccountRegion(ctx, profile, accountID, regions[0])
	}

	logger.Infow("Scanning regions in parallel",
		"total_regions", len(regions),
		"account_id", accountID)

	// Configuration for parallelization
	config := lib.ConservativeConfig()

	// Use our specialized function to process regions in parallel
	// This function automatically handles:
	// - Concurrency control (maximum 10 simultaneous regions)
	// - Timeouts to prevent hangs
	// - Result collection from channels
	// - Partial error handling
	allClusters, err := ProcessRegionsInParallel(ctx, profile, accountID, regions, config)
	if err != nil {
		return nil, fmt.Errorf("error processing regions for account %s: %w", accountID, err)
	}

	logger.Infow("Clusters found in multiple regions",
		"account_id", accountID,
		"total_clusters", len(allClusters),
		"regions_scanned", len(regions))

	return allClusters, nil
}

// GetClustersFromAllAccounts gets clusters from all accounts in the specified regions
// OPTIMIZED VERSION: Parallelizes the processing of multiple AWS accounts
func GetClustersFromAllAccounts(ctx context.Context, regions []string, rolePrefixs []string, roleARN string) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	// If no regions are specified, use default
	if len(regions) == 0 {
		regions = []string{"us-west-2"}
	}

	// Step 1: Read all profiles
	logger.Info("Reading profiles from ~/.aws/config")
	allProfiles, err := ReadAllProfilesFromConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles: %w", err)
	}

	// Step 2: Select profiles based on prefix or specific ARN
	var selectedProfiles map[string]ProfileConfig
	if roleARN != "" {
		logger.Infow("Searching for profile with specific Role ARN", "role_arn", roleARN)
		selectedProfiles = SelectProfileByARN(allProfiles, roleARN)
	} else {
		selectedProfiles = SelectProfilesPerAccount(allProfiles, rolePrefixs)
	}

	logger.Infow("Accounts found to scan",
		"total_accounts", len(selectedProfiles))

	if len(selectedProfiles) == 0 {
		logger.Warn("No accounts found to process")
		return []EKSCluster{}, nil
	}

	// If there's only one account, we don't need parallelization
	if len(selectedProfiles) == 1 {
		for accountID, profile := range selectedProfiles {
			return processAccount(ctx, accountID, profile, regions)
		}
	}

	// Configuration for parallelization
	config := lib.ConservativeConfig()

	// Convert the profile map to a list of account IDs
	var accountIDs []string
	profileMap := make(map[string]ProfileConfig)
	for accountID, profile := range selectedProfiles {
		accountIDs = append(accountIDs, accountID)
		profileMap[accountID] = profile
	}

	logger.Infow("Processing accounts in parallel",
		"total_accounts", len(accountIDs),
		"max_workers", config.MaxWorkers)

	// Step 3: Use parallelization to process all accounts
	// This function will execute login and cluster retrieval for each account simultaneously
	accountResults, errors := lib.ProcessAccountsInParallel(
		ctx,
		accountIDs,
		config,
		// This function executes for each account in parallel
		func(ctx context.Context, accountID string) ([]EKSCluster, error) {
			// Get the profile information for this account
			profile, exists := profileMap[accountID]
			if !exists {
				return nil, fmt.Errorf("profile not found for account %s", accountID)
			}

			// Process this account (login + get clusters)
			return processAccount(ctx, accountID, profile, regions)
		},
	)

	// Report errors but continue with successful results
	if len(errors) > 0 {
		logger.Warnw("Some accounts had errors",
			"error_count", len(errors))
		for _, err := range errors {
			logger.Warnf("  - %v", err)
		}
	}

	// Combine all clusters from all successful accounts
	var allClusters []EKSCluster
	for accountID, clusters := range accountResults {
		allClusters = append(allClusters, clusters...)
		logger.Infow("Account contributed clusters",
			"account_id", accountID,
			"clusters_count", len(clusters))
	}

	logger.Infow("Parallel processing completed",
		"total_clusters", len(allClusters),
		"successful_accounts", len(accountResults),
		"failed_accounts", len(errors))

	return allClusters, nil
}

// processAccount processes a specific account: logs in and gets all clusters
// This function is separated to facilitate parallelization and testing
func processAccount(ctx context.Context, accountID string, profile ProfileConfig, regions []string) ([]EKSCluster, error) {
	logger := logs.GetLogger()

	logger.Infow("Processing account",
		"account_id", accountID,
		"profile", profile.ProfileName,
		"role", profile.RoleName)

	// Step 1: Login with profile (without set-default to avoid conflicts in parallel)
	logger.Debugw("Performing login",
		"profile", profile.ProfileName)
	if err := LoginWithProfile(ctx, profile.ProfileName, false); err != nil {
		return nil, fmt.Errorf("failed to login with profile %s: %w", profile.ProfileName, err)
	}
	logger.Infow("Login successful",
		"profile", profile.ProfileName)

	// Step 2: Get clusters in all specified regions
	// This function is already parallelized to handle multiple regions simultaneously
	logger.Debugw("Scanning regions",
		"regions", regions)
	clusters, err := GetClustersForAccountMultiRegion(ctx, profile.ProfileName, accountID, regions)
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters for account %s: %w", accountID, err)
	}

	if len(clusters) > 0 {
		logger.Infow("Clusters found",
			"account_id", accountID,
			"clusters_count", len(clusters))
	} else {
		logger.Infow("No clusters found",
			"account_id", accountID)
	}

	return clusters, nil
}
