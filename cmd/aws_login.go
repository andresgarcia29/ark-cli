package cmd

import (
	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	"github.com/spf13/cobra"
)

var (
	loginProfile string
	setAsDefault bool

	awsLoginCmd = &cobra.Command{
		Use:   "login",
		Short: "Sign in with a named profile",
		Long:  "Fetch credentials for a profile, refreshing the SSO session if it has expired.",
		RunE:  runAWSLogin,
	}
)

func init() {
	awsCmd.AddCommand(awsLoginCmd)
	awsLoginCmd.Flags().StringVar(&loginProfile, "profile", "", "AWS profile to sign in with")
	awsLoginCmd.Flags().BoolVar(&setAsDefault, "set-default", false, "Also write the credentials to [default]")
	if err := awsLoginCmd.MarkFlagRequired("profile"); err != nil {
		panic(err)
	}
}

func runAWSLogin(cmd *cobra.Command, args []string) error {
	if err := controllers.Login(cmd.Context(), loginProfile, setAsDefault); err != nil {
		return err
	}

	if setAsDefault {
		ui.Done("Signed in as %s and set as default", ui.Strong.Render(loginProfile))
	} else {
		ui.Done("Signed in as %s", ui.Strong.Render(loginProfile))
	}
	return nil
}
