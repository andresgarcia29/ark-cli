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
	kubernetesCmd = &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"k8s", "eks"},
		Short:   "AWS related operations",
		Long:    `AWS related operations`,
		Run:     kubernetes,
	}
)

func init() {
	rootCmd.AddCommand(kubernetesCmd)
	kubernetesCmd.Flags().StringSlice("regions", []string{"us-west-2"}, "List of AWS regions to scan")
	kubernetesCmd.Flags().Bool("clean", true, "Clean kubeconfig before configuring")
	kubernetesCmd.Flags().Bool("set-up", false, "Configure kubeconfig")
	kubernetesCmd.Flags().String("kubeconfig-path", "~/.kube/config", "Path to kubeconfig")
	kubernetesCmd.Flags().StringSlice("role-prefixs", []string{"readonly", "read-only"}, "Role prefixs to scan")
	kubernetesCmd.Flags().String("replace-profile", "", "Replace profile in kubeconfig")
}

// ConfigureAllEKSClusters es el flujo completo para configurar todos los clusters EKS
func ConfigureAllEKSClusters(ctx context.Context, regions []string, cleanKubeconfig bool, kubeconfigPath string, rolePrefixs []string, replaceProfile string) error {
	// Paso 1: Limpiar kubeconfig si se requiere
	if cleanKubeconfig {
		fmt.Println("🧹 Cleaning kubeconfig...")
		if err := services_kubernetes.CleanKubeconfig(kubeconfigPath); err != nil {
			return fmt.Errorf("failed to clean kubeconfig: %w", err)
		}
		fmt.Println()
	}

	// Paso 2: Obtener todos los clusters de todas las cuentas con spinner
	var clusters []services_aws.EKSCluster
	err := animation.ShowSpinner("Fetching EKS clusters from all accounts", func() error {
		var err error
		clusters, err = services_aws.GetClustersFromAllAccounts(ctx, regions, rolePrefixs)
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

	// Mostrar resumen de clusters por cuenta
	accountClusters := make(map[string]int)
	for _, cluster := range clusters {
		accountClusters[cluster.AccountID]++
	}
	fmt.Println("\nClusters by account:")
	for accountID, count := range accountClusters {
		fmt.Printf("  - Account %s: %d cluster(s)\n", accountID, count)
	}

	fmt.Println()

	// Paso 3: Configurar kubeconfig para todos los clusters con progress bar
	if err := controllers_k8s.UpdateKubeconfigWithProgress(clusters, replaceProfile); err != nil {
		return fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	return nil
}

func kubernetes(cmd *cobra.Command, args []string) {
	regions, _ := cmd.Flags().GetStringSlice("regions")
	cleanConfig, _ := cmd.Flags().GetBool("clean")
	setUp, _ := cmd.Flags().GetBool("set-up")
	kubeconfigPath, _ := cmd.Flags().GetString("kubeconfig-path")
	replaceProfile, _ := cmd.Flags().GetString("replace-profile")
	rolePrefixs, _ := cmd.Flags().GetStringSlice("role-prefixs")

	ctx := context.Background()

	if rolePrefixs == nil {
		fmt.Println("No role prefixs provided, using default prefixs: readonly, read-only")
		rolePrefixs = []string{"readonly", "read-only"}
	}

	if setUp {
		if err := ConfigureAllEKSClusters(ctx, regions, cleanConfig, kubeconfigPath, rolePrefixs, replaceProfile); err != nil {
			fmt.Println("Error:", err)
			return
		}
	} else {
		fmt.Println("Skipping kubeconfig configuration")
	}
}
