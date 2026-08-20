package controllers

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib/ui"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

// Login signs in with profileName, refreshing the SSO session through the
// browser if the cached token has lapsed.
func Login(ctx context.Context, profileName string, setAsDefault bool) error {
	err := services_aws.LoginWithProfile(ctx, profileName, setAsDefault)
	if err == nil {
		return nil
	}
	if !services_aws.SSOSessionExpired(err) {
		return err
	}

	ui.Step("SSO session expired, signing in again")

	ssoRegion, ssoStartURL, err := services_aws.ResolveSSOConfiguration(profileName)
	if err != nil {
		return fmt.Errorf("could not resolve the SSO settings for %s: %w", profileName, err)
	}
	if err := SSOLogin(ctx, ssoRegion, ssoStartURL, false); err != nil {
		return err
	}

	if err := services_aws.LoginWithProfile(ctx, profileName, setAsDefault); err != nil {
		return fmt.Errorf("login still failed after signing in: %w", err)
	}
	return nil
}
