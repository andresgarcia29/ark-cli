package ui

import (
	"errors"
	"fmt"
	"testing"
)

func TestReasonFlattensWrappedErrors(t *testing.T) {
	err := fmt.Errorf("failed to get clusters: %w",
		fmt.Errorf("failed to login with profile prod: %w",
			errors.New("SSO session expired")))

	got := Reason(err)
	want := "failed to get clusters → failed to login with profile prod → SSO session expired"
	if got != want {
		t.Errorf("Reason() = %q, want %q", got, want)
	}
}

// Wrapping often repeats the same phrase at several levels; showing it once
// keeps the message readable.
func TestReasonDropsRepeatedSegments(t *testing.T) {
	err := fmt.Errorf("failed to assume role: %w", errors.New("failed to assume role: denied"))
	if got, want := Reason(err), "failed to assume role → denied"; got != want {
		t.Errorf("Reason() = %q, want %q", got, want)
	}
}

func TestReasonNil(t *testing.T) {
	if got := Reason(nil); got != "" {
		t.Errorf("Reason(nil) = %q, want empty", got)
	}
}
