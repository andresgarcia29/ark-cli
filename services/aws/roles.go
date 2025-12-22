package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

// ListAccountRoles lists all available roles for a specific account
func (s *SSOClient) ListAccountRoles(ctx context.Context, accessToken, accountID string) ([]Role, error) {
	logger := logs.GetLogger()
	logger.Debugw("Starting to list account roles", "account_id", accountID)

	var roles []Role
	var nextToken *string
	pageCount := 0

	for {
		pageCount++
		logger.Debugw("Fetching roles page", "account_id", accountID, "page", pageCount, "has_next_token", nextToken != nil)

		input := &sso.ListAccountRolesInput{
			AccessToken: aws.String(accessToken),
			AccountId:   aws.String(accountID),
			MaxResults:  aws.Int32(100), // Maximum allowed per page
			NextToken:   nextToken,
		}

		output, err := s.ssoClient.ListAccountRoles(ctx, input)
		if err != nil {
			logger.Errorw("Failed to list account roles", "account_id", accountID, "page", pageCount, "error", err)
			return nil, fmt.Errorf("failed to list account roles for account %s: %w", accountID, err)
		}

		logger.Debugw("Roles page retrieved", "account_id", accountID, "page", pageCount, "roles_in_page", len(output.RoleList))

		// Add roles from this page
		for _, role := range output.RoleList {
			roleObj := Role{
				RoleName:  aws.ToString(role.RoleName),
				AccountID: accountID,
			}
			roles = append(roles, roleObj)
			logger.Debugw("Role added", "account_id", accountID, "role_name", roleObj.RoleName)
		}

		// If there are no more pages, terminate
		if output.NextToken == nil {
			logger.Debugw("No more pages to fetch", "account_id", accountID)
			break
		}
		nextToken = output.NextToken
	}

	logger.Infow("Successfully listed all account roles", "account_id", accountID, "total_roles", len(roles), "total_pages", pageCount)
	return roles, nil
}

// GetRoleCredentials obtains temporary credentials for a specific role
func (s *SSOClient) GetRoleCredentials(ctx context.Context, accessToken, accountID, roleName string) (*Credentials, error) {
	logger := logs.GetLogger()
	logger.Debugw("Getting role credentials", "account_id", accountID, "role_name", roleName)

	input := &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(accessToken),
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
	}

	output, err := s.ssoClient.GetRoleCredentials(ctx, input)
	if err != nil {
		logger.Errorw("Failed to get role credentials", "account_id", accountID, "role_name", roleName, "error", err)
		return nil, fmt.Errorf("failed to get role credentials: %w", err)
	}

	credentials := &Credentials{
		AccessKeyID:     aws.ToString(output.RoleCredentials.AccessKeyId),
		SecretAccessKey: aws.ToString(output.RoleCredentials.SecretAccessKey),
		SessionToken:    aws.ToString(output.RoleCredentials.SessionToken),
		Expiration:      output.RoleCredentials.Expiration,
	}

	logger.Debugw("Role credentials obtained successfully", "account_id", accountID, "role_name", roleName, "expiration", credentials.Expiration)
	return credentials, nil
}
