package services_kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Entry is one cluster to write into the kubeconfig. It carries exactly what
// `aws eks update-kubeconfig` would have written, obtained from DescribeCluster.
type Entry struct {
	ContextName string
	ClusterARN  string
	Endpoint    string
	CAData      string
	Region      string
	Profile     string
}

// Kubeconfig is the subset of the kubeconfig schema ark reads and writes.
// Unknown fields survive a round trip because every node is kept as yaml.Node.
type Kubeconfig struct {
	APIVersion     string      `yaml:"apiVersion"`
	Kind           string      `yaml:"kind"`
	CurrentContext string      `yaml:"current-context,omitempty"`
	Preferences    yaml.Node   `yaml:"preferences,omitempty"`
	Clusters       []yaml.Node `yaml:"clusters"`
	Contexts       []yaml.Node `yaml:"contexts"`
	Users          []yaml.Node `yaml:"users"`
}

// named is used to read the name out of an arbitrary kubeconfig list entry so
// existing items can be matched and replaced.
type named struct {
	Name string `yaml:"name"`
}

// LoadKubeconfig reads path, returning an empty config when it is absent.
func LoadKubeconfig(path string) (*Kubeconfig, error) {
	cfg := &Kubeconfig{APIVersion: "v1", Kind: "Config"}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if len(data) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s is not a valid kubeconfig: %w", path, err)
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "v1"
	}
	if cfg.Kind == "" {
		cfg.Kind = "Config"
	}
	return cfg, nil
}

// nodeName reads the "name" field of a kubeconfig list entry.
func nodeName(n yaml.Node) string {
	var v named
	if err := n.Decode(&v); err != nil {
		return ""
	}
	return v.Name
}

// upsert replaces the entry called name, or appends it when absent, so
// re-running setup updates clusters instead of duplicating them.
func upsert(list []yaml.Node, name string, value any) ([]yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", name, err)
	}
	for i, existing := range list {
		if nodeName(existing) == name {
			list[i] = node
			return list, nil
		}
	}
	return append(list, node), nil
}

// Merge adds every entry to the config, replacing entries of the same name and
// leaving unrelated contexts, such as minikube or another cloud, untouched.
func (c *Kubeconfig) Merge(entries []Entry) error {
	for _, e := range entries {
		cluster := map[string]any{
			"name": e.ClusterARN,
			"cluster": map[string]any{
				"server":                     e.Endpoint,
				"certificate-authority-data": e.CAData,
			},
		}

		// The exec block matches what the AWS CLI writes, so kubectl keeps
		// working exactly as before.
		execCfg := map[string]any{
			"apiVersion": "client.authentication.k8s.io/v1beta1",
			"command":    "aws",
			"args": []string{
				"--region", e.Region,
				"eks", "get-token",
				"--cluster-name", clusterNameFromARN(e.ClusterARN),
				"--output", "json",
			},
			"interactiveMode":    "IfAvailable",
			"provideClusterInfo": false,
		}
		if e.Profile != "" {
			execCfg["env"] = []map[string]string{{"name": "AWS_PROFILE", "value": e.Profile}}
		}

		user := map[string]any{"name": e.ClusterARN, "user": map[string]any{"exec": execCfg}}
		contextEntry := map[string]any{
			"name": e.ContextName,
			"context": map[string]any{
				"cluster": e.ClusterARN,
				"user":    e.ClusterARN,
			},
		}

		var err error
		if c.Clusters, err = upsert(c.Clusters, e.ClusterARN, cluster); err != nil {
			return err
		}
		if c.Users, err = upsert(c.Users, e.ClusterARN, user); err != nil {
			return err
		}
		if c.Contexts, err = upsert(c.Contexts, e.ContextName, contextEntry); err != nil {
			return err
		}
	}

	sort.SliceStable(c.Contexts, func(i, j int) bool {
		return nodeName(c.Contexts[i]) < nodeName(c.Contexts[j])
	})
	return nil
}

// clusterNameFromARN pulls the cluster name out of an EKS cluster ARN, which
// ends in "cluster/<name>".
func clusterNameFromARN(arn string) string {
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == '/' {
			return arn[i+1:]
		}
	}
	return arn
}

// Save writes the config to path atomically with owner-only permissions.
func (c *Kubeconfig) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to render kubeconfig: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".kubeconfig.*")
	if err != nil {
		return fmt.Errorf("failed to create temp kubeconfig: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set kubeconfig permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close kubeconfig: %w", err)
	}
	return os.Rename(tmp.Name(), path)
}
