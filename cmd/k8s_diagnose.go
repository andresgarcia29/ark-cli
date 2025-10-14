package cmd

import (
	"fmt"
	"os/exec"

	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	"github.com/spf13/cobra"
)

var (
	kubernetesDiagnoseCmd = &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose Kubernetes and kubectl configuration issues",
		Long:  `Diagnose common issues that can cause the EKS command to hang or fail.`,
		Run:   kubernetesDiagnose,
	}
)

func init() {
	kubernetesCmd.AddCommand(kubernetesDiagnoseCmd)
}

func kubernetesDiagnose(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 Kubernetes Environment Diagnostics")
	fmt.Println("=====================================")

	// Test 1: kubectl availability
	fmt.Println("\n1. Testing kubectl availability...")
	if err := testKubectlCommand(); err != nil {
		fmt.Printf("❌ kubectl command failed: %v\n", err)
		fmt.Println("💡 Solution: Install kubectl or add it to your PATH")
		return
	}
	fmt.Println("✅ kubectl is available")

	// Test 2: kubectl configuration
	fmt.Println("\n2. Testing kubectl configuration...")
	if err := testKubectlConfig(); err != nil {
		fmt.Printf("❌ kubectl configuration issue: %v\n", err)
		fmt.Println("💡 Solution: Run 'kubectl config get-contexts' to check your configuration")
		return
	}
	fmt.Println("✅ kubectl configuration is valid")

	// Test 3: Cluster contexts
	fmt.Println("\n3. Testing cluster contexts...")
	clusters, err := services_kubernetes.GetClusterContexts()
	if err != nil {
		fmt.Printf("❌ Failed to get cluster contexts: %v\n", err)
		fmt.Println("💡 This is likely the cause of the hanging issue")
		fmt.Println("💡 Solution: Check your kubeconfig file and network connectivity")
		return
	}

	if len(clusters) == 0 {
		fmt.Println("⚠️  No cluster contexts found")
		fmt.Println("💡 Solution: Run 'ark k8s setup' to configure EKS clusters")
		return
	}

	fmt.Printf("✅ Found %d cluster contexts\n", len(clusters))
	for i, cluster := range clusters {
		if i < 3 { // Show first 3 clusters
			fmt.Printf("   - %s (current: %v)\n", cluster.Name, cluster.Current)
		}
	}
	if len(clusters) > 3 {
		fmt.Printf("   ... and %d more\n", len(clusters)-3)
	}

	// Test 4: Network connectivity (basic)
	fmt.Println("\n4. Testing basic network connectivity...")
	if err := testNetworkConnectivity(); err != nil {
		fmt.Printf("⚠️  Network connectivity issue: %v\n", err)
		fmt.Println("💡 This might cause timeouts when accessing AWS services")
	} else {
		fmt.Println("✅ Basic network connectivity is working")
	}

	fmt.Println("\n🎉 Diagnostics completed!")
	fmt.Println("If all tests passed but the command still hangs, try:")
	fmt.Println("  - Running with --debug flag")
	fmt.Println("  - Checking AWS credentials: aws sts get-caller-identity")
	fmt.Println("  - Checking network connectivity to AWS services")
}

func testKubectlCommand() error {
	cmd := exec.Command("kubectl", "version", "--client")
	return cmd.Run()
}

func testKubectlConfig() error {
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	return cmd.Run()
}

func testNetworkConnectivity() error {
	// Test basic connectivity to a common AWS endpoint
	cmd := exec.Command("ping", "-c", "1", "-W", "5", "eks.us-west-2.amazonaws.com")
	return cmd.Run()
}
