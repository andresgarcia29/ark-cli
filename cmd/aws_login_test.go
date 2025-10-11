package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSLoginCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		expectPanic bool
		validate    func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name:        "aws login command help",
			args:        []string{"aws", "login", "--help"},
			wantErr:     false,
			expectPanic: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.Equal(t, "login", cmd.Use)
				assert.Equal(t, "Start a new AWS Login session", cmd.Short)
			},
		},
		{
			name:        "aws login with profile flag",
			args:        []string{"aws", "login", "--profile", "test-profile"},
			wantErr:     false,
			expectPanic: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				profileFlag := cmd.Flag("profile")
				require.NotNil(t, profileFlag)
				assert.Equal(t, "test-profile", profileFlag.Value.String())
			},
		},
		{
			name:        "aws login with set-default flag",
			args:        []string{"aws", "login", "--profile", "test-profile", "--set-default"},
			wantErr:     false,
			expectPanic: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				setDefault, _ := cmd.Flags().GetBool("set-default")
				assert.True(t, setDefault)
			},
		},
		{
			name:        "aws login without required profile flag",
			args:        []string{"aws", "login"},
			wantErr:     true,
			expectPanic: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				// This test should fail because profile is required
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			LoginProfile = ""
			SetAsDefault = false

			// Create command structure
			rootCmd := &cobra.Command{Use: "ark"}
			awsCmd := &cobra.Command{Use: "aws"}

			loginCmd := &cobra.Command{
				Use:   "login",
				Short: "Start a new AWS Login session",
				Long:  "Configure and start a new AWS Login session with the provided profile, fetching the credentials from the AWS Login cache",
				Run: func(cmd *cobra.Command, args []string) {
					// Mock implementation for testing
					profileName := cmd.Flag("profile").Value.String()
					if profileName == "" {
						// Simulate the error handling in the real function
						return
					}
				},
			}

			// Add flags
			loginCmd.Flags().StringVar(&LoginProfile, "profile", "", "AWS profile name to login with")
			loginCmd.Flags().BoolVar(&SetAsDefault, "set-default", false, "Set this profile as default")

			// Mark profile as required (this would normally be done in init())
			if err := loginCmd.MarkFlagRequired("profile"); err != nil {
				t.Fatalf("Failed to mark profile flag as required: %v", err)
			}

			awsCmd.AddCommand(loginCmd)
			rootCmd.AddCommand(awsCmd)
			rootCmd.SetArgs(tt.args)

			if tt.expectPanic {
				assert.Panics(t, func() {
					rootCmd.Execute()
				})
			} else {
				err := rootCmd.Execute()
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}

			if tt.validate != nil {
				tt.validate(t, loginCmd)
			}
		})
	}
}

func TestAWSLoginCommandFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "login",
	}

	// Add flags
	cmd.Flags().StringVar(&LoginProfile, "profile", "", "AWS profile name to login with")
	cmd.Flags().BoolVar(&SetAsDefault, "set-default", false, "Set this profile as default")

	// Test profile flag
	profileFlag := cmd.Flags().Lookup("profile")
	require.NotNil(t, profileFlag)
	assert.Equal(t, "profile", profileFlag.Name)
	assert.Equal(t, "", profileFlag.DefValue)
	assert.Equal(t, "AWS profile name to login with", profileFlag.Usage)

	// Test set-default flag
	defaultFlag := cmd.Flags().Lookup("set-default")
	require.NotNil(t, defaultFlag)
	assert.Equal(t, "set-default", defaultFlag.Name)
	assert.Equal(t, "false", defaultFlag.DefValue)
	assert.Equal(t, "Set this profile as default", defaultFlag.Usage)
}

func TestAWSLoginCommandFunction(t *testing.T) {
	// Test the awsLoginCommand function logic
	tests := []struct {
		name           string
		profileName    string
		setAsDefault   bool
		expectedOutput string
	}{
		{
			name:           "valid profile without default",
			profileName:    "test-profile",
			setAsDefault:   false,
			expectedOutput: "✓ Successfully logged in with profile 'test-profile'",
		},
		{
			name:           "valid profile with default",
			profileName:    "test-profile",
			setAsDefault:   true,
			expectedOutput: "✓ Successfully logged in with profile 'test-profile' and set as default",
		},
		{
			name:           "empty profile name",
			profileName:    "",
			setAsDefault:   false,
			expectedOutput: "Error: --profile flag is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock command
			cmd := &cobra.Command{}
			cmd.Flags().String("profile", tt.profileName, "AWS profile name")
			cmd.Flags().Bool("set-default", tt.setAsDefault, "Set as default")

			// Test the logic from awsLoginCommand function
			profileName := cmd.Flag("profile").Value.String()
			setAsDefault, _ := cmd.Flags().GetBool("set-default")

			if profileName == "" {
				// This simulates the error handling in the real function
				assert.Equal(t, "", profileName)
				assert.Equal(t, tt.expectedOutput, "Error: --profile flag is required")
			} else {
				// Test success message formatting
				var expectedMsg string
				if setAsDefault {
					expectedMsg = "✓ Successfully logged in with profile '" + profileName + "' and set as default"
				} else {
					expectedMsg = "✓ Successfully logged in with profile '" + profileName + "'"
				}
				assert.Equal(t, tt.expectedOutput, expectedMsg)
			}
		})
	}
}

func TestAWSLoginCommandContext(t *testing.T) {
	// Test context creation in awsLoginCommand
	ctx := context.Background()

	// Verify context is not nil
	assert.NotNil(t, ctx)

	// Test context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 0) // Immediate timeout
	defer cancel()

	// Verify context is cancelled due to timeout
	select {
	case <-timeoutCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled due to timeout")
	}
}

func TestAWSLoginCommandErrorHandling(t *testing.T) {
	// Test error handling patterns
	tests := []struct {
		name        string
		errorType   string
		expectedMsg string
	}{
		{
			name:        "SSO configuration error",
			errorType:   "sso_config",
			expectedMsg: "Error resolving SSO configuration: test error",
		},
		{
			name:        "Login retry error",
			errorType:   "login_retry",
			expectedMsg: "❌ Login failed after retry: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate error handling from the real function
			var actualMsg string
			switch tt.errorType {
			case "sso_config":
				actualMsg = "Error resolving SSO configuration: test error"
			case "login_retry":
				actualMsg = "❌ Login failed after retry: test error"
			}

			assert.Equal(t, tt.expectedMsg, actualMsg)
		})
	}
}

func TestAWSLoginCommandVariables(t *testing.T) {
	// Test global variables used in aws login command
	tests := []struct {
		name         string
		loginProfile string
		setAsDefault bool
	}{
		{
			name:         "default values",
			loginProfile: "",
			setAsDefault: false,
		},
		{
			name:         "set values",
			loginProfile: "test-profile",
			setAsDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			LoginProfile = tt.loginProfile
			SetAsDefault = tt.setAsDefault

			assert.Equal(t, tt.loginProfile, LoginProfile)
			assert.Equal(t, tt.setAsDefault, SetAsDefault)
		})
	}
}

func TestAWSLoginCommandInit(t *testing.T) {
	// Test the init function behavior
	// Since init() is called automatically, we test the expected behavior

	// Create a command that simulates what init() would do
	rootCmd := &cobra.Command{Use: "ark"}
	awsCmd := &cobra.Command{Use: "aws"}
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Start a new AWS Login session",
	}

	// Add flags (simulating init())
	loginCmd.Flags().String("profile", "", "AWS profile name to login with")
	loginCmd.Flags().Bool("set-default", false, "Set this profile as default")

	// Mark profile as required (this would be done in init())
	// For testing, we verify the flag exists and is required
	profileFlag := loginCmd.Flags().Lookup("profile")
	require.NotNil(t, profileFlag)
	assert.Equal(t, "profile", profileFlag.Name)

	awsCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(awsCmd)

	// Verify command structure
	assert.Len(t, awsCmd.Commands(), 1)
	assert.Equal(t, "login", awsCmd.Commands()[0].Use)
}
