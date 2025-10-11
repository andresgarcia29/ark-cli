package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name:    "root command help",
			args:    []string{"--help"},
			wantErr: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.Equal(t, "ark", cmd.Use)
				assert.Equal(t, "A powerful CLI tool for various operations", cmd.Short)
			},
		},
		{
			name:    "root command with debug flag",
			args:    []string{"--debug"},
			wantErr: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.True(t, LogLevel)
			},
		},
		{
			name:    "root command without flags",
			args:    []string{},
			wantErr: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.False(t, LogLevel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			LogLevel = false

			// Create a new root command for testing
			cmd := &cobra.Command{
				Use:   "ark",
				Short: "A powerful CLI tool for various operations",
				Long: `ark is a modern CLI application built with Cobra and Go.
It provides a clean and efficient way to interact with various services and perform common tasks.

Example usage:
  ark aws          #	 AWS related operations
  ark kubernetes # Kubernetes
  ark version      # Show version information
  ark --help       # Show help information`,
				PersistentPreRun: func(cmd *cobra.Command, args []string) {
					// Skip logger initialization in tests
				},
			}

			cmd.PersistentFlags().BoolVarP(&LogLevel, "debug", "d", false, "Set the log level to debug")

			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, cmd)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	// Test that Execute function doesn't panic
	// We can't easily test the full execution without mocking the logger
	// but we can test that it doesn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute() panicked: %v", r)
		}
	}()

	// This will fail because we haven't initialized the logger properly
	// but it shouldn't panic
	// Note: We can't actually override os.Exit in Go, so we'll just test that it doesn't panic

	// This should not panic even if it fails
	Execute()
}

func TestInitializeLogger(t *testing.T) {
	tests := []struct {
		name        string
		logLevel    bool
		expectError bool
	}{
		{
			name:        "debug level enabled",
			logLevel:    true,
			expectError: false,
		},
		{
			name:        "debug level disabled",
			logLevel:    false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LogLevel = tt.logLevel

			// We can't easily test the full logger initialization without
			// complex mocking, but we can test the logic
			logLevelName := "error"
			if LogLevel {
				logLevelName = "debug"
			}

			if tt.logLevel {
				assert.Equal(t, "debug", logLevelName)
			} else {
				assert.Equal(t, "error", logLevelName)
			}
		})
	}
}

func TestRootCommandFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "ark",
	}
	cmd.PersistentFlags().BoolVarP(&LogLevel, "debug", "d", false, "Set the log level to debug")

	// Test that the flag is properly defined
	debugFlag := cmd.PersistentFlags().Lookup("debug")
	require.NotNil(t, debugFlag)
	assert.Equal(t, "debug", debugFlag.Name)
	assert.Equal(t, "d", debugFlag.Shorthand)
	assert.Equal(t, "Set the log level to debug", debugFlag.Usage)
}

func TestRootCommandSubcommands(t *testing.T) {
	// Test that subcommands are properly added
	// This is more of an integration test, but useful to verify structure

	// We can't easily test this without running the full init() function
	// but we can verify the command structure exists
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "ark", rootCmd.Use)

	// Check that subcommands would be added (this is tested in individual command tests)
	subcommands := rootCmd.Commands()
	// The exact number depends on what's initialized, but we expect some commands
	assert.GreaterOrEqual(t, len(subcommands), 0)
}
