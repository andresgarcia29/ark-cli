package cmd

import (
	"runtime"

	"github.com/andresgarcia29/ark-cli/lib/ui"
	"github.com/spf13/cobra"
)

// Build metadata, injected via -ldflags at release time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		ui.Result("ark %s", Version)
		ui.Result("commit  %s", Commit)
		ui.Result("built   %s", BuildDate)
		ui.Result("go      %s", runtime.Version())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
