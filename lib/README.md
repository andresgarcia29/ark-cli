# Lib Package

This package contains common libraries and utilities that can be used across the entire project.

## Parallel Processing (`parallel.go`)

A comprehensive parallelization library for managing concurrent operations with built-in support for:
- Worker pools for controlling concurrency
- Rate limiting to prevent API overload
- Automatic retry logic with exponential backoff
- Timeout management
- Generic type support for flexible data processing

### Components

#### ParallelConfig

Configuration structure for controlling parallel execution behavior:

```go
type ParallelConfig struct {
    MaxWorkers     int           // Maximum concurrent goroutines
    Timeout        time.Duration // Maximum execution time for all operations
    RateLimitDelay time.Duration // Delay between starting new tasks
    MaxRetries     int           // Number of retry attempts for failed operations
    RetryDelay     time.Duration // Delay between retry attempts
}
```

**Preset Configurations:**

- `DefaultParallelConfig()`: Balanced configuration (10 workers, 5min timeout, 100ms rate limit, 3 retries)
- `ConservativeConfig()`: Conservative settings (5 workers, 10min timeout, 500ms rate limit, 5 retries)
- `AggressiveConfig()`: Maximum performance (20 workers, 3min timeout, 50ms rate limit, 2 retries)

#### WorkerPool

Controls the number of concurrent goroutines to prevent system overload:

```go
pool := lib.NewWorkerPool(10) // Max 10 concurrent workers
err := pool.Execute(ctx, func() error {
    // Your task here
    return nil
})
```

#### RateLimiter

Prevents overwhelming APIs with too many requests:

```go
limiter := lib.NewRateLimiter(100 * time.Millisecond)
err := limiter.Wait(ctx) // Enforces minimum delay between operations
```

#### ProcessAccountsInParallel

Generic function for processing multiple accounts in parallel with full type safety:

```go
results, errors := lib.ProcessAccountsInParallel[[]EKSCluster](
    ctx,
    accountIDs,
    config,
    func(ctx context.Context, accountID string) ([]EKSCluster, error) {
        // Process single account
        return getClusters(ctx, accountID)
    },
)
```

**Features:**
- Generic type parameter for flexible return types
- Automatic worker pool management
- Built-in rate limiting
- Retry logic with exponential backoff
- Timeout enforcement
- Progress reporting
- Graceful error handling

#### ExecuteWithRetry

Executes operations with automatic retry logic:

```go
err := lib.ExecuteWithRetry(ctx, config, func() error {
    // Operation that might fail temporarily
    return riskyOperation()
})
```

### Usage Examples

#### Example 1: Processing Multiple AWS Accounts

```go
import "github.com/andresgarcia29/ark-cli/lib"

config := lib.DefaultParallelConfig()

results, errors := lib.ProcessAccountsInParallel[[]EKSCluster](
    ctx,
    accountIDs,
    config,
    func(ctx context.Context, accountID string) ([]EKSCluster, error) {
        return getEKSClusters(ctx, accountID)
    },
)

// Handle results
for accountID, clusters := range results {
    fmt.Printf("Account %s: %d clusters\n", accountID, len(clusters))
}

// Handle errors
for _, err := range errors {
    fmt.Printf("Error: %v\n", err)
}
```

#### Example 2: Custom Worker Pool

```go
import "github.com/andresgarcia29/ark-cli/lib"

pool := lib.NewWorkerPool(5) // Max 5 concurrent operations

for _, item := range items {
    err := pool.Execute(ctx, func() error {
        return processItem(item)
    })
    if err != nil {
        log.Printf("Error processing item: %v", err)
    }
}
```

#### Example 3: Rate Limited Operations

```go
import "github.com/andresgarcia29/ark-cli/lib"

limiter := lib.NewRateLimiter(200 * time.Millisecond)

for _, apiCall := range apiCalls {
    if err := limiter.Wait(ctx); err != nil {
        return err
    }

    response, err := makeAPICall(apiCall)
    if err != nil {
        log.Printf("API call failed: %v", err)
    }
}
```

### Migration from services/aws

The parallel processing functionality was previously located in `services/aws/parallel.go`. It has been moved to `lib/parallel.go` to make it available across the entire project.

**Update your imports:**

```go
// Old
import services_aws "github.com/andresgarcia29/ark-cli/services/aws"
config := services_aws.DefaultParallelConfig()

// New
import "github.com/andresgarcia29/ark-cli/lib"
config := lib.DefaultParallelConfig()
```

**AWS-specific functions** (like `ProcessRegionsInParallel`) are now in `services/aws/parallel_helpers.go`.

### Best Practices

1. **Choose the right config**: Use `DefaultParallelConfig()` for most cases, `ConservativeConfig()` for production systems, and `AggressiveConfig()` only for batch operations.

2. **Handle context cancellation**: Always pass and respect context for proper timeout and cancellation handling.

3. **Monitor errors**: The parallel functions return both results and errors - handle both appropriately.

4. **Adjust worker count**: Set `MaxWorkers` based on:
   - API rate limits
   - System resources (CPU, memory)
   - Network bandwidth

5. **Set appropriate timeouts**: Configure `Timeout` to be longer than the longest expected operation, but short enough to catch hanging operations.

6. **Rate limiting**: Use `RateLimitDelay` to prevent overwhelming external APIs and services.

### Thread Safety

All components in this package are thread-safe and can be used from multiple goroutines concurrently:
- `WorkerPool` uses semaphore channels for safe concurrency control
- `RateLimiter` uses mutex locks for thread-safe time tracking
- `ProcessAccountsInParallel` uses WaitGroups and channels for safe goroutine coordination
