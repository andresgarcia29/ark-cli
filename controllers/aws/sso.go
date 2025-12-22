package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/andresgarcia29/ark-cli/lib"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

func AWSSSOLogin(ctx context.Context, SSORegion string, SSOStartURL string, boostraping bool) error {
	// Step 1: Create SSO client
	client, err := services_aws.NewSSOClient(ctx, SSORegion, SSOStartURL)
	if err != nil {
		fmt.Println("Error creating SSO client:", err)
		return err
	}
	fmt.Printf("SSO client created successfully for region: %s, start URL: %s\n", client.Region, client.StartURL)

	// Step 2: Register client
	fmt.Println("\nRegistering client...")
	registration, err := client.RegisterClient(ctx)
	if err != nil {
		fmt.Println("Error registering client:", err)
		return err
	}
	fmt.Println("Client registered successfully")

	// Step 3: Start device authorization
	fmt.Println("\nStarting device authorization...")
	deviceAuth, err := client.StartDeviceAuthorization(ctx, registration.ClientID, registration.ClientSecret)
	if err != nil {
		fmt.Println("Error starting device authorization:", err)
		return err
	}

	// Step 4: Show instructions to the user
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Please authorize this application:")
	fmt.Printf("Visit: %s\n", deviceAuth.VerificationURIComplete)
	fmt.Printf("Or go to: %s and enter code: %s\n", deviceAuth.VerificationURI, deviceAuth.UserCode)
	fmt.Println(strings.Repeat("=", 60))

	// Open browser automatically
	fmt.Println("\nOpening browser for authorization...")
	if err := lib.OpenBrowser(deviceAuth.VerificationURIComplete); err != nil {
		fmt.Printf("Warning: Failed to open browser automatically: %v\n", err)
		fmt.Println("Please open the URL manually.")
	}

	fmt.Println("\nWaiting for authorization...")

	// Step 5: Polling to get the token
	token, err := client.CreateToken(ctx, registration.ClientID, registration.ClientSecret, deviceAuth.DeviceCode, deviceAuth.Interval)
	if err != nil {
		fmt.Println("Error creating token:", err)
		return err
	}
	fmt.Println("\n✓ Authorization successful!")

	// Step 6: Save token to cache
	fmt.Println("Saving token to cache...")
	if err := client.SaveTokenToCache(token); err != nil {
		fmt.Println("Error saving token:", err)
		return err
	}
	fmt.Println("✓ Token saved successfully")

	if boostraping {
		// Step 7: Get all accounts and roles
		fmt.Println("\nFetching accounts and roles...")
		profiles, err := client.GetAllProfiles(ctx, token.AccessToken)
		if err != nil {
			fmt.Println("Error getting profiles:", err)
			return err
		}
		fmt.Printf("✓ Found %d profiles\n", len(profiles))

		// Step 8: Write config file
		fmt.Println("\nWriting profiles to ~/.aws/config...")
		if err := client.WriteConfigFile(profiles); err != nil {
			fmt.Println("Error writing config file:", err)
			return err
		}
		fmt.Println("✓ Config file updated successfully")
	}

	fmt.Println("\n🎉 AWS SSO sso completed!")

	return nil
}
