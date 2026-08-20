# ark-cli

[![Go Report Card](https://goreportcard.com/badge/github.com/andresgarcia29/ark-cli)](https://goreportcard.com/report/github.com/andresgarcia29/ark-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A powerful command-line interface for AWS and Kubernetes operations, designed to streamline your cloud workflow.

## Features

- **AWS Operations**: Login, SSO, and credential management
- **Kubernetes Integration**: Seamless k8s operations
- **Parallel Processing**: Accounts and regions are scanned concurrently
- **Native kubeconfig**: Entries are written directly by ark, not by spawning `aws eks update-kubeconfig` per cluster
- **Throttle aware**: AWS rate limiting is retried with adaptive backoff and reported instead of looking like a hang
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Auto-Browser**: Automatically opens browser for AWS SSO authentication

## Installation

### Using brew [Install]

```bash
brew update
brew tap andresgarcia29/agm --force
brew install ark --cask
sudo xattr -r -d com.apple.quarantine $(which ark)
```

### Using brew [Upgrade]

```bash
brew update
brew tap andresgarcia29/agm --force
brew upgrade ark --cask
sudo xattr -r -d com.apple.quarantine $(which ark)
```

### Using Go [In Progress]
```bash
go install github.com/andresgarcia29/ark-cli@latest
```

---

## Detailed Command Guide

### ☁️ AWS Commands

#### `ark aws`
Interactive profile selector. Shows all configured profiles in your `~/.aws/config` and lets you pick one to log in.

#### `ark aws login`
Logs into AWS using a specific profile.
- `--profile`: (Required) Name of the profile to use.
- `--set-default`: (Optional) Set this profile as the `[default]` in your credentials file.

#### `ark aws sso`
Configures and starts a new AWS SSO session.
- `--start-url`: (Required) AWS SSO start URL.
- `--region`: (Optional) AWS SSO region (default: `us-east-1`).

### ☸️ Kubernetes Commands

#### `ark k8s`
Interactive cluster selector. Lists all clusters in your `kubeconfig` and lets you switch between them. It will automatically check if you need to assume a role for the selected cluster.

#### `ark k8s setup`
Scans AWS accounts for EKS clusters and configures them in your `kubeconfig`.

Cluster details are read once via `eks:DescribeCluster` during the scan and the
kubeconfig is written in a single pass, so hundreds of clusters take seconds
rather than minutes. Re-running updates entries in place. The AWS CLI is still
needed at *connect* time, because kubectl calls `aws eks get-token`.
- `--role-prefixes`: (Optional) Comma-separated list of role prefixes to search for (default: `readonly,read-only`).
- `--role-arn`: (Optional) Specific static Role ARN to use. **Mutually exclusive with `--role-prefixes`**.
- `--regions`: (Optional) List of AWS regions to scan (default: `us-west-2`).
- `--clean`: (Optional) Replace `kubeconfig` instead of merging into it (default: `false`). The previous file is kept as a timestamped backup.
- `--kubeconfig-path`: (Optional) Path to `kubeconfig` (default: `~/.kube/config`).
- `--replace-profile`: (Optional) Replace profile in `kubeconfig` with a specific one.

#### `ark k8s diagnose`
Diagnoses common issues with your Kubernetes and `kubectl` configuration.

### Terminal Output

The interface adapts to where it is running: colours follow a palette that
adjusts to light and dark terminals, and `NO_COLOR`, dumb terminals and
redirected output automatically degrade to plain ASCII with no escape codes.
Interactive pickers and spinners are skipped entirely when stderr is not a
terminal, so CI logs stay readable.

### Global Flags

- `--debug`, `-d`: Show internal diagnostics on stderr.
- `--quiet`, `-q`: Only print results and failures.

Results go to stdout and progress to stderr, so `ark k8s setup -q` can be piped.
Commands exit non-zero on failure, and `130` when you cancel.

### ℹ️ General Commands

#### `ark version`
Shows the current version of the CLI.

---

## Development

### Prerequisites
- Go 1.21 or later
- AWS CLI (for AWS operations)
- kubectl (for Kubernetes operations)

### Building from Source
```bash
git clone https://github.com/andresgarcia29/ark-cli.git
cd ark-cli
go build -o ark main.go
```

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage report
make coverage
```

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run tests and linting
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- [Issues](https://github.com/andresgarcia29/ark-cli/issues)
- [Discussions](https://github.com/andresgarcia29/ark-cli/discussions)
