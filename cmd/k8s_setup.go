package cmd

import (
	"context"

	controllers_k8s "github.com/andresgarcia29/ark-cli/controllers/kubernetes"
	"github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	"github.com/spf13/cobra"
)

var (
	setupRegions        []string
	setupClean          bool
	setupKubeconfigPath string
	setupRolePrefixes   []string
	setupReplaceProfile string
	setupRoleARN        string

	kubernetesSetupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Import EKS clusters into your kubeconfig",
		Long: `Scan every AWS account you can reach for EKS clusters and add them to your
kubeconfig. Existing contexts are kept unless --clean is passed.`,
		RunE: runKubernetesSetup,
	}
)

func init() {
	kubernetesCmd.AddCommand(kubernetesSetupCmd)
	f := kubernetesSetupCmd.Flags()
	f.StringSliceVar(&setupRegions, "regions", []string{"us-west-2"}, "AWS regions to scan")
	f.BoolVar(&setupClean, "clean", false, "Replace the kubeconfig instead of merging into it")
	f.StringVar(&setupKubeconfigPath, "kubeconfig-path", "~/.kube/config", "Path to the kubeconfig to update")
	f.StringSliceVar(&setupRolePrefixes, "role-prefixes", []string{"readonly", "read-only"}, "Prefer roles whose name contains one of these")
	f.StringVar(&setupReplaceProfile, "replace-profile", "", "Use this profile for every context")
	f.StringVar(&setupRoleARN, "role-arn", "", "Use one specific role ARN instead of matching prefixes")
	kubernetesSetupCmd.MarkFlagsMutuallyExclusive("role-prefixes", "role-arn")
}

func runKubernetesSetup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	rolePrefixes := setupRolePrefixes
	if setupRoleARN != "" {
		rolePrefixes = nil
	}

	if setupClean {
		backup, err := services_kubernetes.BackupKubeconfig(setupKubeconfigPath)
		if err != nil {
			return err
		}
		if backup != "" {
			ui.Warn("Replaced your kubeconfig")
			ui.Detail("previous version saved at %s", backup)
		}
	}

	services_aws.ResetThrottleCount()

	var clusters []services_aws.EKSCluster
	var scanErrs []error
	err := animation.Spin(ctx, "Scanning AWS accounts for EKS clusters",
		func(ctx context.Context, note animation.Progress) error {
			done := reportScanProgress(ctx, note)
			defer done()

			var err error
			clusters, scanErrs, err = services_aws.GetClustersFromAllAccounts(ctx, setupRegions, rolePrefixes, setupRoleARN)
			return err
		})
	if err != nil {
		return err
	}

	if n := services_aws.ThrottleCount(); n > 0 {
		ui.Warn("AWS rate limited %d requests; ark backed off and retried them", n)
		ui.Hint("scan fewer regions or accounts at once if this keeps happening")
	}

	for _, e := range scanErrs {
		ui.Warn("Skipped an account: %s", ui.Reason(e))
	}

	if len(clusters) == 0 {
		ui.Warn("No EKS clusters found in %v", setupRegions)
		return nil
	}

	report, err := controllers_k8s.ConfigureClusters(ctx, clusters, setupKubeconfigPath, setupReplaceProfile)
	if err != nil {
		return err
	}

	for _, e := range report.Skipped {
		ui.Warn("%s", ui.Reason(e))
	}

	ui.Done("Added %d clusters to your kubeconfig", len(report.Configured))
	ui.Hint("ark k8s")

	for _, name := range report.Configured {
		ui.Result("%s", name)
	}
	return nil
}
