package cmd

import (
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib/animation"
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
	// Mostrar selector interactivo de clusters
	selectedCluster, err := animation.InteractiveClusterSelector()
	if err != nil {
		fmt.Printf("❌ Error selecting cluster: %v\n", err)
		return
	}

	// Mostrar información del cluster seleccionado
	fmt.Printf("\n✅ Selected cluster: %s", selectedCluster.Name)
	if selectedCluster.Current {
		fmt.Printf(" (currently active)")
	}
	fmt.Println()

	// Si el cluster ya está activo, no hacer nada
	if selectedCluster.Current {
		fmt.Println("🎉 This cluster is already active!")
		return
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
