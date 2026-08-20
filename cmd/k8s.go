package cmd

import (
	"context"
	"time"

	controllers "github.com/andresgarcia29/ark-cli/controllers/aws"
	"github.com/andresgarcia29/ark-cli/lib/animation"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	"github.com/spf13/cobra"
)

var kubernetesCmd = &cobra.Command{
	Use:     "kubernetes",
	Aliases: []string{"k8s", "eks"},
	Short:   "Kubernetes cluster access",
	Long:    "Switch between the EKS clusters in your kubeconfig, refreshing AWS credentials as needed.",
	RunE:    runKubernetes,
}

func init() {
	rootCmd.AddCommand(kubernetesCmd)
}

func runKubernetes(cmd *cobra.Command, args []string) error {
	// Only the kubectl calls get a deadline. The picker waits on a human and
	// must never time out.
	loadCtx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	clusters, err := services_kubernetes.GetClusterContexts(loadCtx)
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		ui.Fail("No clusters in your kubeconfig")
		ui.Hint("ark k8s setup")
		return errQuiet
	}

	cluster, err := animation.SelectCluster(clusters)
	if err != nil {
		return err
	}

	profile, _, _, err := services_kubernetes.GetKubernetesContextDetails(cmd.Context(), cluster.Name)
	if err != nil {
		return err
	}

	if profile != "" {
		if err := controllers.Login(cmd.Context(), profile, true); err != nil {
			return err
		}
	}

	if cluster.Current {
		ui.Done("Already on %s", ui.Strong.Render(cluster.Name))
		return nil
	}

	if err := services_kubernetes.SwitchToContext(cmd.Context(), cluster.Name); err != nil {
		return err
	}

	ui.Done("Switched to %s", ui.Strong.Render(cluster.Name))
	return nil
}
