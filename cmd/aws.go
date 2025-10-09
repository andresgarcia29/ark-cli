package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	awsCmd = &cobra.Command{
		Use:   "aws",
		Short: "AWS related operations",
		Long:  `AWS related operations`,
		Run:   aws,
	}
)

func init() {
	rootCmd.AddCommand(awsCmd)
}

func aws(cmd *cobra.Command, args []string) {
	fmt.Println("AWS related operations")
}
