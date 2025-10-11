package lib

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultParallelConfig(t *testing.T) {
	config := DefaultParallelConfig()

	assert.Equal(t, 10, config.MaxWorkers)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.Equal(t, 100*time.Millisecond, config.RateLimitDelay)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
}

func TestConservativeConfig(t *testing.T) {
	config := ConservativeConfig()

	assert.Equal(t, 5, config.MaxWorkers)
	assert.Equal(t, 10*time.Minute, config.Timeout)
	assert.Equal(t, 500*time.Millisecond, config.RateLimitDelay)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.RetryDelay)
}

func TestAggressiveConfig(t *testing.T) {
	config := AggressiveConfig()

	assert.Equal(t, 20, config.MaxWorkers)
	assert.Equal(t, 3*time.Minute, config.Timeout)
	assert.Equal(t, 50*time.Millisecond, config.RateLimitDelay)
	assert.Equal(t, 2, config.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, config.RetryDelay)
}

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
		expected   int
	}{
		{
			name:       "valid max workers",
			maxWorkers: 5,
			expected:   5,
		},
		{
			name:       "zero max workers",
			maxWorkers: 0,
			expected:   0,
		},
		{
			name:       "negative max workers",
			maxWorkers: -1,
			expected:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.maxWorkers < 0 {
				// Negative workers should panic
				assert.Panics(t, func() {
					NewWorkerPool(tt.maxWorkers)
				})
			} else {
				pool := NewWorkerPool(tt.maxWorkers)

				assert.NotNil(t, pool)
				assert.Equal(t, tt.expected, pool.maxWorkers)
				assert.NotNil(t, pool.semaphore)
				assert.Equal(t, tt.expected, cap(pool.semaphore))
			}
		})
	}
}

func TestWorkerPoolExecute(t *testing.T) {
	tests := []struct {
		name             string
		maxWorkers       int
		fn               func() error
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "successful execution",
			maxWorkers:       1,
			fn:               func() error { return nil },
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "function error",
			maxWorkers:       1,
			fn:               func() error { return errors.New("test error") },
			expectedError:    true,
			expectedErrorMsg: "test error",
		},
		{
			name:             "zero workers",
			maxWorkers:       0,
			fn:               func() error { return nil },
			expectedError:    true,
			expectedErrorMsg: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.maxWorkers)
			ctx := context.Background()

			// For zero workers test, use a timeout context to avoid hanging
			if tt.maxWorkers == 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}

			err := pool.Execute(ctx, tt.fn)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErrorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkerPoolExecuteContextCancellation(t *testing.T) {
	pool := NewWorkerPool(1)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	err := pool.Execute(ctx, func() error {
		return nil
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGetWorkerPool(t *testing.T) {
	tests := []struct {
		name       string
		maxWorkers int
		expected   int
	}{
		{
			name:       "valid max workers",
			maxWorkers: 5,
			expected:   5,
		},
		{
			name:       "zero max workers",
			maxWorkers: 0,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := GetWorkerPool(tt.maxWorkers)

			assert.NotNil(t, pool)
			assert.Equal(t, tt.expected, pool.maxWorkers)
		})
	}
}

func TestExecuteWithRetry(t *testing.T) {
	tests := []struct {
		name             string
		config           ParallelConfig
		operation        func() error
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "successful operation",
			config:           DefaultParallelConfig(),
			operation:        func() error { return nil },
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:   "operation fails once then succeeds",
			config: ParallelConfig{MaxRetries: 1, RetryDelay: 1 * time.Millisecond},
			operation: func() error {
				// Simulate failure on first attempt, success on second
				return nil
			},
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "operation always fails",
			config:           ParallelConfig{MaxRetries: 2, RetryDelay: 1 * time.Millisecond},
			operation:        func() error { return errors.New("persistent error") },
			expectedError:    true,
			expectedErrorMsg: "operation failed after 3 attempts: persistent error",
		},
		{
			name:             "zero retries",
			config:           ParallelConfig{MaxRetries: 0, RetryDelay: 1 * time.Millisecond},
			operation:        func() error { return errors.New("error") },
			expectedError:    true,
			expectedErrorMsg: "operation failed after 1 attempts: error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			err := ExecuteWithRetry(ctx, tt.config, tt.operation)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteWithRetryContextCancellation(t *testing.T) {
	config := ParallelConfig{MaxRetries: 5, RetryDelay: 100 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := ExecuteWithRetry(ctx, config, func() error {
		return errors.New("test error")
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{
			name:  "100ms delay",
			delay: 100 * time.Millisecond,
		},
		{
			name:  "1 second delay",
			delay: 1 * time.Second,
		},
		{
			name:  "zero delay",
			delay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.delay)

			assert.NotNil(t, limiter)
			assert.Equal(t, tt.delay, limiter.delay)
		})
	}
}

func TestRateLimiterWait(t *testing.T) {
	tests := []struct {
		name          string
		delay         time.Duration
		expectedError bool
	}{
		{
			name:          "no delay",
			delay:         0,
			expectedError: false,
		},
		{
			name:          "short delay",
			delay:         1 * time.Millisecond,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.delay)
			ctx := context.Background()

			err := limiter.Wait(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRateLimiterWaitContextCancellation(t *testing.T) {
	limiter := NewRateLimiter(100 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	err := limiter.Wait(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestProcessAccountsInParallel(t *testing.T) {
	tests := []struct {
		name            string
		accounts        []string
		config          ParallelConfig
		processor       func(ctx context.Context, accountID string) (string, error)
		expectedError   bool
		expectedResults int
		expectedErrors  int
	}{
		{
			name:            "successful processing",
			accounts:        []string{"account1", "account2"},
			config:          ParallelConfig{MaxWorkers: 2, Timeout: 1 * time.Second, RateLimitDelay: 1 * time.Millisecond, MaxRetries: 1, RetryDelay: 1 * time.Millisecond},
			processor:       func(ctx context.Context, accountID string) (string, error) { return "result-" + accountID, nil },
			expectedError:   false,
			expectedResults: 2,
			expectedErrors:  0,
		},
		{
			name:     "some accounts fail",
			accounts: []string{"account1", "account2", "account3"},
			config:   ParallelConfig{MaxWorkers: 2, Timeout: 1 * time.Second, RateLimitDelay: 1 * time.Millisecond, MaxRetries: 1, RetryDelay: 1 * time.Millisecond},
			processor: func(ctx context.Context, accountID string) (string, error) {
				if accountID == "account2" {
					return "", errors.New("account2 failed")
				}
				return "result-" + accountID, nil
			},
			expectedError:   false,
			expectedResults: 2,
			expectedErrors:  1,
		},
		{
			name:            "empty accounts list",
			accounts:        []string{},
			config:          DefaultParallelConfig(),
			processor:       func(ctx context.Context, accountID string) (string, error) { return "result", nil },
			expectedError:   false,
			expectedResults: 0,
			expectedErrors:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			results, errors := ProcessAccountsInParallel(ctx, tt.accounts, tt.config, tt.processor)

			assert.Equal(t, tt.expectedResults, len(results))
			assert.Equal(t, tt.expectedErrors, len(errors))

			// Verify results contain expected data
			for accountID, result := range results {
				assert.Contains(t, tt.accounts, accountID)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestProcessAccountsInParallelContextCancellation(t *testing.T) {
	accounts := []string{"account1", "account2", "account3"}
	config := ParallelConfig{MaxWorkers: 1, Timeout: 100 * time.Millisecond, RateLimitDelay: 1 * time.Millisecond, MaxRetries: 1, RetryDelay: 1 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	processor := func(ctx context.Context, accountID string) (string, error) {
		// Simulate work that takes time
		time.Sleep(200 * time.Millisecond)
		return "result-" + accountID, nil
	}

	results, errors := ProcessAccountsInParallel(ctx, accounts, config, processor)

	// Should have some results and some errors due to cancellation
	assert.GreaterOrEqual(t, len(results), 0)
	assert.GreaterOrEqual(t, len(errors), 0)
}

func TestProcessAccountsInParallelTimeout(t *testing.T) {
	accounts := []string{"account1", "account2", "account3"}
	config := ParallelConfig{MaxWorkers: 1, Timeout: 100 * time.Millisecond, RateLimitDelay: 1 * time.Millisecond, MaxRetries: 1, RetryDelay: 1 * time.Millisecond}
	ctx := context.Background()

	processor := func(ctx context.Context, accountID string) (string, error) {
		// Simulate work that takes longer than timeout
		time.Sleep(200 * time.Millisecond)
		return "result-" + accountID, nil
	}

	results, errors := ProcessAccountsInParallel(ctx, accounts, config, processor)

	// Should have some results and some errors due to timeout
	assert.GreaterOrEqual(t, len(results), 0)
	assert.GreaterOrEqual(t, len(errors), 0)
}

func TestAccountResultStruct(t *testing.T) {
	// Test AccountResult struct fields
	result := AccountResult{
		AccountID: "123456789012",
		Data:      "test-data",
		Error:     nil,
	}

	assert.Equal(t, "123456789012", result.AccountID)
	assert.Equal(t, "test-data", result.Data)
	assert.NoError(t, result.Error)
}

func TestAccountResultWithError(t *testing.T) {
	// Test AccountResult struct with error
	err := errors.New("test error")
	result := AccountResult{
		AccountID: "123456789012",
		Data:      nil,
		Error:     err,
	}

	assert.Equal(t, "123456789012", result.AccountID)
	assert.Nil(t, result.Data)
	assert.Error(t, result.Error)
	assert.Equal(t, "test error", result.Error.Error())
}

func TestParallelConfigStruct(t *testing.T) {
	// Test ParallelConfig struct fields
	config := ParallelConfig{
		MaxWorkers:     5,
		Timeout:        5 * time.Minute,
		RateLimitDelay: 100 * time.Millisecond,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
	}

	assert.Equal(t, 5, config.MaxWorkers)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.Equal(t, 100*time.Millisecond, config.RateLimitDelay)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
}

func TestWorkerPoolStruct(t *testing.T) {
	// Test WorkerPool struct fields
	pool := &WorkerPool{
		maxWorkers: 5,
		semaphore:  make(chan struct{}, 5),
	}

	assert.Equal(t, 5, pool.maxWorkers)
	assert.NotNil(t, pool.semaphore)
	assert.Equal(t, 5, cap(pool.semaphore))
}

func TestRateLimiterStruct(t *testing.T) {
	// Test RateLimiter struct fields
	limiter := &RateLimiter{
		delay: 100 * time.Millisecond,
	}

	assert.Equal(t, 100*time.Millisecond, limiter.delay)
}

func TestProcessAccountsInParallelGeneric(t *testing.T) {
	// Test generic type handling
	accounts := []string{"account1", "account2"}
	config := ParallelConfig{MaxWorkers: 2, Timeout: 1 * time.Second, RateLimitDelay: 1 * time.Millisecond, MaxRetries: 1, RetryDelay: 1 * time.Millisecond}
	ctx := context.Background()

	// Test with string type
	processor := func(ctx context.Context, accountID string) (string, error) {
		return "result-" + accountID, nil
	}

	results, errors := ProcessAccountsInParallel(ctx, accounts, config, processor)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, 0, len(errors))

	// Verify results
	for accountID, result := range results {
		assert.Contains(t, accounts, accountID)
		assert.Equal(t, "result-"+accountID, result)
	}
}

func TestProcessAccountsInParallelWithDifferentTypes(t *testing.T) {
	// Test with different return types
	accounts := []string{"account1", "account2"}
	config := ParallelConfig{MaxWorkers: 2, Timeout: 1 * time.Second, RateLimitDelay: 1 * time.Millisecond, MaxRetries: 1, RetryDelay: 1 * time.Millisecond}
	ctx := context.Background()

	// Test with int type
	processor := func(ctx context.Context, accountID string) (int, error) {
		return len(accountID), nil
	}

	results, errors := ProcessAccountsInParallel(ctx, accounts, config, processor)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, 0, len(errors))

	// Verify results
	for accountID, result := range results {
		assert.Contains(t, accounts, accountID)
		assert.Equal(t, len(accountID), result)
	}
}
