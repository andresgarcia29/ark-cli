package cmd

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/andresgarcia29/ark-cli/lib/ui"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	"github.com/spf13/cobra"
)

var kubernetesDiagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Check that kubectl, AWS and the network are usable",
	Long:  "Run every prerequisite check ark depends on and report what to fix.",
	RunE:  runKubernetesDiagnose,
}

func init() {
	kubernetesCmd.AddCommand(kubernetesDiagnoseCmd)
}

// check is one diagnostic. It returns a detail line and, when it fails, the
// command that would fix it.
type check struct {
	name string
	run  func(context.Context) (detail string, err error)
	fix  string
}

func runKubernetesDiagnose(cmd *cobra.Command, args []string) error {
	checks := []check{
		{
			name: "kubectl installed",
			run: func(ctx context.Context) (string, error) {
				path, err := exec.LookPath("kubectl")
				return path, err
			},
			fix: "install kubectl and make sure it is on your PATH",
		},
		{
			name: "aws cli installed",
			run: func(ctx context.Context) (string, error) {
				path, err := exec.LookPath("aws")
				return path, err
			},
			fix: "install the AWS CLI; ark uses it to write kubeconfig entries",
		},
		{
			name: "kubeconfig readable",
			run: func(ctx context.Context) (string, error) {
				path, err := services_kubernetes.ExpandPath("~/.kube/config")
				if err != nil {
					return "", err
				}
				contexts, err := services_kubernetes.GetClusterContexts(ctx)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("%d contexts in %s", len(contexts), path), nil
			},
			fix: "run 'ark k8s setup' to import your EKS clusters",
		},
		{
			name: "aws credentials valid",
			run: func(ctx context.Context) (string, error) {
				out, err := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text").Output()
				if err != nil {
					return "", fmt.Errorf("sts get-caller-identity was rejected")
				}
				return "account " + strings.TrimSpace(string(out)), nil
			},
			fix: "run 'ark aws' to sign in",
		},
		{
			name: "eks endpoint reachable",
			run: func(ctx context.Context) (string, error) {
				// A TCP dial, not ping: AWS endpoints drop ICMP, so pinging
				// them reports a failure on perfectly healthy networks.
				d := net.Dialer{Timeout: 5 * time.Second}
				conn, err := d.DialContext(ctx, "tcp", "eks.us-west-2.amazonaws.com:443")
				if err != nil {
					return "", err
				}
				defer func() { _ = conn.Close() }()
				return "tcp/443 open", nil
			},
			fix: "check your network, VPN or proxy settings",
		},
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	ui.Heading("Diagnostics")
	failed := 0
	for _, c := range checks {
		detail, err := c.run(ctx)
		if err != nil {
			failed++
			ui.Fail("%s", c.name)
			ui.Detail("%s", ui.Reason(err))
			ui.Hint("%s", c.fix)
			continue
		}
		ui.Done("%s", c.name)
		if detail != "" {
			ui.Detail("%s", detail)
		}
	}

	ui.Blank()
	if failed > 0 {
		ui.Fail("%d of %d checks failed", failed, len(checks))
		return errQuiet
	}
	ui.Done("Everything checks out")
	return nil
}
