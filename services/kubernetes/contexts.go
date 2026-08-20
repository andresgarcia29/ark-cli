package services_kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ClusterContext is one entry in the user's kubeconfig.
type ClusterContext struct {
	Name        string
	Current     bool
	Profile     string
	Region      string
	ClusterName string
}

// kubectl runs a kubectl subcommand and returns its stdout.
func kubectl(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		if i := strings.IndexByte(detail, '\n'); i >= 0 {
			detail = detail[:i]
		}
		return "", fmt.Errorf("kubectl %s: %s", args[0], detail)
	}
	return stdout.String(), nil
}

// GetClusterContexts lists the contexts in the kubeconfig, marking the active one.
func GetClusterContexts(ctx context.Context) ([]ClusterContext, error) {
	out, err := kubectl(ctx, "config", "get-contexts", "-o", "name")
	if err != nil {
		return nil, err
	}

	current, err := kubectl(ctx, "config", "current-context")
	if err != nil {
		// A kubeconfig with no active context is valid; nothing is marked.
		current = ""
	}
	current = strings.TrimSpace(current)

	var contexts []ClusterContext
	for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		contexts = append(contexts, ClusterContext{Name: name, Current: name == current})
	}
	return contexts, nil
}

// GetKubernetesContextDetails reports the AWS profile, region and cluster name
// backing a context, reading them from the exec credential plugin kubectl
// stores for EKS. Missing values come back empty rather than as an error.
func GetKubernetesContextDetails(ctx context.Context, contextName string) (profile, region, clusterName string, err error) {
	user, err := kubectl(ctx, "config", "view", "-o",
		fmt.Sprintf("jsonpath={.contexts[?(@.name==%q)].context.user}", contextName))
	if err != nil {
		return "", "", "", err
	}
	user = strings.TrimSpace(user)
	if user == "" {
		return "", "", "", nil
	}

	profile, _ = kubectlValue(ctx, fmt.Sprintf(
		"jsonpath={.users[?(@.name==%q)].user.exec.env[?(@.name=='AWS_PROFILE')].value}", user))
	args, _ := kubectlValue(ctx, fmt.Sprintf(
		"jsonpath={.users[?(@.name==%q)].user.exec.args}", user))

	region = flagValue(args, "--region")
	clusterName = flagValue(args, "--cluster-name")
	return profile, region, clusterName, nil
}

func kubectlValue(ctx context.Context, format string) (string, error) {
	out, err := kubectl(ctx, "config", "view", "-o", format)
	return strings.TrimSpace(out), err
}

// flagValue pulls the argument following flag out of kubectl's JSON array
// rendering of the exec plugin args, e.g. ["eks","get-token","--region","us-west-2"].
func flagValue(args, flag string) string {
	fields := strings.FieldsFunc(args, func(r rune) bool {
		return r == '[' || r == ']' || r == ',' || r == '"' || r == ' '
	})
	for i, f := range fields {
		if f == flag && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

// SwitchToContext makes contextName the active kubectl context.
func SwitchToContext(ctx context.Context, contextName string) error {
	_, err := kubectl(ctx, "config", "use-context", contextName)
	return err
}
