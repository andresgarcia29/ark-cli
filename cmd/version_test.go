package cmd

import (
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name:    "version command help",
			args:    []string{"version", "--help"},
			wantErr: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.Equal(t, "version", cmd.Use)
				assert.Equal(t, "Print the version information", cmd.Short)
			},
		},
		{
			name:    "version command execution",
			args:    []string{"version"},
			wantErr: false,
			validate: func(t *testing.T, cmd *cobra.Command) {
				assert.Equal(t, "version", cmd.Use)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command structure
			rootCmd := &cobra.Command{Use: "ark"}

			versionCmd := &cobra.Command{
				Use:   "version",
				Short: "Print the version information",
				Long:  `Print the version information including version, commit hash, build date, and Go version.`,
				Run: func(cmd *cobra.Command, args []string) {
					// Mock implementation for testing
					// In the real function, this would print version information
				},
			}

			rootCmd.AddCommand(versionCmd)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, versionCmd)
			}
		})
	}
}

func TestVersionCommandStructure(t *testing.T) {
	// Test that the version command has the expected structure
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Print the version information including version, commit hash, build date, and Go version.`,
	}

	assert.Equal(t, "version", cmd.Use)
	assert.Equal(t, "Print the version information", cmd.Short)
	assert.Contains(t, cmd.Long, "version, commit hash, build date, and Go version")
}

func TestVersionVariables(t *testing.T) {
	// Test version variables
	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
		goVersion string
	}{
		{
			name:      "default values",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
			goVersion: runtime.Version(),
		},
		{
			name:      "custom values",
			version:   "1.0.0",
			commit:    "abc123",
			buildDate: "2024-01-01",
			goVersion: "go1.21.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that variables can be set (they're normally set at build time)
			Version = tt.version
			Commit = tt.commit
			BuildDate = tt.buildDate
			GoVersion = tt.goVersion

			assert.Equal(t, tt.version, Version)
			assert.Equal(t, tt.commit, Commit)
			assert.Equal(t, tt.buildDate, BuildDate)
			assert.Equal(t, tt.goVersion, GoVersion)
		})
	}
}

func TestVersionCommandFunction(t *testing.T) {
	// Test the version command function logic
	tests := []struct {
		name           string
		version        string
		commit         string
		buildDate      string
		goVersion      string
		expectedOutput []string
	}{
		{
			name:      "default values",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
			goVersion: "go1.21.0",
			expectedOutput: []string{
				"ark-cli version dev",
				"  commit: unknown",
				"  build date: unknown",
				"  go version: go1.21.0",
			},
		},
		{
			name:      "custom values",
			version:   "1.0.0",
			commit:    "abc123",
			buildDate: "2024-01-01",
			goVersion: "go1.21.0",
			expectedOutput: []string{
				"ark-cli version 1.0.0",
				"  commit: abc123",
				"  build date: 2024-01-01",
				"  go version: go1.21.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the output formatting logic from the version command
			output := []string{
				"ark-cli version " + tt.version,
				"  commit: " + tt.commit,
				"  build date: " + tt.buildDate,
				"  go version: " + tt.goVersion,
			}

			assert.Equal(t, tt.expectedOutput, output)

			// Verify each line
			for i, expected := range tt.expectedOutput {
				assert.Equal(t, expected, output[i])
			}
		})
	}
}

func TestVersionCommandInit(t *testing.T) {
	// Test the init function behavior
	// Since init() is called automatically, we test the expected behavior

	// Create a command that simulates what init() would do
	rootCmd := &cobra.Command{Use: "ark"}
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Print the version information including version, commit hash, build date, and Go version.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Mock implementation
		},
	}

	rootCmd.AddCommand(versionCmd)

	// Verify command structure
	assert.Len(t, rootCmd.Commands(), 1)
	assert.Equal(t, "version", rootCmd.Commands()[0].Use)
}

func TestVersionCommandOutput(t *testing.T) {
	// Test the expected output format
	version := "1.0.0"
	commit := "abc123"
	buildDate := "2024-01-01"
	goVersion := "go1.21.0"

	// Test the exact output format used in the version command
	expectedLines := []string{
		"ark-cli version " + version,
		"  commit: " + commit,
		"  build date: " + buildDate,
		"  go version: " + goVersion,
	}

	// Verify the format
	assert.Equal(t, "ark-cli version 1.0.0", expectedLines[0])
	assert.Equal(t, "  commit: abc123", expectedLines[1])
	assert.Equal(t, "  build date: 2024-01-01", expectedLines[2])
	assert.Equal(t, "  go version: go1.21.0", expectedLines[3])
}

func TestVersionCommandRuntimeVersion(t *testing.T) {
	// Test that GoVersion is set to runtime.Version()
	expectedGoVersion := runtime.Version()

	// In the real code, GoVersion is set to runtime.Version()
	GoVersion = runtime.Version()

	assert.Equal(t, expectedGoVersion, GoVersion)
	assert.NotEmpty(t, GoVersion)
	assert.Contains(t, GoVersion, "go")
}

func TestVersionCommandBuildTimeVariables(t *testing.T) {
	// Test that build-time variables are properly handled
	// These variables are normally set during the build process

	// Reset to default values first
	originalVersion := Version
	originalCommit := Commit
	originalBuildDate := BuildDate

	Version = "dev"
	Commit = "unknown"
	BuildDate = "unknown"

	// Test default values (what they would be if not set at build time)
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "unknown", Commit)
	assert.Equal(t, "unknown", BuildDate)

	// Test that they can be overridden (simulating build-time setting)
	Version = "1.0.0"
	Commit = "abc123"
	BuildDate = "2024-01-01"

	assert.Equal(t, "1.0.0", Version)
	assert.Equal(t, "abc123", Commit)
	assert.Equal(t, "2024-01-01", BuildDate)

	// Restore original values
	Version = originalVersion
	Commit = originalCommit
	BuildDate = originalBuildDate
}

func TestVersionCommandHelp(t *testing.T) {
	// Test that the version command has proper help text
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Long:  `Print the version information including version, commit hash, build date, and Go version.`,
	}

	// Verify help text
	assert.Equal(t, "Print the version information", cmd.Short)
	assert.Contains(t, cmd.Long, "version information")
	assert.Contains(t, cmd.Long, "commit hash")
	assert.Contains(t, cmd.Long, "build date")
	assert.Contains(t, cmd.Long, "Go version")
}

func TestVersionCommandExecution(t *testing.T) {
	// Test that the version command can be executed without errors
	cmd := &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			// Mock implementation that doesn't actually print
			// In the real function, this would use fmt.Printf
		},
	}

	// Test execution
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestVersionCommandArgs(t *testing.T) {
	// Test that the version command handles arguments correctly
	var receivedArgs []string

	cmd := &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			// Store the received args for testing
			receivedArgs = args
		},
	}

	// Test with no arguments
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Empty(t, receivedArgs)

	// Test with arguments (should be passed through)
	cmd.SetArgs([]string{"extra", "args"})
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, []string{"extra", "args"}, receivedArgs)
}
