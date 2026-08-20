package ui

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestReasonFlattensWrappedErrors(t *testing.T) {
	err := fmt.Errorf("failed to get clusters: %w",
		fmt.Errorf("failed to login with profile prod: %w",
			errors.New("SSO session expired")))

	sep := " " + GlyphStep + " "
	want := strings.Join([]string{
		"failed to get clusters",
		"failed to login with profile prod",
		"SSO session expired",
	}, sep)

	if got := Reason(err); got != want {
		t.Errorf("Reason() = %q, want %q", got, want)
	}
}

// Wrapping often repeats the same phrase at several levels; showing it once
// keeps the message readable.
func TestReasonDropsRepeatedSegments(t *testing.T) {
	err := fmt.Errorf("failed to assume role: %w", errors.New("failed to assume role: denied"))
	want := "failed to assume role" + " " + GlyphStep + " " + "denied"
	if got := Reason(err); got != want {
		t.Errorf("Reason() = %q, want %q", got, want)
	}
}

func TestReasonNil(t *testing.T) {
	if got := Reason(nil); got != "" {
		t.Errorf("Reason(nil) = %q, want empty", got)
	}
}

// Redirected output and NO_COLOR terminals must get plain ASCII, never boxes
// of question marks. Tests run without a TTY, which is exactly that case.
func TestGlyphsFallBackToASCIIWithoutTTY(t *testing.T) {
	if HasColor() {
		t.Skip("test terminal reports colour support")
	}
	for name, g := range map[string]string{
		"step": GlyphStep, "good": GlyphGood, "bad": GlyphBad,
		"warn": GlyphWarn, "hint": GlyphHint, "cursor": GlyphCurs, "dot": GlyphDot,
	} {
		for _, r := range g {
			if r > 127 {
				t.Errorf("glyph %q = %q contains non-ASCII %q", name, g, r)
			}
		}
	}
}

// lipgloss v2 always emits escape codes, so the writer must strip them when
// the destination is a pipe or file. Otherwise redirected logs fill with ANSI.
func TestOutputStripsColorWhenRedirected(t *testing.T) {
	var buf bytes.Buffer
	orig := Err
	SetErr(&buf)
	t.Cleanup(func() { SetErr(orig) })

	Step("scanning %s", "accounts")
	Done("finished")
	Fail("broke")

	got := buf.String()
	if strings.Contains(got, "\x1b[") {
		t.Errorf("redirected output contains escape codes: %q", got)
	}
	for _, want := range []string{"scanning accounts", "finished", "broke"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestWidthHasSaneFallback(t *testing.T) {
	if w := Width(); w <= 0 {
		t.Errorf("Width() = %d, want a positive fallback", w)
	}
}
