package services_kubernetes

import (
	"testing"
)

func TestGetClusterContexts(t *testing.T) {
	// This test requires kubectl to be available and configured
	// It's more of an integration test than a unit test
	contexts, err := GetClusterContexts()
	if err != nil {
		// If kubectl is not available or no contexts exist, that's okay for testing
		t.Logf("GetClusterContexts returned error (expected if kubectl not configured): %v", err)
		return
	}

	// If we get here, we have contexts
	t.Logf("Found %d cluster contexts", len(contexts))

	for _, context := range contexts {
		t.Logf("Context: %s (current: %v)", context.Name, context.Current)
	}
}

func TestSwitchToContext(t *testing.T) {
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

	// Try to switch to the first context
	firstContext := contexts[0].Name
	err = SwitchToContext(firstContext)
	if err != nil {
		t.Errorf("Failed to switch to context %s: %v", firstContext, err)
	}

	// Switch back to the original context if it was different
	for _, context := range contexts {
		if context.Current && context.Name != firstContext {
			err = SwitchToContext(context.Name)
			if err != nil {
				t.Logf("Warning: Failed to switch back to original context %s: %v", context.Name, err)
			}
			break
		}
	}
}
