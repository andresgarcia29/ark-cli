package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/andresgarcia29/ark-cli/lib"
	"github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

// SSOLogin runs the device authorization flow. When bootstrap is set it also
// writes every reachable account+role into ~/.aws/config.
func SSOLogin(ctx context.Context, ssoRegion, ssoStartURL string, bootstrap bool) error {
	client, err := services_aws.NewSSOClient(ctx, ssoRegion, ssoStartURL)
	if err != nil {
		return fmt.Errorf("could not reach AWS SSO in %s: %w", ssoRegion, err)
	}

	registration, err := client.RegisterClient(ctx)
	if err != nil {
		return fmt.Errorf("could not register with AWS SSO: %w", err)
	}

	deviceAuth, err := client.StartDeviceAuthorization(ctx, registration.ClientID, registration.ClientSecret)
	if err != nil {
		return fmt.Errorf("could not start the browser sign-in: %w", err)
	}

	ui.Panel("Approve this sign-in",
		ui.Muted.Render("code ")+ui.Strong.Render(deviceAuth.UserCode),
		ui.Muted.Render(deviceAuth.VerificationURI),
	)

	if err := lib.OpenBrowser(deviceAuth.VerificationURIComplete); err != nil {
		ui.Warn("Could not open your browser automatically")
		ui.Detail("Open %s", deviceAuth.VerificationURIComplete)
	}

	// The device code has its own lifetime; stop polling when it lapses.
	authCtx, cancel := context.WithTimeout(ctx, time.Duration(deviceAuth.ExpiresIn)*time.Second)
	defer cancel()

	var token *services_aws.TokenResponse
	err = animation.Spin(authCtx, "Waiting for approval in your browser",
		func(ctx context.Context, _ animation.Progress) error {
			var err error
			token, err = client.CreateToken(ctx, registration.ClientID, registration.ClientSecret, deviceAuth.DeviceCode, deviceAuth.Interval)
			return err
		})
	if err != nil {
		return err
	}

	if err := client.SaveTokenToCache(token); err != nil {
		return fmt.Errorf("signed in but could not cache the session: %w", err)
	}

	if !bootstrap {
		return nil
	}

	var profiles []services_aws.AWSProfile
	err = animation.Spin(ctx, "Discovering accounts and roles",
		func(ctx context.Context, _ animation.Progress) error {
			var err error
			profiles, err = client.GetAllProfiles(ctx, token.AccessToken)
			return err
		})
	if err != nil {
		return err
	}

	if err := client.WriteConfigFile(profiles); err != nil {
		return err
	}

	ui.Done("Wrote %d profiles to ~/.aws/config", len(profiles))
	return nil
}
