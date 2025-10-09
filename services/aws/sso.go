package services_aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/smithy-go"
)

func StartSSOSession(ctx context.Context, region, startURL string) error {
	fmt.Println("Starting AWS SSO session")
	return nil
}

// StartDeviceAuthorization inicia el flujo de autorización del dispositivo
func (s *SSOClient) StartDeviceAuthorization(ctx context.Context, clientID, clientSecret string) (*DeviceAuthorization, error) {
	input := &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(clientID),
		ClientSecret: aws.String(clientSecret),
		StartUrl:     aws.String(s.StartURL),
	}

	output, err := s.oidcClient.StartDeviceAuthorization(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}

	return &DeviceAuthorization{
		DeviceCode:              aws.ToString(output.DeviceCode),
		UserCode:                aws.ToString(output.UserCode),
		VerificationURI:         aws.ToString(output.VerificationUri),
		VerificationURIComplete: aws.ToString(output.VerificationUriComplete),
		ExpiresIn:               output.ExpiresIn,
		Interval:                output.Interval,
	}, nil
}

// CreateToken hace polling hasta que el usuario autorice o expire el tiempo
func (s *SSOClient) CreateToken(ctx context.Context, clientID, clientSecret, deviceCode string, interval int32) (*TokenResponse, error) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			input := &ssooidc.CreateTokenInput{
				ClientId:     aws.String(clientID),
				ClientSecret: aws.String(clientSecret),
				DeviceCode:   aws.String(deviceCode),
				GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
			}

			output, err := s.oidcClient.CreateToken(ctx, input)
			if err != nil {
				// Si es AuthorizationPendingException, continuar polling
				if isAuthorizationPending(err) {
					continue
				}
				// Si es SlowDownException, aumentar el intervalo
				if isSlowDown(err) {
					ticker.Reset(time.Duration(interval+5) * time.Second)
					continue
				}
				// Cualquier otro error, fallar
				return nil, fmt.Errorf("failed to create token: %w", err)
			}

			// Token obtenido exitosamente
			return &TokenResponse{
				AccessToken:  aws.ToString(output.AccessToken),
				ExpiresIn:    output.ExpiresIn,
				TokenType:    aws.ToString(output.TokenType),
				RefreshToken: aws.ToString(output.RefreshToken),
			}, nil
		}
	}
}

// Helper functions para identificar errores específicos
func isAuthorizationPending(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AuthorizationPendingException"
	}
	return false
}

func isSlowDown(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "SlowDownException"
	}
	return false
}
