package controllers

import (
	"context"
	"fmt"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

// AttemptLoginWithRetry handles login with automatic retry
func AttemptLoginWithRetry(ctx context.Context, profileName string, setAsDefault bool, ssoRegion string, ssoStartURL string) error {
	// First login attempt
	if err := services_aws.LoginWithProfile(ctx, profileName, setAsDefault); err != nil {
		fmt.Printf("❌ Login failed: %v\n", err)
		fmt.Println("🔄 Attempting SSO login...")

		// Perform SSO login
		if ssoErr := AWSSSOLogin(ctx, ssoRegion, ssoStartURL, false); ssoErr != nil {
			return fmt.Errorf("SSO login failed: %v", ssoErr)
		}

		fmt.Println("🔄 Retrying login with updated credentials...")

		// Second login attempt after SSO
		if retryErr := services_aws.LoginWithProfile(ctx, profileName, setAsDefault); retryErr != nil {
			return fmt.Errorf("login failed after SSO: %v", retryErr)
		}
	}

	return nil
}
