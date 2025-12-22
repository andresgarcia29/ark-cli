# ark-cli

[![Go Report Card](https://goreportcard.com/badge/github.com/andresgarcia29/ark-cli)](https://goreportcard.com/report/github.com/andresgarcia29/ark-cli)
[![Coverage Status](https://img.shields.io/badge/coverage-85%25-brightgreen)](https://github.com/andresgarcia29/ark-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A powerful command-line interface for AWS and Kubernetes operations, designed to streamline your cloud workflow.

## Features

- **AWS Operations**: Login, SSO, and credential management
- **Kubernetes Integration**: Seamless k8s operations
- **Parallel Processing**: Efficient handling of multiple operations (see [PARALLELIZATION.md](PARALLELIZATION.md))
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Auto-Browser**: Automatically opens browser for AWS SSO authentication

## Installation

### Using brew

```bash
brew update
brew tap andresgarcia29/agm --force
brew upgrade ark --cask
sudo xattr -r -d com.apple.quarantine $(which ark)
```

### Using Go
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
- `--role-prefixs`: (Optional) Comma-separated list of role prefixes to search for (default: `readonly,read-only`).
- `--role-arn`: (Optional) Specific static Role ARN to use. **Mutually exclusive with `--role-prefixs`**.
- `--regions`: (Optional) List of AWS regions to scan (default: `us-west-2`).
- `--clean`: (Optional) Clean `kubeconfig` before configuring (default: `true`).
- `--kubeconfig-path`: (Optional) Path to `kubeconfig` (default: `~/.kube/config`).
- `--replace-profile`: (Optional) Replace profile in `kubeconfig` with a specific one.

#### `ark k8s diagnose`
Diagnoses common issues with your Kubernetes and `kubectl` configuration.

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
