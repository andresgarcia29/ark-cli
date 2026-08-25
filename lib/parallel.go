package lib

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// Limits bounds a parallel fan-out over AWS APIs.
type Limits struct {
	MaxWorkers int
	Timeout    time.Duration
	Rate       rate.Limit // requests per second across all workers
	MaxRetries int
	RetryDelay time.Duration

	// OnDone, when set, is called once per key after all its attempts finish,
	// so retries do not inflate a progress count.
	OnDone func()
}

// DefaultLimits is tuned to stay under AWS SSO/EKS request throttling.
func DefaultLimits() Limits {
	return Limits{
		MaxWorkers: 10,
		Timeout:    5 * time.Minute,
		Rate:       20,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// Retry runs op until it succeeds or attempts are exhausted, backing off
// exponentially from RetryDelay. Errors marked by Permanent are not retried.
func Retry(ctx context.Context, l Limits, op func() error) error {
	delay := l.RetryDelay
	var err error

	for attempt := 0; attempt <= l.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(delay):
				delay *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err = op(); err == nil {
			return nil
		}
		if isPermanent(err) {
			return err
		}
	}

	return fmt.Errorf("after %d attempts: %w", l.MaxRetries+1, err)
}

// permanentError marks a failure that retrying cannot fix, such as a missing
// profile or expired credentials.
type permanentError struct{ err error }

func (p permanentError) Error() string { return p.err.Error() }
func (p permanentError) Unwrap() error { return p.err }

// Permanent wraps err so Retry gives up immediately instead of backing off.
func Permanent(err error) error { return permanentError{err} }

func isPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}

// MapConcurrent applies fn to every key, returning the successful results by
// key alongside the errors from the rest. A failing key never aborts the others.
func MapConcurrent[T any](
	ctx context.Context,
	keys []string,
	l Limits,
	fn func(ctx context.Context, key string) (T, error),
) (map[string]T, []error) {
	if len(keys) == 0 {
		return map[string]T{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, l.Timeout)
	defer cancel()

	limiter := rate.NewLimiter(l.Rate, l.MaxWorkers)
	results := make([]T, len(keys))
	failures := make([]error, len(keys))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(l.MaxWorkers)

	for i, key := range keys {
		g.Go(func() error {
			if l.OnDone != nil {
				defer l.OnDone()
			}
			if err := limiter.Wait(gctx); err != nil {
				failures[i] = fmt.Errorf("%s: %w", key, err)
				return nil
			}
			err := Retry(gctx, l, func() error {
				var err error
				results[i], err = fn(gctx, key)
				return err
			})
			if err != nil {
				failures[i] = fmt.Errorf("%s: %w", key, err)
			}
			return nil
		})
	}
	_ = g.Wait()

	out := make(map[string]T, len(keys))
	var errs []error
	for i, key := range keys {
		if failures[i] != nil {
			errs = append(errs, failures[i])
			continue
		}
		out[key] = results[i]
	}
	return out, errs
}
