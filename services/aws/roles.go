package services_aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
)

// ListAccountRoles lista todos los roles disponibles para una cuenta específica
func (s *SSOClient) ListAccountRoles(ctx context.Context, accessToken, accountID string) ([]Role, error) {
	var roles []Role
	var nextToken *string

	for {
		input := &sso.ListAccountRolesInput{
			AccessToken: aws.String(accessToken),
			AccountId:   aws.String(accountID),
			MaxResults:  aws.Int32(100), // Máximo permitido por página
			NextToken:   nextToken,
		}

		output, err := s.ssoClient.ListAccountRoles(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list account roles for account %s: %w", accountID, err)
		}

		// Agregar roles de esta página
		for _, role := range output.RoleList {
			roles = append(roles, Role{
				RoleName:  aws.ToString(role.RoleName),
				AccountID: accountID,
			})
		}

		// Si no hay más páginas, terminar
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return roles, nil
}

// GetRoleCredentials obtiene credenciales temporales para un rol específico
func (s *SSOClient) GetRoleCredentials(ctx context.Context, accessToken, accountID, roleName string) (*Credentials, error) {
	input := &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(accessToken),
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
	}

	output, err := s.ssoClient.GetRoleCredentials(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get role credentials: %w", err)
	}

	return &Credentials{
		AccessKeyID:     aws.ToString(output.RoleCredentials.AccessKeyId),
		SecretAccessKey: aws.ToString(output.RoleCredentials.SecretAccessKey),
		SessionToken:    aws.ToString(output.RoleCredentials.SessionToken),
		Expiration:      output.RoleCredentials.Expiration,
	}, nil
}
