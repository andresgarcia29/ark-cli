package controllers

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireKubectl(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not installed")
	}
}

func kubeconfigFixture(cluster, ctxName string) string {
	return `apiVersion: v1
kind: Config
clusters:
- cluster: {server: https://` + cluster + `.example.com}
  name: ` + cluster + `
contexts:
- context: {cluster: ` + cluster + `, user: ` + cluster + `}
  name: ` + ctxName + `
users:
- name: ` + cluster + `
  user: {token: t}
`
}

// The merge must fold new clusters in without discarding contexts the user
// already had, such as minikube or another cloud.
func TestMergeKubeconfigsPreservesExistingContexts(t *testing.T) {
	requireKubectl(t)
	dir := t.TempDir()

	target := filepath.Join(dir, "config")
	if err := os.WriteFile(target, []byte(kubeconfigFixture("minikube", "minikube")), 0600); err != nil {
		t.Fatal(err)
	}
	incoming := filepath.Join(dir, "incoming.yaml")
	if err := os.WriteFile(incoming, []byte(kubeconfigFixture("eks-prod", "eks-prod")), 0600); err != nil {
		t.Fatal(err)
	}

	if err := mergeKubeconfigs(context.Background(), []string{incoming}, target); err != nil {
		t.Fatalf("mergeKubeconfigs: %v", err)
	}

	merged, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"minikube", "eks-prod"} {
		if !strings.Contains(string(merged), want) {
			t.Errorf("merged kubeconfig lost %q:\n%s", want, merged)
		}
	}

	info, _ := os.Stat(target)
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("kubeconfig permissions = %o, want 600", perm)
	}
}

func TestMergeKubeconfigsCreatesMissingTarget(t *testing.T) {
	requireKubectl(t)
	dir := t.TempDir()

	incoming := filepath.Join(dir, "incoming.yaml")
	if err := os.WriteFile(incoming, []byte(kubeconfigFixture("eks-prod", "eks-prod")), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "nested", "config")

	if err := mergeKubeconfigs(context.Background(), []string{incoming}, target); err != nil {
		t.Fatalf("mergeKubeconfigs: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || !strings.Contains(string(data), "eks-prod") {
		t.Errorf("target not written: %v", err)
	}
}

func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"boom\ntrace\nmore": "boom",
		"  single  ":        "single",
		"":                  "aws eks update-kubeconfig failed",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitise(t *testing.T) {
	if got := sanitise("1234/us-west-2/prod cluster"); strings.ContainsAny(got, "/ ") {
		t.Errorf("sanitise left a path separator: %q", got)
	}
}
