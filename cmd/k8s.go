package cmd

import (
	"context"
	"fmt"
	"time"

	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	"github.com/andresgarcia29/ark-cli/lib/animation"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	"github.com/spf13/cobra"
)

var (
	kubernetesCmd = &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"k8s", "eks"},
		Short:   "Kubernetes cluster operations",
		Long:    `Kubernetes cluster operations - List and switch between cluster contexts`,
		Run:     kubernetes,
	}
)

func init() {
	rootCmd.AddCommand(kubernetesCmd)
}

func kubernetes(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Add timeout to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Mostrar selector interactivo de clusters with timeout
	fmt.Println("🔍 Loading cluster contexts...")
	selectedCluster, err := interactiveClusterSelectorWithTimeout(timeoutCtx)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			fmt.Printf("❌ Timeout: Cluster selector took too long to respond\n")
			fmt.Println("💡 This might be due to:")
			fmt.Println("   - Network connectivity issues")
			fmt.Println("   - kubectl configuration problems")
			fmt.Println("   - AWS credentials issues")
			fmt.Println("   - Try running with --debug flag for more details")
		} else {
			fmt.Printf("❌ Error selecting cluster: %v\n", err)
		}
		return
	}

	// Mostrar información del cluster seleccionado
	fmt.Printf("\n✅ Selected cluster: %s", selectedCluster.Name)
	if selectedCluster.Current {
		fmt.Printf(" (currently active)")
	}
	fmt.Println()

	// Si el cluster ya está activo, verificar si necesitamos asumir el rol
	if selectedCluster.Current {
		fmt.Println("🎉 This cluster is already active!")

		// Si hay un perfil asociado, verificar si necesitamos asumir el rol
		if selectedCluster.Profile != "" {
			fmt.Printf("🔍 Checking if we need to assume role for profile: %s\n", selectedCluster.Profile)
			if err := assumeRoleForCluster(ctx, selectedCluster); err != nil {
				fmt.Printf("❌ Failed to assume role: %v\n", err)
				return
			}
		}
		return
	}

	// Si hay un perfil asociado, asumir el rol antes de cambiar de contexto
	if selectedCluster.Profile != "" {
		fmt.Printf("🔐 Assuming role for profile: %s\n", selectedCluster.Profile)
		if err := assumeRoleForCluster(ctx, selectedCluster); err != nil {
			fmt.Printf("❌ Failed to assume role: %v\n", err)
			return
		}
	}

	// Cambiar al cluster seleccionado
	fmt.Println("🔄 Switching to cluster context...")
	if err := services_kubernetes.SwitchToContext(selectedCluster.Name); err != nil {
		fmt.Printf("❌ Failed to switch to cluster: %v\n", err)
		return
	}

	fmt.Printf("🎉 Successfully switched to cluster: %s\n", selectedCluster.Name)
	fmt.Println("💡 You can now use kubectl commands with this cluster")
}

// assumeRoleForCluster assumes the AWS role for the given cluster
func assumeRoleForCluster(ctx context.Context, cluster *services_kubernetes.ClusterContext) error {
	if cluster.Profile == "" {
		return fmt.Errorf("no profile associated with cluster %s", cluster.Name)
	}

	// Resolver configuración SSO (puede venir del source profile para assume role)
	ssoRegion, ssoStartURL, err := services_aws.ResolveSSOConfiguration(cluster.Profile)
	if err != nil {
		return fmt.Errorf("error resolving SSO configuration for profile %s: %w", cluster.Profile, err)
	}

	// Realizar login con el perfil usando retry
	if err := controllers.AttemptLoginWithRetry(ctx, cluster.Profile, true, ssoRegion, ssoStartURL); err != nil {
		return fmt.Errorf("failed to login with profile %s: %w", cluster.Profile, err)
	}

	fmt.Printf("✅ Successfully assumed role for profile: %s\n", cluster.Profile)
	return nil
}

// interactiveClusterSelectorWithTimeout wraps the cluster selector with timeout handling
func interactiveClusterSelectorWithTimeout(ctx context.Context) (*services_kubernetes.ClusterContext, error) {
	// Create a channel to receive the result
	resultChan := make(chan *services_kubernetes.ClusterContext, 1)
	errorChan := make(chan error, 1)

	// Run the selector in a goroutine
	go func() {
		cluster, err := animation.InteractiveClusterSelector()
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- cluster
	}()

	// Wait for either result or timeout
	select {
	case cluster := <-resultChan:
		return cluster, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
