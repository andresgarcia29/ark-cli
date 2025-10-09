package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
)

// GetAllProfiles gets all available account+role combinations
// OPTIMIZED VERSION: Parallelizes role retrieval for multiple accounts
func (s *SSOClient) GetAllProfiles(ctx context.Context, accessToken string) ([]AWSProfile, error) {
	logger := logs.GetLogger()

	// Step 1: Get all accounts (this must be sequential)
	logger.Info("Getting account list")
	accounts, err := s.ListAccounts(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting accounts: %w", err)
	}

	logger.Infow("Accounts found, getting roles in parallel",
		"total_accounts", len(accounts))

	// Configuration for parallel operations
	config := lib.ConservativeConfig()

	// Step 2: Use generic function to process accounts in parallel
	// This function will execute ListAccountRoles for each account simultaneously
	accountRoles, errors := lib.ProcessAccountsInParallel(
		ctx,
		// Convert the account list to a list of IDs
		func() []string {
			var accountIDs []string
			for _, account := range accounts {
				accountIDs = append(accountIDs, account.AccountID)
			}
			return accountIDs
		}(),
		config,
		// This function executes for each account in parallel
		func(ctx context.Context, accountID string) ([]Role, error) {
			logger.Debugf("Getting roles for account: %s", accountID)

			// This is where we make the actual call to the AWS SSO API
			// This function can take several seconds, that's why we parallelize it
			roles, err := s.ListAccountRoles(ctx, accessToken, accountID)
			if err != nil {
				return nil, fmt.Errorf("error getting roles for account %s: %w", accountID, err)
			}

			logger.Infow("Roles obtained for account",
				"account_id", accountID,
				"roles_count", len(roles))
			return roles, nil
		},
	)

	// If there were errors in some accounts, we report them but continue
	if len(errors) > 0 {
		logger.Warnw("Some accounts had errors",
			"error_count", len(errors))
		for _, err := range errors {
			logger.Warnf("  - %v", err)
		}
	}

	// Step 3: Convert results to profiles
	// We need to combine account information with obtained roles
	var profiles []AWSProfile

	// Create a map for fast account information lookup
	accountMap := make(map[string]Account)
	for _, account := range accounts {
		accountMap[account.AccountID] = account
	}

	// For each account that was processed successfully
	for accountID, roles := range accountRoles {
		// Search for complete account information
		account, found := accountMap[accountID]
		if !found {
			// This shouldn't happen, but we handle it for safety
			logger.Warnw("Complete information not found for account",
				"account_id", accountID)
			continue
		}

		// Create a profile for each account+role combination
		for _, role := range roles {
			profiles = append(profiles, AWSProfile{
				AccountID:    account.AccountID,
				AccountName:  account.AccountName,
				RoleName:     role.RoleName,
				EmailAddress: account.EmailAddress,
			})
		}
	}

	logger.Infow("Profiles created successfully",
		"total_profiles", len(profiles))
	return profiles, nil
}

// LoginWithProfile performs complete login with a specific profile
func LoginWithProfile(ctx context.Context, profileName string, setAsDefault bool) error {
	// Step 1: Read profile configuration
	profileConfig, err := ReadProfileFromConfig(profileName)
	if err != nil {
		return fmt.Errorf("failed to read profile config: %w", err)
	}

	// Step 2: Read token from cache
	cachedToken, err := ReadTokenFromCache(profileConfig.StartURL)
	if err != nil {
		return fmt.Errorf("failed to read token from cache (you may need to run login first): %w", err)
	}

	// Step 3: Create SSO client
	client, err := NewSSOClient(ctx, profileConfig.SSORegion, profileConfig.StartURL)
	if err != nil {
		return fmt.Errorf("failed to create SSO client: %w", err)
	}

	// Step 4: Get temporary credentials
	creds, err := client.GetRoleCredentials(ctx, cachedToken.AccessToken, profileConfig.AccountID, profileConfig.RoleName)
	if err != nil {
		return fmt.Errorf("failed to get role credentials: %w", err)
	}

	// Step 5: Write credentials to file
	if err := WriteCredentialsFile(profileName, creds, setAsDefault); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}
