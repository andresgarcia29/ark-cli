package services_aws

import (
	"errors"
	"net/http"
	"testing"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type apiErr struct{ code string }

func (e *apiErr) Error() string                 { return e.code }
func (e *apiErr) ErrorCode() string             { return e.code }
func (e *apiErr) ErrorMessage() string          { return e.code }
func (e *apiErr) ErrorFault() smithy.ErrorFault { return smithy.FaultServer }

func TestRetryerCountsThrottles(t *testing.T) {
	ResetThrottleCount()
	r := newRetryer()

	// The codes AWS actually returns when it rate limits a wide scan.
	for _, code := range []string{"ThrottlingException", "TooManyRequestsException", "RequestLimitExceeded"} {
		if !r.IsErrorRetryable(&apiErr{code: code}) {
			t.Errorf("%s should be retryable", code)
		}
	}
	if got := ThrottleCount(); got != 3 {
		t.Errorf("ThrottleCount = %d, want 3", got)
	}

	// A bare HTTP 429 with no modelled error code must count too.
	ResetThrottleCount()
	r.IsErrorRetryable(&awshttp.ResponseError{
		ResponseError: &smithyhttp.ResponseError{
			Response: &smithyhttp.Response{Response: &http.Response{StatusCode: 429}},
			Err:      errors.New("Too Many Requests"),
		},
	})
	if ThrottleCount() == 0 {
		t.Error("a bare HTTP 429 was not recorded as throttling")
	}
}

func TestScanProgressResets(t *testing.T) {
	totalAccounts.Store(40)
	scannedAccounts.Store(12)
	throttleCount.Store(7)

	if done, total := ScanProgress(); done != 12 || total != 40 {
		t.Errorf("ScanProgress = %d/%d, want 12/40", done, total)
	}

	ResetThrottleCount()
	done, total := ScanProgress()
	if done != 0 || total != 0 || ThrottleCount() != 0 {
		t.Errorf("counters not reset: %d/%d throttled=%d", done, total, ThrottleCount())
	}
}
