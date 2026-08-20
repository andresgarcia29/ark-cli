package services_aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/logs"
)

// credentialSkew is how much life credentials must have left to be reused
// without refreshing.
const credentialSkew = 10 * time.Minute

// errSSOSessionExpired is the routine "your browser session lapsed" case, which
// the retry path turns into a fresh SSO login rather than a hard failure.
var errSSOSessionExpired = errors.New("SSO session expired")

// SSOSessionExpired reports whether err means the cached SSO token is gone.
func SSOSessionExpired(err error) bool { return errors.Is(err, errSSOSessionExpired) }

// GetAllProfiles returns every account+role combination the token can reach,
// fetching each account's roles concurrently.
func (s *SSOClient) GetAllProfiles(ctx context.Context, accessToken string) ([]AWSProfile, error) {
	accounts, err := s.ListAccounts(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting accounts: %w", err)
	}

	accountIDs := make([]string, 0, len(accounts))
	byID := make(map[string]Account, len(accounts))
	for _, a := range accounts {
		accountIDs = append(accountIDs, a.AccountID)
		byID[a.AccountID] = a
	}
	sort.Strings(accountIDs)

	roles, errs := lib.MapConcurrent(ctx, accountIDs, lib.DefaultLimits(),
		func(ctx context.Context, accountID string) ([]Role, error) {
			return s.ListAccountRoles(ctx, accessToken, accountID)
		})

	logger := logs.GetLogger()
	for _, err := range errs {
		logger.Debugw("listing roles failed", "error", err)
	}

	var profiles []AWSProfile
	for _, accountID := range accountIDs {
		account := byID[accountID]
		for _, role := range roles[accountID] {
			profiles = append(profiles, AWSProfile{
				AccountID:    account.AccountID,
				AccountName:  account.AccountName,
				RoleName:     role.RoleName,
				EmailAddress: account.EmailAddress,
			})
		}
	}

	if len(profiles) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("could not list roles for any account: %w", errors.Join(errs...))
	}
	return profiles, nil
}

// LoginWithProfile performs complete login with a specific profile
func LoginWithProfile(ctx context.Context, profileName string, setAsDefault bool) error {
	logger := logs.GetLogger()

	// Skip the round trip when the file already holds credentials that will
	// outlive the work about to be done with them.
	if CachedCredentialsValid(profileName, credentialSkew) && !setAsDefault {
		logger.Debugw("reusing cached credentials", "profile", profileName)
		return nil
	}

	profileConfig, err := ReadProfileFromConfig(profileName)
	if err != nil {
		return lib.Permanent(err)
	}

	var creds *Credentials

	// Step 2: Handle different profile types
	switch profileConfig.ProfileType {
	case ProfileTypeSSO:
		cachedToken, err := ReadTokenFromCache(profileConfig.StartURL)
		if err != nil {
			return lib.Permanent(errSSOSessionExpired)
		}

		// Create SSO client
		client, err := NewSSOClient(ctx, profileConfig.SSORegion, profileConfig.StartURL)
		if err != nil {
			return fmt.Errorf("failed to create SSO client: %w", err)
		}

		// Get temporary credentials
		creds, err = client.GetRoleCredentials(ctx, cachedToken.AccessToken, profileConfig.AccountID, profileConfig.RoleName)
		if err != nil {
			return fmt.Errorf("failed to get role credentials: %w", err)
		}

	case ProfileTypeAssumeRole:
		if profileConfig.RoleARN == "" {
			return lib.Permanent(fmt.Errorf("profile %s is missing role_arn", profileName))
		}
		if profileConfig.SourceProfile == "" {
			return lib.Permanent(fmt.Errorf("profile %s is missing source_profile", profileName))
		}

		// Assume the role
		creds, err = AssumeRoleWithProfile(ctx, profileConfig)
		if err != nil {
			return fmt.Errorf("failed to assume role: %w", err)
		}

	default:
		return lib.Permanent(fmt.Errorf("unsupported profile type: %s", profileConfig.ProfileType))
	}

	// Step 3: Write credentials to file
	if err := WriteCredentialsFile(profileName, creds, setAsDefault); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	logger.Debugw("login successful", "profile", profileName, "type", profileConfig.ProfileType)
	return nil
}

// AssumeRoleWithProfile assumes a role using source profile credentials
func AssumeRoleWithProfile(ctx context.Context, profileConfig *ProfileConfig) (*Credentials, error) {
	// Create source profile configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profileConfig.SourceProfile),
		config.WithRegion(profileConfig.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load source profile config: %w", err)
	}

	// Create STS client
	stsClient := sts.NewFromConfig(cfg)

	// Prepare assume role input
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(profileConfig.RoleARN),
		RoleSessionName: aws.String(fmt.Sprintf("ark-cli-%d", time.Now().Unix())),
	}

	// Add ExternalID if present
	if profileConfig.ExternalID != "" {
		input.ExternalId = aws.String(profileConfig.ExternalID)
	}

	// Assume the role
	result, err := stsClient.AssumeRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	// Convert to our credentials format
	creds := &Credentials{
		AccessKeyID:     aws.ToString(result.Credentials.AccessKeyId),
		SecretAccessKey: aws.ToString(result.Credentials.SecretAccessKey),
		SessionToken:    aws.ToString(result.Credentials.SessionToken),
		Expiration:      result.Credentials.Expiration.UnixMilli(),
	}

	return creds, nil
}
