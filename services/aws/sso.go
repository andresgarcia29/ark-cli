package services_aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/smithy-go"
)

func StartSSOSession(ctx context.Context, region, startURL string) error {
	logger := logs.GetLogger()
	logger.Infow("Starting AWS SSO session", "region", region, "start_url", startURL)
	fmt.Println("Starting AWS SSO session")
	return nil
}

// StartDeviceAuthorization starts the device authorization flow
func (s *SSOClient) StartDeviceAuthorization(ctx context.Context, clientID, clientSecret string) (*DeviceAuthorization, error) {
	logger := logs.GetLogger()
	logger.Debugw("Starting device authorization", "client_id", clientID, "start_url", s.StartURL)

	input := &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(clientID),
		ClientSecret: aws.String(clientSecret),
		StartUrl:     aws.String(s.StartURL),
	}

	output, err := s.oidcClient.StartDeviceAuthorization(ctx, input)
	if err != nil {
		logger.Errorw("Failed to start device authorization", "client_id", clientID, "error", err)
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}

	auth := &DeviceAuthorization{
		DeviceCode:              aws.ToString(output.DeviceCode),
		UserCode:                aws.ToString(output.UserCode),
		VerificationURI:         aws.ToString(output.VerificationUri),
		VerificationURIComplete: aws.ToString(output.VerificationUriComplete),
		ExpiresIn:               output.ExpiresIn,
		Interval:                output.Interval,
	}

	logger.Infow("Device authorization started", "user_code", auth.UserCode, "verification_uri", auth.VerificationURI, "expires_in", auth.ExpiresIn)
	return auth, nil
}

// CreateToken polls until the user authorizes or the time expires
func (s *SSOClient) CreateToken(ctx context.Context, clientID, clientSecret, deviceCode string, interval int32) (*TokenResponse, error) {
	logger := logs.GetLogger()
	logger.Debugw("Starting token creation polling", "client_id", clientID, "interval", interval)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	pollCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Token creation cancelled by context")
			return nil, ctx.Err()
		case <-ticker.C:
			pollCount++
			logger.Debugw("Polling for token", "attempt", pollCount)

			input := &ssooidc.CreateTokenInput{
				ClientId:     aws.String(clientID),
				ClientSecret: aws.String(clientSecret),
				DeviceCode:   aws.String(deviceCode),
				GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
			}

			output, err := s.oidcClient.CreateToken(ctx, input)
			if err != nil {
				// If it is AuthorizationPendingException, continue polling
				if isAuthorizationPending(err) {
					logger.Debugw("Authorization still pending", "attempt", pollCount)
					continue
				}
				// If it is SlowDownException, increase the interval
				if isSlowDown(err) {
					newInterval := interval + 5
					logger.Debugw("Rate limited, increasing interval", "old_interval", interval, "new_interval", newInterval)
					ticker.Reset(time.Duration(newInterval) * time.Second)
					continue
				}
				// Any other error, fail
				logger.Errorw("Failed to create token", "attempt", pollCount, "error", err)
				return nil, fmt.Errorf("failed to create token: %w", err)
			}

			// Token obtained successfully
			token := &TokenResponse{
				AccessToken:  aws.ToString(output.AccessToken),
				ExpiresIn:    output.ExpiresIn,
				TokenType:    aws.ToString(output.TokenType),
				RefreshToken: aws.ToString(output.RefreshToken),
			}

			logger.Infow("Token created successfully", "attempts", pollCount, "expires_in", token.ExpiresIn)
			return token, nil
		}
	}
}

// Helper functions to identify specific errors
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
