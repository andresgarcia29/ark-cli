# Testing Guide for Ark CLI

This document provides a comprehensive guide for testing the Ark CLI application.

## Test Structure

The test suite is organized into the following categories:

### 1. Command Tests (`cmd/` package)
- **`cmd/root_test.go`** - Tests for the root command and global flags
- **`cmd/aws_test.go`** - Tests for AWS command functionality
- **`cmd/aws_login_test.go`** - Tests for AWS login command
- **`cmd/aws_sso_test.go`** - Tests for AWS SSO command
- **`cmd/k8s_test.go`** - Tests for Kubernetes command
- **`cmd/version_test.go`** - Tests for version command

### 2. Controller Tests (`controllers/` package)
- **`controllers/aws/login_test.go`** - Tests for AWS login controller
- **`controllers/aws/sso_test.go`** - Tests for AWS SSO controller
- **`controllers/kubernetes/clusters_test.go`** - Tests for Kubernetes cluster controller

### 3. Service Tests (`services/` package)
- **`services/aws/core_test.go`** - Tests for AWS core functionality
- **`services/aws/sso_test.go`** - Tests for AWS SSO service
- **`services/aws/accounts_test.go`** - Tests for AWS accounts service
- **`services/aws/roles_test.go`** - Tests for AWS roles service
- **`services/aws/credentials_test.go`** - Tests for AWS credentials service
- **`services/aws/profiles_test.go`** - Tests for AWS profiles service
- **`services/aws/eks_test.go`** - Tests for AWS EKS service
- **`services/kubernetes/config_test.go`** - Tests for Kubernetes config service

### 4. Library Tests (`lib/` package)
- **`lib/parallel_test.go`** - Tests for parallel execution utilities
- **`lib/animation/progress_test.go`** - Tests for progress bar animation
- **`lib/animation/selector_test.go`** - Tests for profile selector animation
- **`lib/animation/spinner_test.go`** - Tests for spinner animation

### 5. Logging Tests (`logs/` package)
- **`logs/logs_test.go`** - Tests for logging functionality

## Test Utilities

### Test Helpers (`test_helpers.go`)
The test helpers provide common utilities for testing:

- **MockAWSClient** - Mock implementation of AWS client interfaces
- **MockSSOClient** - Mock implementation of SSO client
- **MockEKSClient** - Mock implementation of EKS client
- **MockLogger** - Mock implementation of logger
- **TestHelper** - Common test utilities
- **TestData** - Common test data structures
- **TestError** - Common test errors
- **TestConfig** - Test configuration
- **TestFileSystem** - Test file system utilities
- **TestNetwork** - Test network utilities
- **TestTimer** - Test timer utilities
- **TestAssertions** - Additional test assertions
- **TestCleanup** - Test cleanup utilities
- **TestMetrics** - Test metrics utilities
- **TestConcurrency** - Test concurrency utilities

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Run Tests with Verbose Output
```bash
go test -v ./...
```

### Run Specific Test Package
```bash
go test ./cmd
go test ./controllers
go test ./services
go test ./lib
go test ./logs
```

### Run Specific Test Function
```bash
go test -run TestFunctionName ./package
```

### Run Tests with Race Detection
```bash
go test -race ./...
```

### Run Tests with Benchmark
```bash
go test -bench=. ./...
```

## Test Categories

### Unit Tests
Unit tests focus on testing individual functions and methods in isolation:

- **Function Logic** - Test the core logic of functions
- **Parameter Validation** - Test input validation
- **Error Handling** - Test error scenarios
- **Edge Cases** - Test boundary conditions
- **Return Values** - Test expected return values

### Integration Tests
Integration tests focus on testing the interaction between components:

- **Command Execution** - Test command line interface
- **Service Integration** - Test service layer interactions
- **External Dependencies** - Test integration with AWS services
- **Data Flow** - Test data flow between components

### Mock Tests
Mock tests use mock objects to isolate components:

- **AWS SDK Mocks** - Mock AWS service calls
- **File System Mocks** - Mock file operations
- **Network Mocks** - Mock network calls
- **Logger Mocks** - Mock logging operations

## Test Data

### Common Test Data
The test suite uses consistent test data:

- **Account ID**: `123456789012`
- **Region**: `us-west-2`
- **Start URL**: `https://example.awsapps.com/start`
- **Role ARN**: `arn:aws:iam::123456789012:role/TestRole`
- **Profile Name**: `test-profile`
- **Cluster Name**: `test-cluster`

### Test Configuration
Default test configuration:

- **Timeout**: 5 seconds
- **Retry Count**: 3
- **Retry Delay**: 100ms
- **Max Workers**: 5
- **Rate Limit Delay**: 50ms

## Test Patterns

### Table-Driven Tests
Most tests use table-driven patterns for comprehensive coverage:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expected    string
        expectedErr bool
    }{
        {
            name:        "valid input",
            input:       "test",
            expected:    "result",
            expectedErr: false,
        },
        {
            name:        "invalid input",
            input:       "",
            expected:    "",
            expectedErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.expectedErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Context Testing
Tests that involve context handling:

```go
func TestWithContext(t *testing.T) {
    ctx := context.Background()
    
    // Test with timeout
    timeoutCtx, cancel := context.WithTimeout(ctx, 0)
    defer cancel()
    
    // Test with cancellation
    cancelCtx, cancel := context.WithCancel(ctx)
    cancel()
}
```

### Error Testing
Tests for error scenarios:

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        errorType   string
        expectedMsg string
    }{
        {
            name:        "validation error",
            errorType:   "validation",
            expectedMsg: "validation failed",
        },
    }
}
```

## Best Practices

### Test Organization
- Group related tests together
- Use descriptive test names
- Follow the Arrange-Act-Assert pattern
- Keep tests focused and simple

### Test Data Management
- Use consistent test data
- Avoid hardcoded values
- Use test fixtures for complex data
- Clean up test data after tests

### Error Testing
- Test both success and failure scenarios
- Test edge cases and boundary conditions
- Verify error messages are meaningful
- Test error recovery mechanisms

### Performance Testing
- Use benchmarks for performance-critical code
- Test with realistic data sizes
- Monitor memory usage
- Test concurrent operations

### Mock Usage
- Mock external dependencies
- Use mocks to isolate components
- Verify mock interactions
- Keep mocks simple and focused

## Continuous Integration

### GitHub Actions
The test suite is designed to run in CI/CD pipelines:

- **Go Version**: 1.21+
- **Test Command**: `go test ./...`
- **Coverage**: `go test -cover ./...`
- **Race Detection**: `go test -race ./...`

### Test Coverage
Target test coverage:

- **Overall**: 80%+
- **Critical Paths**: 90%+
- **Error Handling**: 100%
- **Public APIs**: 100%

## Troubleshooting

### Common Issues

1. **Import Errors**: Ensure all dependencies are installed
2. **Mock Failures**: Check mock setup and expectations
3. **Race Conditions**: Use race detection to identify issues
4. **Timeout Issues**: Adjust test timeouts for slow operations

### Debug Tips

1. **Verbose Output**: Use `-v` flag for detailed test output
2. **Single Test**: Run individual tests for debugging
3. **Logging**: Add temporary logging for debugging
4. **Breakpoints**: Use debugger for complex issues

## Contributing

### Adding New Tests
When adding new functionality:

1. **Write Tests First**: Follow TDD principles
2. **Cover Edge Cases**: Test boundary conditions
3. **Test Error Scenarios**: Ensure proper error handling
4. **Update Documentation**: Keep this guide updated

### Test Review
When reviewing tests:

1. **Coverage**: Ensure adequate test coverage
2. **Clarity**: Tests should be easy to understand
3. **Maintainability**: Tests should be easy to maintain
4. **Performance**: Tests should run efficiently

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Mock Documentation](https://github.com/stretchr/testify#mock-package)
- [AWS SDK Testing](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/testing.html)
