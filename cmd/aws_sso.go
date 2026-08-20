package cmd

import (
	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	"github.com/spf13/cobra"
)

var (
	ssoRegion   string
	ssoStartURL string

	awsSSOCmd = &cobra.Command{
		Use:   "sso",
		Short: "Start a new SSO session",
		Long:  "Authorize through the browser, then import every account and role you can reach into ~/.aws/config.",
		RunE:  runAWSSSO,
	}
)

func init() {
	awsCmd.AddCommand(awsSSOCmd)
	awsSSOCmd.Flags().StringVar(&ssoRegion, "region", "us-east-1", "AWS SSO region")
	awsSSOCmd.Flags().StringVar(&ssoStartURL, "start-url", "", "AWS SSO start URL")
	if err := awsSSOCmd.MarkFlagRequired("start-url"); err != nil {
		panic(err)
	}
}

func runAWSSSO(cmd *cobra.Command, args []string) error {
	return controllers.SSOLogin(cmd.Context(), ssoRegion, ssoStartURL, true)
}
