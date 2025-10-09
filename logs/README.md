# Logging Package

This package provides structured logging using Zap with configurable log levels and output formats.

## Features

- **Structured logging** with key-value pairs
- **Configurable log levels** (debug, info, warn, error)
- **Multiple output formats** (console with colors, JSON)
- **Dynamic log level changes** at runtime
- **Thread-safe** singleton logger instance
- **Trace ID generation** for request tracking

## Quick Start

### Basic Usage

```go
import "github.com/andresgarcia29/ark-cli/logs"

func main() {
    // Get logger (auto-initializes with default config if not already initialized)
    logger := logs.GetLogger()

    // Log messages
    logger.Info("Application started")
    logger.Infof("Processing %d items", 10)

    // Structured logging with key-value pairs
    logger.Infow("User logged in",
        "user_id", "12345",
        "ip_address", "192.168.1.1")

    // Don't forget to sync before exiting
    defer logs.Sync()
}
```

## Configuration

### Initialize with Custom Config

```go
import "github.com/andresgarcia29/ark-cli/logs"

func main() {
    // Initialize logger with custom configuration
    config := logs.LogConfig{
        Level:      "debug",  // debug, info, warn, error
        Format:     "console", // console or json
        OutputPath: "stdout",  // stdout, stderr, or file path
    }

    if err := logs.InitLogger(config); err != nil {
        panic(err)
    }

    logger := logs.GetLogger()
    logger.Debug("This debug message will be shown")

    defer logs.Sync()
}
```

### Configuration Options

#### Log Levels

- `debug`: Shows all log messages (most verbose)
- `info`: Shows info, warn, and error messages (default)
- `warn`: Shows warnings and errors only
- `error`: Shows errors only (least verbose)

#### Output Formats

- `console`: Human-readable format with colors (default, great for development)
- `json`: Structured JSON format (ideal for production and log aggregation)

#### Output Paths

- `stdout`: Standard output (default)
- `stderr`: Standard error
- `"/path/to/file.log"`: Write to a file

### Examples

#### Production Configuration (JSON to File)

```go
config := logs.LogConfig{
    Level:      "info",
    Format:     "json",
    OutputPath: "/var/log/x-cli/app.log",
}
logs.InitLogger(config)
```

#### Development Configuration (Console with Debug)

```go
config := logs.LogConfig{
    Level:      "debug",
    Format:     "console",
    OutputPath: "stdout",
}
logs.InitLogger(config)
```

## Dynamic Log Level Changes

You can change the log level at runtime:

```go
logger := logs.GetLogger()

// Initially set to info
logger.Info("This will show")
logger.Debug("This won't show")

// Change to debug level
logs.SetLogLevel("debug")

// Now debug messages will show
logger.Debug("This will now show")
```

## Logging Methods

### Simple Logging

```go
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")
```

### Formatted Logging

```go
logger.Debugf("Processing item %d of %d", 5, 10)
logger.Infof("User %s logged in", username)
logger.Warnf("Rate limit exceeded: %d requests", count)
logger.Errorf("Failed to connect to %s: %v", host, err)
```

### Structured Logging (Recommended)

```go
logger.Debugw("Query executed",
    "query", sqlQuery,
    "duration_ms", duration)

logger.Infow("Request completed",
    "method", "GET",
    "path", "/api/users",
    "status", 200,
    "duration_ms", 45)

logger.Warnw("Slow query detected",
    "query", sqlQuery,
    "duration_ms", 1500,
    "threshold_ms", 1000)

logger.Errorw("Database connection failed",
    "host", dbHost,
    "port", dbPort,
    "error", err)
```

## Trace ID Generation

Generate unique trace IDs for request tracking:

```go
traceID := logs.GetTraceID()
logger.Infow("Processing request",
    "trace_id", traceID,
    "user_id", userID)
```

## Best Practices

1. **Use structured logging**: Prefer `logger.Infow()` over `logger.Infof()` for better log parsing
   ```go
   // Good
   logger.Infow("User action", "user_id", userID, "action", "login")

   // Less ideal
   logger.Infof("User %s performed action: %s", userID, "login")
   ```

2. **Choose appropriate log levels**:
   - `Debug`: Detailed information for debugging (e.g., variable values, function calls)
   - `Info`: General informational messages (e.g., request received, task completed)
   - `Warn`: Warning messages that don't stop execution (e.g., deprecated usage, slow queries)
   - `Error`: Error messages that indicate failures (e.g., database connection failed)

3. **Include context**: Always include relevant context in your logs
   ```go
   logger.Errorw("Failed to process payment",
       "user_id", userID,
       "order_id", orderID,
       "amount", amount,
       "error", err)
   ```

4. **Sync before exit**: Always call `logs.Sync()` before your application exits
   ```go
   func main() {
       defer logs.Sync()
       // Your application code
   }
   ```

5. **Initialize early**: Initialize the logger at the start of your application
   ```go
   func main() {
       config := logs.LogConfig{
           Level:  "info",
           Format: "console",
       }
       if err := logs.InitLogger(config); err != nil {
           panic(err)
       }

       // Rest of your application
   }
   ```

## Environment-Specific Configuration

```go
import "os"

func initLogger() {
    env := os.Getenv("APP_ENV")

    var config logs.LogConfig
    switch env {
    case "production":
        config = logs.LogConfig{
            Level:      "info",
            Format:     "json",
            OutputPath: "/var/log/app/x-cli.log",
        }
    case "development":
        config = logs.LogConfig{
            Level:      "debug",
            Format:     "console",
            OutputPath: "stdout",
        }
    default:
        config = logs.DefaultLogConfig()
    }

    if err := logs.InitLogger(config); err != nil {
        panic(err)
    }
}
```

## Migration from fmt.Printf

### Before
```go
fmt.Printf("Processing account: %s\n", accountID)
fmt.Printf("Error: %v\n", err)
```

### After
```go
logger := logs.GetLogger()
logger.Infof("Processing account: %s", accountID)
logger.Errorw("Operation failed",
    "account_id", accountID,
    "error", err)
```

## Thread Safety

The logger is completely thread-safe and can be used from multiple goroutines concurrently:

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        logger := logs.GetLogger()
        logger.Infow("Worker started", "worker_id", id)
    }(i)
}
wg.Wait()
```
