package controllers

import (
	"context"
	"fmt"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
)

// ClusterReport summarises a kubeconfig update run.
type ClusterReport struct {
	Configured []string
	Skipped    []error
}

// ConfigureClusters writes every cluster into kubeconfigPath in a single pass.
//
// It deliberately does not shell out to `aws eks update-kubeconfig`: that spawns
// a Python process and re-resolves credentials per cluster, which is what made
// large accounts take tens of minutes. Everything it would have written already
// came back from DescribeCluster during the scan.
func ConfigureClusters(ctx context.Context, clusters []services_aws.EKSCluster, kubeconfigPath, replaceProfile string) (ClusterReport, error) {
	var report ClusterReport
	if len(clusters) == 0 {
		return report, nil
	}

	path, err := services_kubernetes.ExpandPath(kubeconfigPath)
	if err != nil {
		return report, err
	}

	entries := make([]services_kubernetes.Entry, 0, len(clusters))
	for _, c := range clusters {
		if !c.Complete() {
			report.Skipped = append(report.Skipped,
				fmt.Errorf("%s: could not read its endpoint or certificate", c.Name))
			continue
		}

		profile := c.Profile
		if replaceProfile != "" {
			profile = replaceProfile
		}

		entries = append(entries, services_kubernetes.Entry{
			ContextName: c.Name,
			ClusterARN:  c.ARN,
			Endpoint:    c.Endpoint,
			CAData:      c.CertificateAuthority,
			Region:      c.Region,
			Profile:     profile,
		})
		report.Configured = append(report.Configured, c.Name)
	}

	if len(entries) == 0 {
		return report, fmt.Errorf("no cluster had usable connection details")
	}

	cfg, err := services_kubernetes.LoadKubeconfig(path)
	if err != nil {
		return report, err
	}
	if err := cfg.Merge(entries); err != nil {
		return report, err
	}
	if err := cfg.Save(path); err != nil {
		return report, err
	}
	return report, nil
}
