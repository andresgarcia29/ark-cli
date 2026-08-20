package services_aws

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
)

// throttleCount records how many AWS requests came back throttled (HTTP 429 /
// ThrottlingException) across the whole run. Throttling is handled silently by
// the retryer, so without this the user has no idea why a scan crawled.
var throttleCount atomic.Int64

// ThrottleCount returns how many throttled responses AWS has returned so far.
func ThrottleCount() int64 { return throttleCount.Load() }

// scannedAccounts and totalAccounts track how far a multi-account scan has got,
// so a long run can show progress instead of an idle spinner.
var scannedAccounts, totalAccounts atomic.Int64

// ScanProgress reports how many accounts have been scanned out of the total.
func ScanProgress() (done, total int64) {
	return scannedAccounts.Load(), totalAccounts.Load()
}

// ResetThrottleCount clears the per-command counters.
func ResetThrottleCount() {
	throttleCount.Store(0)
	scannedAccounts.Store(0)
	totalAccounts.Store(0)
}

// countingRetryer is an adaptive retryer that also tallies throttled requests.
// Adaptive mode keeps a client-side token bucket, so once AWS starts throttling
// it slows every caller down instead of hammering the API into a longer penalty.
type countingRetryer struct {
	aws.Retryer
	throttles retry.IsErrorThrottles
}

func (r *countingRetryer) IsErrorRetryable(err error) bool {
	if r.throttles.IsErrorThrottle(err) == aws.TrueTernary {
		throttleCount.Add(1)
	}
	return r.Retryer.IsErrorRetryable(err)
}

// throttleStatusCodes are the HTTP statuses that mean "slow down" even when the
// response carries no modelled error code, which is how a plain 429 arrives.
var throttleStatusCodes = map[int]struct{}{
	429: {},
	503: {},
}

// httpStatusThrottle recognises throttling from the HTTP status alone.
type httpStatusThrottle struct{}

func (httpStatusThrottle) IsErrorThrottle(err error) aws.Ternary {
	var resp interface{ HTTPStatusCode() int }
	if errors.As(err, &resp) {
		if _, ok := throttleStatusCodes[resp.HTTPStatusCode()]; ok {
			return aws.TrueTernary
		}
	}
	return aws.UnknownTernary
}

// newRetryer builds the retryer every ark AWS client uses. AWS defaults to 3
// attempts, which a wide multi-account scan blows through immediately.
func newRetryer() aws.Retryer {
	throttles := append([]retry.IsErrorThrottle{httpStatusThrottle{}}, retry.DefaultThrottles...)

	adaptive := retry.NewAdaptiveMode(func(o *retry.AdaptiveModeOptions) {
		o.Throttles = throttles
		o.StandardOptions = append(o.StandardOptions, func(so *retry.StandardOptions) {
			so.MaxAttempts = 8
			so.MaxBackoff = 20 * time.Second
		})
	})

	return &countingRetryer{
		Retryer:   adaptive,
		throttles: retry.IsErrorThrottles(throttles),
	}
}
