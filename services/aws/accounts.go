package services_aws

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

func (s *SSOClient) ListAccounts(ctx context.Context, accessToken string) ([]Account, error) {
	logger := logs.GetLogger()
	logger.Debug("Starting to list AWS accounts")

	var accounts []Account
	var nextToken *string
	pageCount := 0

	for {
		pageCount++
		logger.Debugw("Fetching accounts page", "page", pageCount, "has_next_token", nextToken != nil)

		input := &sso.ListAccountsInput{
			AccessToken: aws.String(accessToken),
			MaxResults:  aws.Int32(100), // Maximum allowed per page
			NextToken:   nextToken,
		}

		output, err := s.ssoClient.ListAccounts(ctx, input)
		if err != nil {
			logger.Errorw("Failed to list accounts", "page", pageCount, "error", err)
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		logger.Debugw("Accounts page retrieved", "page", pageCount, "accounts_in_page", len(output.AccountList))

		// Add accounts from this page
		for _, acc := range output.AccountList {
			account := Account{
				AccountID:    aws.ToString(acc.AccountId),
				AccountName:  aws.ToString(acc.AccountName),
				EmailAddress: aws.ToString(acc.EmailAddress),
			}
			accounts = append(accounts, account)
			logger.Debugw("Account added", "account_id", account.AccountID, "account_name", account.AccountName)
		}

		// If there are no more pages, terminate
		if output.NextToken == nil {
			logger.Debug("No more pages to fetch")
			break
		}
		nextToken = output.NextToken
	}

	logger.Infow("Successfully listed all accounts", "total_accounts", len(accounts), "total_pages", pageCount)
	return accounts, nil
}
