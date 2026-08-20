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

// CreateToken polls AWS until the user approves the device authorization.
// The caller's context bounds the wait, so an unapproved login cannot hang.
func (s *SSOClient) CreateToken(ctx context.Context, clientID, clientSecret, deviceCode string, interval int32) (*TokenResponse, error) {
	logger := logs.GetLogger()
	if interval <= 0 {
		interval = 5
	}
	wait := time.Duration(interval) * time.Second

	input := &ssooidc.CreateTokenInput{
		ClientId:     aws.String(clientID),
		ClientSecret: aws.String(clientSecret),
		DeviceCode:   aws.String(deviceCode),
		GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
	}

	for attempt := 1; ; attempt++ {
		output, err := s.oidcClient.CreateToken(ctx, input)
		switch {
		case err == nil:
			logger.Debugw("token created", "attempts", attempt)
			return &TokenResponse{
				AccessToken:  aws.ToString(output.AccessToken),
				ExpiresIn:    output.ExpiresIn,
				TokenType:    aws.ToString(output.TokenType),
				RefreshToken: aws.ToString(output.RefreshToken),
			}, nil
		case isAuthorizationPending(err):
			// Expected until the user finishes approving in the browser.
		case isSlowDown(err):
			wait += 5 * time.Second
		default:
			return nil, fmt.Errorf("failed to create token: %w", err)
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, fmt.Errorf("authorization was not completed in time: %w", ctx.Err())
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
