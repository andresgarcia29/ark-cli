package animation

import (
	"fmt"
	"sort"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
)

// SelectProfile lets the user pick one of the supplied AWS profiles.
func SelectProfile(profiles []services_aws.ProfileConfig) (*services_aws.ProfileConfig, error) {
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].ProfileName < profiles[j].ProfileName })

	items := make([]Item, len(profiles))
	for i, p := range profiles {
		detail := fmt.Sprintf("account %s", p.AccountID)
		if p.RoleName != "" {
			detail += " · " + p.RoleName
		}
		if p.ProfileType == services_aws.ProfileTypeAssumeRole && p.SourceProfile != "" {
			detail += " · via " + p.SourceProfile
		}
		items[i] = Item{Title: p.ProfileName, Badge: string(p.ProfileType), Detail: detail}
	}

	idx, err := Select("Select an AWS profile", items)
	if err != nil {
		return nil, err
	}
	return &profiles[idx], nil
}

// SelectCluster lets the user pick one of the supplied kube contexts.
func SelectCluster(clusters []services_kubernetes.ClusterContext) (*services_kubernetes.ClusterContext, error) {
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].Name < clusters[j].Name })

	items := make([]Item, len(clusters))
	for i, c := range clusters {
		items[i] = Item{Title: c.Name, Detail: c.Region, Marked: c.Current}
	}

	idx, err := Select("Select a Kubernetes cluster", items)
	if err != nil {
		return nil, err
	}
	return &clusters[idx], nil
}
