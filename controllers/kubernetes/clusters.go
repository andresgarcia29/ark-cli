package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/andresgarcia29/ark-cli/lib"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
)

// ClusterReport summarises a kubeconfig update run.
type ClusterReport struct {
	Configured []string
	Failed     []error
}

// writeClusterEntry runs `aws eks update-kubeconfig` for one cluster into the
// given kubeconfig file. Each call owns its file, so calls can run in parallel.
func writeClusterEntry(ctx context.Context, cluster services_aws.EKSCluster, kubeconfig, replaceProfile string) error {
	profile := cluster.Profile
	if replaceProfile != "" {
		profile = replaceProfile
	}

	cmd := exec.CommandContext(ctx, "aws", "eks", "update-kubeconfig",
		"--name", cluster.Name,
		"--region", cluster.Region,
		"--profile", profile,
		"--alias", cluster.Name,
		"--kubeconfig", kubeconfig,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", cluster.Name, firstLine(stderr.String()))
	}
	return nil
}

// firstLine trims a subprocess error down to something a user can read.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "aws eks update-kubeconfig failed"
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

// ConfigureClusters writes every cluster into kubeconfigPath. Clusters are
// resolved concurrently into private temp files and then merged in one pass,
// because `aws eks update-kubeconfig` rewrites the whole file it is given.
func ConfigureClusters(ctx context.Context, clusters []services_aws.EKSCluster, kubeconfigPath, replaceProfile string, note func(string, ...any)) (ClusterReport, error) {
	var report ClusterReport
	if len(clusters) == 0 {
		return report, nil
	}

	target, err := services_kubernetes.ExpandPath(kubeconfigPath)
	if err != nil {
		return report, err
	}

	work, err := os.MkdirTemp("", "ark-kubeconfig-")
	if err != nil {
		return report, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(work)

	names := make([]string, len(clusters))
	byName := make(map[string]services_aws.EKSCluster, len(clusters))
	for i, c := range clusters {
		key := fmt.Sprintf("%s/%s/%s", c.AccountID, c.Region, c.Name)
		names[i] = key
		byName[key] = c
	}

	var completed atomic.Int64
	fragments, errs := lib.MapConcurrent(ctx, names, lib.DefaultLimits(),
		func(ctx context.Context, key string) (string, error) {
			path := filepath.Join(work, sanitise(key)+".yaml")
			if err := writeClusterEntry(ctx, byName[key], path, replaceProfile); err != nil {
				return "", lib.Permanent(err)
			}
			if note != nil {
				note("configured %d/%d", completed.Add(1), len(names))
			}
			return path, nil
		})

	report.Failed = errs

	var merge []string
	for _, key := range names {
		if path, ok := fragments[key]; ok {
			merge = append(merge, path)
			report.Configured = append(report.Configured, byName[key].Name)
		}
	}
	sort.Strings(report.Configured)

	if len(merge) == 0 {
		return report, fmt.Errorf("no cluster could be configured: %w", errors.Join(errs...))
	}

	if err := mergeKubeconfigs(ctx, merge, target); err != nil {
		return report, err
	}
	return report, nil
}

// sanitise turns a cluster key into a safe file name.
func sanitise(key string) string {
	return strings.NewReplacer("/", "_", " ", "_", ":", "_").Replace(key)
}

// mergeKubeconfigs flattens sources, plus whatever target already holds, into
// target. kubectl performs the merge so the output stays a valid kubeconfig.
func mergeKubeconfigs(ctx context.Context, sources []string, target string) error {
	inputs := sources
	if _, err := os.Stat(target); err == nil {
		if data, err := os.ReadFile(target); err == nil && len(bytes.TrimSpace(data)) > 0 {
			inputs = append([]string{target}, sources...)
		}
	}

	cmd := exec.CommandContext(ctx, "kubectl", "config", "view", "--flatten")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+strings.Join(inputs, string(os.PathListSeparator)))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to merge kubeconfig: %s", firstLine(stderr.String()))
	}

	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return fmt.Errorf("failed to create %s: %w", filepath.Dir(target), err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".kubeconfig.*")
	if err != nil {
		return fmt.Errorf("failed to create temp kubeconfig: %w", err)
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to set kubeconfig permissions: %w", err)
	}
	if _, err := tmp.Write(stdout.Bytes()); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close kubeconfig: %w", err)
	}
	return os.Rename(tmp.Name(), target)
}
