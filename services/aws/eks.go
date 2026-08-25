package services_aws

import (
	"context"
	"errors"
	"fmt"
	"sort"

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

// describeCluster fetches the endpoint and CA that a kubeconfig entry needs.
func (e *EKSClient) describeCluster(ctx context.Context, name string) (*eks.DescribeClusterOutput, error) {
	return e.client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(name)})
}

// GetClustersForAccountRegion lists every cluster in one account and region,
// including the details needed to write kubeconfig entries. The describe calls
// share one client, so credentials are resolved once rather than per cluster.
func GetClustersForAccountRegion(ctx context.Context, profile, accountID, region string) ([]EKSCluster, error) {
	eksClient, err := NewEKSClient(ctx, region, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS client: %w", err)
	}

	names, err := eksClient.ListClusters(ctx)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}

	described, errs := lib.MapConcurrent(ctx, names, lib.DefaultLimits(),
		func(ctx context.Context, name string) (EKSCluster, error) {
			out, err := eksClient.describeCluster(ctx, name)
			if err != nil {
				return EKSCluster{}, err
			}
			cluster := EKSCluster{
				Name:      name,
				Region:    region,
				AccountID: accountID,
				Profile:   profile,
				ARN:       aws.ToString(out.Cluster.Arn),
				Endpoint:  aws.ToString(out.Cluster.Endpoint),
			}
			if out.Cluster.CertificateAuthority != nil {
				cluster.CertificateAuthority = aws.ToString(out.Cluster.CertificateAuthority.Data)
			}
			return cluster, nil
		})

	logger := logs.GetLogger()
	for _, err := range errs {
		logger.Debugw("describe cluster failed", "region", region, "error", err)
	}

	clusters := make([]EKSCluster, 0, len(described))
	for _, name := range names {
		if c, ok := described[name]; ok {
			clusters = append(clusters, c)
		}
	}
	return clusters, nil
}

// GetClustersForAccountMultiRegion lists clusters for one account across
// regions, scanning the regions concurrently.
func GetClustersForAccountMultiRegion(ctx context.Context, profile, accountID string, regions []string) ([]EKSCluster, error) {
	switch len(regions) {
	case 0:
		return nil, nil
	case 1:
		return GetClustersForAccountRegion(ctx, profile, accountID, regions[0])
	}

	byRegion, errs := lib.MapConcurrent(ctx, regions, lib.DefaultLimits(),
		func(ctx context.Context, region string) ([]EKSCluster, error) {
			return GetClustersForAccountRegion(ctx, profile, accountID, region)
		})

	var clusters []EKSCluster
	for _, region := range regions {
		clusters = append(clusters, byRegion[region]...)
	}

	if len(clusters) == 0 && len(errs) == len(regions) {
		return nil, fmt.Errorf("every region failed for account %s: %w", accountID, errors.Join(errs...))
	}
	for _, err := range errs {
		logs.GetLogger().Debugw("region scan failed", "account_id", accountID, "error", err)
	}
	return clusters, nil
}

// GetClustersFromAllAccounts gets clusters from all accounts in the specified regions
// OPTIMIZED VERSION: Parallelizes the processing of multiple AWS accounts
func GetClustersFromAllAccounts(ctx context.Context, regions []string, rolePrefixs []string, roleARN string) ([]EKSCluster, []error, error) {
	logger := logs.GetLogger()

	// If no regions are specified, use default
	if len(regions) == 0 {
		regions = []string{"us-west-2"}
	}

	// Step 1: Read all profiles
	logger.Info("Reading profiles from ~/.aws/config")
	allProfiles, err := ReadAllProfilesFromConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read profiles: %w", err)
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
		return nil, nil, nil
	}

	accountIDs := make([]string, 0, len(selectedProfiles))
	for accountID := range selectedProfiles {
		accountIDs = append(accountIDs, accountID)
	}
	sort.Strings(accountIDs)

	totalAccounts.Store(int64(len(accountIDs)))
	limits := lib.DefaultLimits()
	limits.OnDone = func() { scannedAccounts.Add(1) }
	byAccount, errs := lib.MapConcurrent(ctx, accountIDs, limits,
		func(ctx context.Context, accountID string) ([]EKSCluster, error) {
			return processAccount(ctx, accountID, selectedProfiles[accountID], regions)
		})

	var allClusters []EKSCluster
	for _, accountID := range accountIDs {
		allClusters = append(allClusters, byAccount[accountID]...)
	}

	for _, err := range errs {
		logger.Debugw("account scan failed", "error", err)
	}
	if len(allClusters) == 0 && len(errs) == len(accountIDs) {
		return nil, errs, fmt.Errorf("every account failed: %w", errors.Join(errs...))
	}

	return allClusters, errs, nil
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
