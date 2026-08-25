package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fastLimits() Limits {
	return Limits{MaxWorkers: 4, Timeout: 5 * time.Second, Rate: 1000, MaxRetries: 2, RetryDelay: time.Millisecond}
}

func TestMapConcurrentCollectsResultsAndErrors(t *testing.T) {
	keys := []string{"a", "b", "bad", "c"}

	got, errs := MapConcurrent(context.Background(), keys, fastLimits(),
		func(ctx context.Context, key string) (string, error) {
			if key == "bad" {
				return "", Permanent(errors.New("nope"))
			}
			return strings.ToUpper(key), nil
		})

	if len(got) != 3 {
		t.Errorf("got %d results, want 3: %v", len(got), got)
	}
	if got["a"] != "A" || got["c"] != "C" {
		t.Errorf("wrong values: %v", got)
	}
	if _, ok := got["bad"]; ok {
		t.Error("a failed key must not appear in the results")
	}
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	// The error must name the key it belongs to.
	if !strings.Contains(errs[0].Error(), "bad") {
		t.Errorf("error %q does not identify the key", errs[0])
	}
}

// One failing key must never abort the others.
func TestMapConcurrentIsolatesFailures(t *testing.T) {
	keys := make([]string, 20)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%02d", i)
	}

	got, errs := MapConcurrent(context.Background(), keys, fastLimits(),
		func(ctx context.Context, key string) (int, error) {
			if strings.HasSuffix(key, "3") {
				return 0, Permanent(errors.New("boom"))
			}
			return 1, nil
		})

	if len(got)+len(errs) != len(keys) {
		t.Errorf("accounted for %d of %d keys", len(got)+len(errs), len(keys))
	}
	if len(errs) != 2 {
		t.Errorf("got %d errors, want 2", len(errs))
	}
}

func TestMapConcurrentRespectsWorkerLimit(t *testing.T) {
	keys := make([]string, 30)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%d", i)
	}

	var inFlight, peak atomic.Int64
	limits := fastLimits()
	limits.MaxWorkers = 4

	MapConcurrent(context.Background(), keys, limits, func(ctx context.Context, key string) (bool, error) {
		n := inFlight.Add(1)
		for {
			old := peak.Load()
			if n <= old || peak.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		inFlight.Add(-1)
		return true, nil
	})

	if peak.Load() > int64(limits.MaxWorkers) {
		t.Errorf("ran %d workers at once, limit is %d", peak.Load(), limits.MaxWorkers)
	}
}

func TestMapConcurrentEmptyInput(t *testing.T) {
	got, errs := MapConcurrent(context.Background(), nil, fastLimits(),
		func(ctx context.Context, key string) (int, error) { return 1, nil })
	if len(got) != 0 || len(errs) != 0 {
		t.Errorf("empty input produced %v / %v", got, errs)
	}
}

func TestRetryRetriesThenSucceeds(t *testing.T) {
	var calls int
	err := Retry(context.Background(), fastLimits(), func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if calls != 3 {
		t.Errorf("called %d times, want 3", calls)
	}
}

func TestRetryGivesUpAndWrapsLastError(t *testing.T) {
	sentinel := errors.New("still broken")
	err := Retry(context.Background(), fastLimits(), func() error { return sentinel })

	if !errors.Is(err, sentinel) {
		t.Errorf("error chain lost the cause: %v", err)
	}
	if !strings.Contains(err.Error(), "3 attempts") {
		t.Errorf("error %q should report the attempt count", err)
	}
}

// A missing profile or bad ARN cannot be fixed by waiting.
func TestRetrySkipsPermanentErrors(t *testing.T) {
	var calls int
	start := time.Now()
	limits := fastLimits()
	limits.RetryDelay = 200 * time.Millisecond

	err := Retry(context.Background(), limits, func() error {
		calls++
		return Permanent(errors.New("no such profile"))
	})

	if calls != 1 {
		t.Errorf("permanent error retried %d times", calls)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("permanent error still waited %v", elapsed)
	}
	if !strings.Contains(err.Error(), "no such profile") {
		t.Errorf("error text lost: %v", err)
	}
}

func TestRetryStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int
	err := Retry(ctx, fastLimits(), func() error {
		calls++
		return errors.New("fail")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
	if calls > 1 {
		t.Errorf("kept retrying a cancelled context %d times", calls)
	}
}

func TestMapConcurrentHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	var started atomic.Int64
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, errs := MapConcurrent(ctx, keys, fastLimits(), func(ctx context.Context, key string) (int, error) {
		started.Add(1)
		select {
		case <-time.After(2 * time.Second):
			return 1, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})

	if len(errs) == 0 {
		t.Error("cancellation produced no errors")
	}
}

func TestMapConcurrentOnDoneFiresOncePerKey(t *testing.T) {
	var calls atomic.Int64
	l := Limits{MaxWorkers: 2, Timeout: time.Second, Rate: 100, MaxRetries: 2, RetryDelay: time.Millisecond}
	l.OnDone = func() { calls.Add(1) }

	_, errs := MapConcurrent(context.Background(), []string{"a", "b"}, l,
		func(ctx context.Context, key string) (int, error) {
			return 0, errors.New("always fails")
		})

	if len(errs) != 2 {
		t.Fatalf("errs = %d, want 2", len(errs))
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("OnDone called %d times, want 2 (once per key, not per retry)", got)
	}
}
