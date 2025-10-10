package cmd

import (
	"context"
	"fmt"

	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	"github.com/spf13/cobra"
)

var (
	SSORegion   string
	SSOStartURL string

	awsSSOnCmd = &cobra.Command{
		Use:   "sso",
		Short: "Start a new AWS SSO session",
		Long:  "Configure and start a new AWS SSO session with the provided profile, fetching the credentials from the AWS SSO cache",
		Run:   awsSSOCommand,
	}
)

func init() {
	awsCmd.AddCommand(awsSSOnCmd)
	awsSSOnCmd.Flags().StringVar(&SSORegion, "region", "us-east-1", "AWS SSO region")
	awsSSOnCmd.Flags().StringVar(&SSOStartURL, "start-url", "", "AWS SSO start URL (required)")
	if err := awsSSOnCmd.MarkFlagRequired("start-url"); err != nil {
		panic(err)
	}
}

func awsSSOCommand(cmd *cobra.Command, args []string) {
	fmt.Println("AWS sso")
	ctx := context.Background()

	if err := controllers.AWSSSOLogin(ctx, SSORegion, SSOStartURL, true); err != nil {
		fmt.Println("Error:", err)
		return
	}
}
