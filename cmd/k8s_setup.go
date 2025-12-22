package cmd

import (
	"context"
	"fmt"

	controllers_k8s "github.com/andresgarcia29/ark-cli/controllers/kubernetes"
	"github.com/andresgarcia29/ark-cli/lib/animation"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	"github.com/spf13/cobra"
)

var (
	kubernetesSetupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Setup and configure EKS clusters in kubeconfig",
		Long:  `Setup and configure EKS clusters in kubeconfig by fetching clusters from all AWS accounts and updating the kubeconfig file.`,
		Run:   kubernetesSetup,
	}
)

func init() {
	kubernetesCmd.AddCommand(kubernetesSetupCmd)
	kubernetesSetupCmd.Flags().StringSlice("regions", []string{"us-west-2"}, "List of AWS regions to scan")
	kubernetesSetupCmd.Flags().Bool("clean", true, "Clean kubeconfig before configuring")
	kubernetesSetupCmd.Flags().String("kubeconfig-path", "~/.kube/config", "Path to kubeconfig")
	kubernetesSetupCmd.Flags().StringSlice("role-prefixs", []string{"readonly", "read-only"}, "Role prefixs to scan")
	kubernetesSetupCmd.Flags().String("replace-profile", "", "Replace profile in kubeconfig")
	kubernetesSetupCmd.Flags().String("role-arn", "", "Specific Role ARN to use for authentication (mutually exclusive with role-prefixs)")
}

// ConfigureAllEKSClusters is the complete flow to configure all EKS clusters
func ConfigureAllEKSClusters(ctx context.Context, regions []string, cleanKubeconfig bool, kubeconfigPath string, rolePrefixs []string, replaceProfile string, roleARN string) error {
	// Step 1: Clean kubeconfig if required
	if cleanKubeconfig {
		fmt.Println("🧹 Cleaning kubeconfig...")
		if err := services_kubernetes.CleanKubeconfig(kubeconfigPath); err != nil {
			return fmt.Errorf("failed to clean kubeconfig: %w", err)
		}
		fmt.Println()
	}

	// Step 2: Get all clusters from all accounts with a spinner
	var clusters []services_aws.EKSCluster
	err := animation.ShowSpinner("Fetching EKS clusters from all accounts", func() error {
		var err error
		clusters, err = services_aws.GetClustersFromAllAccounts(ctx, regions, rolePrefixs, roleARN)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to get clusters: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Println("\nNo EKS clusters found in any account")
		return nil
	}

	fmt.Printf("\n✓ Total clusters found: %d\n", len(clusters))

	// Show clusters summary per account
	accountClusters := make(map[string]int)
	for _, cluster := range clusters {
		accountClusters[cluster.AccountID]++
	}
	fmt.Println("\nClusters by account:")
	for accountID, count := range accountClusters {
		fmt.Printf("  - Account %s: %d cluster(s)\n", accountID, count)
	}

	fmt.Println()

	// Step 3: Configure kubeconfig for all clusters with progress bar
	if err := controllers_k8s.UpdateKubeconfigWithProgress(clusters, replaceProfile); err != nil {
		return fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	return nil
}

func kubernetesSetup(cmd *cobra.Command, args []string) {
	regions, _ := cmd.Flags().GetStringSlice("regions")
	cleanConfig, _ := cmd.Flags().GetBool("clean")
	kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig-path")
	replaceProfile, _ := cmd.Flags().GetString("replace-profile")
	rolePrefixs, _ := cmd.Flags().GetStringSlice("role-prefixs")
	roleARN, _ := cmd.Flags().GetString("role-arn")

	ctx := context.Background()

	// Validate flags exclusivity
	if cmd.Flags().Changed("role-prefixs") && cmd.Flags().Changed("role-arn") {
		fmt.Println("Error: --role-prefixs and --role-arn are mutually exclusive")
		return
	}

	// If role-arn is provided, we don't use prefixes
	if roleARN != "" {
		rolePrefixs = nil
	} else if !cmd.Flags().Changed("role-prefixs") {
		// Only use defaults if the flag hasn't changed and there is no ARN
		fmt.Println("No role prefixs or ARN provided, using default prefixs: readonly, read-only")
		rolePrefixs = []string{"readonly", "read-only"}
	}

	if err := ConfigureAllEKSClusters(ctx, regions, cleanConfig, kubeconfigPath, rolePrefixs, replaceProfile, roleARN); err != nil {
		fmt.Println("Error:", err)
		return
	}
}
