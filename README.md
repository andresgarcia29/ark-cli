# ark-cli

A powerful command-line interface for AWS and Kubernetes operations, designed to streamline your cloud workflow.

## Features

- **AWS Operations**: Login, SSO, and credential management
- **Kubernetes Integration**: Seamless k8s operations
- **Parallel Processing**: Efficient handling of multiple operations
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Installation

### Using Go
```bash
go install github.com/andresgarcia29/ark-cli@latest
```

### Download Binary
Download the appropriate binary for your platform from the [releases page](https://github.com/andresgarcia29/ark-cli/releases).

## Quick Start

### AWS Operations

#### Login to AWS
```bash
ark aws login --profile my-profile
```

#### AWS SSO
```bash
ark aws sso --start-url https://your-sso-domain.awsapps.com/start
```

### Kubernetes Operations

```bash
ark k8s [command]
```

## Commands

### AWS Commands
- `ark aws login` - Login to AWS with profile
- `ark aws sso` - Configure AWS SSO

### Kubernetes Commands
- `ark k8s` - Kubernetes operations

## Configuration

The tool uses standard AWS and Kubernetes configuration files:
- AWS: `~/.aws/config` and `~/.aws/credentials`
- Kubernetes: `~/.kube/config`

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
go test ./...
```

### Linting
```bash
golangci-lint run
```

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

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a list of changes and version history.
