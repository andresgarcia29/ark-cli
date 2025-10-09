package lib

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/andresgarcia29/ark-cli/logs"
)

// ParallelConfig controls the parallelization parameters
type ParallelConfig struct {
	// MaxWorkers defines the maximum number of goroutines that can execute simultaneously
	// This prevents overloading AWS APIs and the local system
	MaxWorkers int

	// Timeout defines the maximum time the entire parallel operation can take
	// If this time is exceeded, all pending operations are cancelled
	Timeout time.Duration

	// RateLimitDelay defines the wait time between the start of each new task
	// This helps prevent overloading AWS APIs with too many simultaneous requests
	RateLimitDelay time.Duration

	// MaxRetries defines how many times a failed operation will be retried
	// Useful for handling temporary network errors or API limits
	MaxRetries int

	// RetryDelay defines how long to wait between retries
	RetryDelay time.Duration
}

// DefaultParallelConfig returns a default configuration optimized for AWS
func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		MaxWorkers:     10,                     // 10 concurrent workers - balance between speed and AWS rate limits
		Timeout:        5 * time.Minute,        // 5 minutes maximum for parallel operations
		RateLimitDelay: 100 * time.Millisecond, // 100ms between tasks to respect rate limits
		MaxRetries:     3,                      // 3 retries for failed operations
		RetryDelay:     1 * time.Second,        // 1 second between retries
	}
}

// ConservativeConfig returns a more conservative configuration for sensitive environments
func ConservativeConfig() ParallelConfig {
	return ParallelConfig{
		MaxWorkers:     5,                      // Fewer workers to be more conservative
		Timeout:        10 * time.Minute,       // More time for operations
		RateLimitDelay: 500 * time.Millisecond, // More delay between requests
		MaxRetries:     5,                      // More retries
		RetryDelay:     2 * time.Second,        // More time between retries
	}
}

// AggressiveConfig returns a more aggressive configuration for maximum performance
func AggressiveConfig() ParallelConfig {
	return ParallelConfig{
		MaxWorkers:     20,                     // More workers for maximum parallelism
		Timeout:        3 * time.Minute,        // Less time for operations
		RateLimitDelay: 50 * time.Millisecond,  // Less delay between requests
		MaxRetries:     2,                      // Fewer retries
		RetryDelay:     500 * time.Millisecond, // Less time between retries
	}
}

// WorkerPool represents a worker pool for executing tasks in parallel
type WorkerPool struct {
	// maxWorkers controls how many goroutines can execute simultaneously
	maxWorkers int
	// semaphore is a channel that acts as a semaphore to control concurrency
	// When full, new goroutines wait until space is freed
	semaphore chan struct{}
}

// NewWorkerPool creates a new worker pool with the specified maximum number
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		// Create a channel with capacity equal to the maximum number of workers
		// This acts as a semaphore: when full, new tasks wait
		semaphore: make(chan struct{}, maxWorkers),
	}
}

// Execute executes a function in the worker pool
// This function blocks until a worker is available
func (wp *WorkerPool) Execute(ctx context.Context, fn func() error) error {
	select {
	// Attempt to acquire a slot in the semaphore
	case wp.semaphore <- struct{}{}:
		// We have a slot! Execute the function
		defer func() {
			// When finished, free the slot so another worker can use it
			<-wp.semaphore
		}()
		return fn()

	// If the context is cancelled while waiting for a slot, return error
	case <-ctx.Done():
		return ctx.Err()
	}
}

// AccountResult represents the result of processing a specific account
type AccountResult struct {
	// AccountID identifies which account was processed
	AccountID string
	// Data contains the obtained data (can be []EKSCluster, []Role, etc.)
	Data interface{}
	// Error contains any error that occurred during processing
	Error error
}

// GetWorkerPool is an alias for NewWorkerPool to facilitate external use
func GetWorkerPool(maxWorkers int) *WorkerPool {
	return NewWorkerPool(maxWorkers)
}

// ExecuteWithRetry executes a function with automatic retries
// This function is useful for operations that can fail temporarily (network, rate limits, etc.)
func ExecuteWithRetry(ctx context.Context, config ParallelConfig, operation func() error) error {
	logger := logs.GetLogger()
	var lastErr error

	// Attempt the operation up to MaxRetries + 1 times (initial attempt + retries)
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// If it's not the first attempt, wait before retrying
		if attempt > 0 {
			logger.Debugw("Retrying operation",
				"attempt", attempt,
				"max_retries", config.MaxRetries,
				"delay", config.RetryDelay)

			// Use select to respect the context during the wait
			select {
			case <-time.After(config.RetryDelay):
				// Wait time completed, continue
			case <-ctx.Done():
				// The context was cancelled, return the error
				return fmt.Errorf("operation cancelled during retry: %w", ctx.Err())
			}
		}

		// Execute the operation
		err := operation()
		if err == nil {
			// Success! No more retries needed
			if attempt > 0 {
				logger.Infow("Operation successful after retries",
					"successful_attempt", attempt+1)
			}
			return nil
		}

		// Save the error to report it if all attempts fail
		lastErr = err

		// If it's the last attempt, don't show retry message
		if attempt < config.MaxRetries {
			logger.Warnw("Attempt failed, retrying",
				"attempt", attempt+1,
				"error", err)
		}
	}

	// All attempts failed
	logger.Errorw("Operation failed after all retries",
		"attempts", config.MaxRetries+1,
		"error", lastErr)
	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// RateLimiter controls the execution rate of operations
type RateLimiter struct {
	// delay is the wait time between operations
	delay time.Duration
	// lastExecution stores when the last operation was executed
	lastExecution time.Time
	// mutex protects concurrent access to lastExecution
	mutex sync.Mutex
}

// NewRateLimiter crea un nuevo rate limiter con el delay especificado
func NewRateLimiter(delay time.Duration) *RateLimiter {
	return &RateLimiter{
		delay: delay,
	}
}

// Wait waits the necessary time to respect the rate limit
// This function ensures we don't execute operations too quickly
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Calculate how long we need to wait
	now := time.Now()
	timeSinceLastExecution := now.Sub(rl.lastExecution)

	if timeSinceLastExecution < rl.delay {
		// We need to wait longer
		waitTime := rl.delay - timeSinceLastExecution

		// Release the mutex during the wait to not block other workers
		rl.mutex.Unlock()

		select {
		case <-time.After(waitTime):
			// Wait time completed
		case <-ctx.Done():
			// The context was cancelled
			rl.mutex.Lock() // Re-acquire the mutex for the defer
			return ctx.Err()
		}

		// Re-acquire the mutex
		rl.mutex.Lock()
	}

	// Update the last execution time
	rl.lastExecution = time.Now()
	return nil
}

// ProcessAccountsInParallel processes multiple AWS accounts in parallel
// This function is generic and can be used for any operation that needs
// to execute in parallel for multiple accounts
func ProcessAccountsInParallel[T any](
	ctx context.Context,
	accounts []string,
	config ParallelConfig,
	processor func(ctx context.Context, accountID string) (T, error),
) (map[string]T, []error) {

	// Create a context with timeout for the entire operation
	// If the operation takes longer than the configured timeout, it will be cancelled automatically
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel() // Important: always cancel the context when finishing

	// WaitGroup allows us to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Channel to receive results from each goroutine
	// Has capacity equal to the number of accounts to prevent blocking
	resultChan := make(chan AccountResult, len(accounts))

	// Create the worker pool to control concurrency
	workerPool := NewWorkerPool(config.MaxWorkers)

	// Create a rate limiter to control the request rate
	rateLimiter := NewRateLimiter(config.RateLimitDelay)

	logger := logs.GetLogger()
	logger.Infow("Starting parallel processing",
		"total_accounts", len(accounts),
		"max_workers", config.MaxWorkers,
		"rate_limit", config.RateLimitDelay,
		"timeout", config.Timeout)

	// Launch a goroutine for each account
	for _, accountID := range accounts {
		// Increment the WaitGroup counter before launching the goroutine
		wg.Add(1)

		// Capture the accountID value in a local variable
		// This is important in Go to avoid problems with closures
		currentAccountID := accountID

		// Launch the goroutine
		go func() {
			// Decrement the WaitGroup counter when we finish
			defer wg.Done()

			logger.Debugf("Processing account: %s", currentAccountID)

			// Execute the processing in the worker pool
			// This will control concurrency automatically
			err := workerPool.Execute(timeoutCtx, func() error {
				// First wait to respect the rate limit
				// This prevents overloading AWS APIs
				if err := rateLimiter.Wait(timeoutCtx); err != nil {
					return fmt.Errorf("rate limit cancelled: %w", err)
				}

				// Now execute the operation with automatic retries
				var result T
				var processingErr error

				retryErr := ExecuteWithRetry(timeoutCtx, config, func() error {
					// Here we execute the specific processing function
					var err error
					result, err = processor(timeoutCtx, currentAccountID)
					processingErr = err
					return err
				})

				// If retries failed, use the last error
				if retryErr != nil {
					processingErr = retryErr
				}

				// Send the result to the channel
				// Use select to handle the case where the context is cancelled
				select {
				case resultChan <- AccountResult{
					AccountID: currentAccountID,
					Data:      result,
					Error:     processingErr,
				}:
					// Result sent successfully
					if processingErr != nil {
						logger.Errorw("Error processing account",
							"account_id", currentAccountID,
							"error", processingErr)
					} else {
						logger.Infow("Account processed successfully",
							"account_id", currentAccountID)
					}
				case <-timeoutCtx.Done():
					// The context was cancelled, we cannot send the result
					return timeoutCtx.Err()
				}
				return nil
			})

			// If there was an error in the worker pool (due to timeout), send the error
			if err != nil {
				select {
				case resultChan <- AccountResult{
					AccountID: currentAccountID,
					Data:      *new(T), // zero value of type T
					Error:     err,
				}:
				case <-timeoutCtx.Done():
					// Cannot send, but it doesn't matter because we're already cancelling
				}
			}
		}()
	}

	// Launch a goroutine to close the channel when all tasks finish
	go func() {
		// Wait for all goroutines to finish
		wg.Wait()
		// Close the channel to indicate there will be no more results
		close(resultChan)
		logger.Debug("All accounts have been processed")
	}()

	// Collect all results from the channel
	results := make(map[string]T)
	var errors []error

	// Read from the channel until it closes
	for result := range resultChan {
		if result.Error != nil {
			// If there was an error, add it to the error list
			errors = append(errors, fmt.Errorf("account %s: %w", result.AccountID, result.Error))
		} else {
			// If successful, add the result to the map
			results[result.AccountID] = result.Data.(T)
		}
	}

	logger.Infow("Parallel processing completed",
		"successful", len(results),
		"errors", len(errors))

	return results, errors
}
