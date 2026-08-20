package cmd

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	"github.com/andresgarcia29/ark-cli/logs"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var (
	debug bool
	quiet bool

	rootCmd = &cobra.Command{
		Use:   "ark",
		Short: "AWS SSO and EKS access from one command",
		Long: `ark signs you into AWS through SSO and keeps your kubeconfig in sync
with the EKS clusters you can reach.

  ark aws            Pick a profile and sign in
  ark aws sso        Start a new SSO session and import every profile
  ark k8s            Switch between clusters
  ark k8s setup      Import EKS clusters into your kubeconfig`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ui.SetQuiet(quiet)
			logs.Init(debug)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Show internal diagnostics on stderr")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Only print results and failures")
}

// Execute runs the CLI and exits non-zero when a command fails, so ark can be
// chained with && in scripts.
func Execute() {
	// Ctrl+C cancels in-flight AWS calls instead of leaving the terminal in
	// whatever state a TUI was using.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := fang.Execute(ctx, rootCmd,
		fang.WithColorSchemeFunc(helpScheme),
		fang.WithVersion(Version),
		fang.WithCommit(Commit),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
		// ark reports its own failures through ui.Fail, in one voice with the
		// rest of its output; fang's error box would print them a second time.
		fang.WithErrorHandler(func(io.Writer, fang.Styles, error) {}),
	)
	logs.Sync()

	switch {
	case err == nil:
		return
	case errors.Is(err, animation.ErrCancelled), errors.Is(err, context.Canceled):
		// The user backed out on purpose; that is not a failure to report.
		os.Exit(130)
	case errors.Is(err, errQuiet):
		// The command already explained itself.
		os.Exit(1)
	default:
		ui.Fail("%s", ui.Reason(err))
		os.Exit(1)
	}
}
