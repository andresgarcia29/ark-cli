package cmd

import (
	"context"
	"fmt"

	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	animation "github.com/andresgarcia29/ark-cli/lib/animation"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	"github.com/spf13/cobra"
)

var (
	awsCmd = &cobra.Command{
		Use:   "aws",
		Short: "AWS related operations",
		Long:  `AWS related operations - Interactive profile selection and login`,
		Run:   aws,
	}
)

func init() {
	rootCmd.AddCommand(awsCmd)
}

func aws(cmd *cobra.Command, args []string) {
	// Create context
	ctx := context.Background()

	// Show interactive profile selector
	selectedProfile, err := animation.InteractiveProfileSelector()
	if err != nil {
		fmt.Printf("❌ Error selecting profile: %v\n", err)
		return
	}

	// Show selected profile information
	fmt.Printf("\n✅ Selected profile: %s (%s)\n", selectedProfile.ProfileName, selectedProfile.ProfileType)
	fmt.Println("🔐 Logging in...")

	// Resolve SSO configuration (can come from source profile for assume role)
	ssoRegion, ssoStartURL, err := services_aws.ResolveSSOConfiguration(selectedProfile.ProfileName)
	if err != nil {
		fmt.Printf("Error resolving SSO configuration: %v\n", err)
		return
	}

	// Perform login with the selected profile using retry
	if err := controllers.AttemptLoginWithRetry(ctx, selectedProfile.ProfileName, true, ssoRegion, ssoStartURL); err != nil {
		fmt.Printf("❌ Login failed after retry: %v\n", err)
		return
	}

	fmt.Printf("🎉 Successfully logged in with profile: %s\n", selectedProfile.ProfileName)
	fmt.Println("💡 You can now use AWS CLI commands with this profile")
}
