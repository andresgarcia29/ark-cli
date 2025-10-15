package services_kubernetes

import (
	"testing"
)

func TestGetClusterContextsIntegration(t *testing.T) {
	// This is an integration test that requires kubectl to be available
	contexts, err := GetClusterContexts()
	if err != nil {
		t.Logf("GetClusterContexts returned error (expected if kubectl not configured): %v", err)
		return
	}

	t.Logf("Found %d cluster contexts", len(contexts))

	for _, context := range contexts {
		t.Logf("Context: %s (current: %v, profile: %s, region: %s, cluster: %s)",
			context.Name, context.Current, context.Profile, context.Region, context.ClusterName)
	}
}

func TestGetContextDetailsIntegration(t *testing.T) {
	// First get available contexts
	contexts, err := GetClusterContexts()
	if err != nil {
		t.Skipf("Skipping test - no kubectl contexts available: %v", err)
		return
	}

	if len(contexts) == 0 {
		t.Skip("Skipping test - no contexts found")
		return
	}

	// Test context details extraction for the first context
	firstContext := contexts[0].Name
	profile, region, clusterName, err := GetKubernetesContextDetails(firstContext)
	if err != nil {
		t.Errorf("Failed to get context details for %s: %v", firstContext, err)
		return
	}

	t.Logf("Context details for %s:", firstContext)
	t.Logf("  Profile: %s", profile)
	t.Logf("  Region: %s", region)
	t.Logf("  Cluster Name: %s", clusterName)
}
