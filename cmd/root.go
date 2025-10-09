package cmd

import (
	"fmt"
	"os"

	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/spf13/cobra"
)

var (
	LogLevel bool

	rootCmd = &cobra.Command{
		Use:   "ark",
		Short: "A powerful CLI tool for various operations",
		Long: `ark is a modern CLI application built with Cobra and Go.
It provides a clean and efficient way to interact with various services and perform common tasks.

Example usage:
  ark aws          #	 AWS related operations
  ark kubernetes # Kubernetes
  ark --help       # Show help information`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initializeLogger()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&LogLevel, "debug", "d", false, "Set the log level to debug")
}

func Execute() {
	// First, execute the command to parse flags
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// initializeLogger initializes the logger with the current LogLevel setting
func initializeLogger() {
	logLevelName := "error"
	if LogLevel {
		fmt.Printf("Setting log level to debug\n")
		logLevelName = "debug"
	}

	if err := logs.InitLogger(logs.LogConfig{
		Level:      logLevelName,
		Format:     "console",
		OutputPath: "stdout",
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Verify logger is working
	logger := logs.GetLogger()
	if logger == nil {
		fmt.Printf("Failed to get logger instance\n")
		os.Exit(1)
	}
}
