package services_aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

func (s *SSOClient) ListAccounts(ctx context.Context, accessToken string) ([]Account, error) {
	var accounts []Account
	var nextToken *string

	for {
		input := &sso.ListAccountsInput{
			AccessToken: aws.String(accessToken),
			MaxResults:  aws.Int32(100), // Máximo permitido por página
			NextToken:   nextToken,
		}

		output, err := s.ssoClient.ListAccounts(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		// Agregar cuentas de esta página
		for _, acc := range output.AccountList {
			accounts = append(accounts, Account{
				AccountID:    aws.ToString(acc.AccountId),
				AccountName:  aws.ToString(acc.AccountName),
				EmailAddress: aws.ToString(acc.EmailAddress),
			})
		}

		// Si no hay más páginas, terminar
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return accounts, nil
}
