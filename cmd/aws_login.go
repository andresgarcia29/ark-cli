package cmd

import (
	"context"
	"fmt"

	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	"github.com/spf13/cobra"
)

var (
	awsLoginnCmd = &cobra.Command{
		Use:   "login",
		Short: "Start a new AWS Login session",
		Long:  "Configure and start a new AWS Login session with the provided profile, fetching the credentials from the AWS Login cache",
		Run:   awsLoginCommand,
	}
)

var (
	LoginProfile string
	SetAsDefault bool
)

func init() {
	awsCmd.AddCommand(awsLoginnCmd)
	awsLoginnCmd.Flags().StringVar(&LoginProfile, "profile", "", "AWS profile name to login with")
	awsLoginnCmd.Flags().BoolVar(&SetAsDefault, "set-default", false, "Set this profile as default")
	if err := awsLoginnCmd.MarkFlagRequired("profile"); err != nil {
		panic(err)
	}
}

func awsLoginCommand(cmd *cobra.Command, args []string) {
	profileName := cmd.Flag("profile").Value.String()
	setAsDefault, _ := cmd.Flags().GetBool("set-default")

	if profileName == "" {
		fmt.Println("Error: --profile flag is required")
		return
	}

	fmt.Printf("Logging in with profile: %s\n", profileName)

	ctx := context.Background()

	// Resolve SSO configuration (can come from source profile for assume role)
	ssoRegion, ssoStartURL, err := services_aws.ResolveSSOConfiguration(profileName)
	if err != nil {
		fmt.Printf("Error resolving SSO configuration: %v\n", err)
		return
	}

	fmt.Printf("✅ Resolved SSO configuration - Region: %s, Start URL: %s\n", ssoRegion, ssoStartURL)

	// Use retry function for login
	if err := controllers.AttemptLoginWithRetry(ctx, profileName, setAsDefault, ssoRegion, ssoStartURL); err != nil {
		fmt.Printf("❌ Login failed after retry: %v\n", err)
		return
	}

	if setAsDefault {
		fmt.Printf("✓ Successfully logged in with profile '%s' and set as default\n", profileName)
	} else {
		fmt.Printf("✓ Successfully logged in with profile '%s'\n", profileName)
	}
}
