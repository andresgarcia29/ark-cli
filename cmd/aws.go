package cmd

import (
	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	"github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	"github.com/spf13/cobra"
)

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "AWS access",
	Long:  "Pick one of your configured AWS profiles and sign in.",
	RunE:  runAWS,
}

func init() {
	rootCmd.AddCommand(awsCmd)
}

func runAWS(cmd *cobra.Command, args []string) error {
	profiles, err := services_aws.ReadAllProfilesFromConfig()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		ui.Fail("No profiles found in ~/.aws/config")
		ui.Hint("ark aws sso --start-url <your SSO url>")
		return errQuiet
	}

	profile, err := animation.SelectProfile(profiles)
	if err != nil {
		return err
	}

	if err := controllers.Login(cmd.Context(), profile.ProfileName, true); err != nil {
		return err
	}

	ui.Done("Signed in as %s", ui.Strong.Render(profile.ProfileName))
	return nil
}
