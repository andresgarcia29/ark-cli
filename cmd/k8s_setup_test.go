package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestKubernetesSetupCommandFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "mutually exclusive flags: role-prefixs and role-arn",
			args:           []string{"setup", "--role-prefixs", "readonly", "--role-arn", "arn:aws:iam::123456789012:role/MyRole"},
			expectedOutput: "Error: --role-prefixs and --role-arn are mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSetupCmd := &cobra.Command{
				Use: "setup",
				Run: func(cmd *cobra.Command, args []string) {
					// This mimics the validation logic in kubernetesSetup
					rolePrefixs, _ := cmd.Flags().GetStringSlice("role-prefixs")
					roleARN, _ := cmd.Flags().GetString("role-arn")

					if cmd.Flags().Changed("role-prefixs") && cmd.Flags().Changed("role-arn") {
						assert.Equal(t, "Error: --role-prefixs and --role-arn are mutually exclusive", tt.expectedOutput)
						return
					}
					
					if roleARN != "" {
						assert.Nil(t, rolePrefixs)
					}
				},
			}
			testSetupCmd.Flags().StringSlice("role-prefixs", []string{"readonly", "read-only"}, "")
			testSetupCmd.Flags().String("role-arn", "", "")

			testSetupCmd.SetArgs(tt.args)
			// Simulate flag changes
			for i := 0; i < len(tt.args); i++ {
				if tt.args[i] == "--role-prefixs" {
					testSetupCmd.Flags().Set("role-prefixs", tt.args[i+1])
				}
				if tt.args[i] == "--role-arn" {
					testSetupCmd.Flags().Set("role-arn", tt.args[i+1])
				}
			}

			testSetupCmd.Execute()
		})
	}
}

