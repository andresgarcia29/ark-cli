package services_kubernetes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func sampleEntry(name, region string) Entry {
	return Entry{
		ContextName: name,
		ClusterARN:  "arn:aws:eks:" + region + ":111122223333:cluster/" + name,
		Endpoint:    "https://" + name + ".eks.amazonaws.com",
		CAData:      "LS0tLS1CRUdJTiBDRVJU",
		Region:      region,
		Profile:     "acme-readonly",
	}
}

func TestMergeWritesUsableEntry(t *testing.T) {
	cfg, err := LoadKubeconfig(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Merge([]Entry{sampleEntry("prod", "us-west-2")}); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	for _, want := range []string{
		"https://prod.eks.amazonaws.com",
		"certificate-authority-data: LS0tLS1CRUdJTiBDRVJU",
		"AWS_PROFILE",
		"acme-readonly",
		"--cluster-name",
		"client.authentication.k8s.io/v1beta1",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("kubeconfig missing %q:\n%s", want, data)
		}
	}

	if info, _ := os.Stat(path); info.Mode().Perm() != 0600 {
		t.Errorf("permissions = %o, want 600", info.Mode().Perm())
	}
}

// kubectl is the real judge of whether the generated file is valid.
func TestGeneratedKubeconfigIsValid(t *testing.T) {
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not installed")
	}

	cfg, _ := LoadKubeconfig(filepath.Join(t.TempDir(), "absent"))
	if err := cfg.Merge([]Entry{sampleEntry("prod", "us-west-2"), sampleEntry("staging", "eu-west-1")}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kubectl rejected the generated kubeconfig: %v\n%s", err, out)
	}

	got := strings.Fields(string(out))
	if len(got) != 2 {
		t.Errorf("kubectl saw %v, want prod and staging", got)
	}

	// The exec block must survive kubectl's own parsing.
	cmd = exec.Command("kubectl", "config", "view", "-o",
		`jsonpath={.users[0].user.exec.env[?(@.name=='AWS_PROFILE')].value}`)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+path)
	out, _ = cmd.Output()
	if strings.TrimSpace(string(out)) != "acme-readonly" {
		t.Errorf("AWS_PROFILE round trip = %q", out)
	}
}

// Re-running setup must update clusters in place, not pile up duplicates.
func TestMergeIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")

	for i := 0; i < 3; i++ {
		cfg, err := LoadKubeconfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := cfg.Merge([]Entry{sampleEntry("prod", "us-west-2")}); err != nil {
			t.Fatal(err)
		}
		if err := cfg.Save(path); err != nil {
			t.Fatal(err)
		}
	}

	cfg, _ := LoadKubeconfig(path)
	if len(cfg.Contexts) != 1 || len(cfg.Clusters) != 1 || len(cfg.Users) != 1 {
		t.Errorf("after 3 runs: %d contexts, %d clusters, %d users; want 1 each",
			len(cfg.Contexts), len(cfg.Clusters), len(cfg.Users))
	}
}

// Contexts the user already had must survive a merge.
func TestMergePreservesForeignContexts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	existing := `apiVersion: v1
kind: Config
current-context: minikube
clusters:
- cluster: {server: https://127.0.0.1:8443}
  name: minikube
contexts:
- context: {cluster: minikube, user: minikube}
  name: minikube
users:
- name: minikube
  user: {token: abc}
`
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadKubeconfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Merge([]Entry{sampleEntry("prod", "us-west-2")}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	for _, want := range []string{"minikube", "127.0.0.1:8443", "prod"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("merge lost %q:\n%s", want, data)
		}
	}
	if !strings.Contains(string(data), "current-context: minikube") {
		t.Error("merge dropped the active context")
	}
}

func TestLoadKubeconfigRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("\tnot: [valid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKubeconfig(path); err == nil {
		t.Error("a malformed kubeconfig should be reported, not silently replaced")
	}
}

func TestClusterNameFromARN(t *testing.T) {
	cases := map[string]string{
		"arn:aws:eks:us-west-2:1:cluster/prod": "prod",
		"plain-name":                           "plain-name",
	}
	for in, want := range cases {
		if got := clusterNameFromARN(in); got != want {
			t.Errorf("clusterNameFromARN(%q) = %q, want %q", in, got, want)
		}
	}
}
