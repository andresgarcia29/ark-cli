// Package ui is the single place ark writes human-facing output.
// Results go to stdout; everything else goes to stderr so piping stays useful.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-isatty"
)

var (
	// Out receives command results meant to be piped or captured.
	Out io.Writer = os.Stdout
	// Err receives progress, warnings and failures. Bubble Tea writes here
	// directly, because it manages colour degradation itself.
	Err io.Writer = os.Stderr

	quiet bool
)

// styledErr downgrades or strips ANSI to match what the destination can show.
// lipgloss v2 dropped the global renderer, so styles always emit escape codes
// and it is the writer's job to adapt them; without this, redirecting output
// would litter the file with colour codes.
var styledErr = colorprofile.NewWriter(os.Stderr, os.Environ())

// SetErr redirects diagnostic output, keeping colour handling consistent.
func SetErr(w io.Writer) {
	Err = w
	styledErr = colorprofile.NewWriter(w, os.Environ())
}

// SetQuiet silences everything except Result and Fail.
func SetQuiet(q bool) { quiet = q }

// Interactive reports whether stderr is a terminal, so callers can skip
// animations when output is redirected or piped.
func Interactive() bool {
	f, ok := Err.(*os.File)
	return ok && isatty.IsTerminal(f.Fd())
}

// Width returns the usable terminal width, falling back to a sane default when
// the size cannot be determined.
func Width() int {
	f, ok := Err.(*os.File)
	if !ok {
		return 80
	}
	w, _, err := term.GetSize(f.Fd())
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func emit(format string, a ...any) {
	fmt.Fprintln(styledErr, fmt.Sprintf(format, a...))
}

// prefixed writes a glyph-led line, indenting any wrapped continuation so
// multi-line messages stay aligned under their marker.
func prefixed(style lipgloss.Style, glyph, format string, a ...any) {
	body := fmt.Sprintf(format, a...)
	marker := style.Render(glyph)
	lines := strings.Split(body, "\n")

	emit("%s %s", marker, lines[0])
	for _, l := range lines[1:] {
		emit("  %s", Muted.Render(l))
	}
}

// Step announces the start of a unit of work.
func Step(format string, a ...any) {
	if quiet {
		return
	}
	prefixed(Accent, GlyphStep, format, a...)
}

// Done reports a completed unit of work.
func Done(format string, a ...any) {
	if quiet {
		return
	}
	prefixed(Good, GlyphGood, format, a...)
}

// Warn reports something the user should know but that does not stop the run.
func Warn(format string, a ...any) {
	if quiet {
		return
	}
	prefixed(Warned, GlyphWarn, format, a...)
}

// Fail reports a failure. It is never silenced by quiet mode.
func Fail(format string, a ...any) {
	prefixed(Bad, GlyphBad, format, a...)
}

// Detail prints supporting context, indented under the last message.
func Detail(format string, a ...any) {
	if quiet {
		return
	}
	emit("  %s", Muted.Render(fmt.Sprintf(format, a...)))
}

// Hint suggests the user's next command.
func Hint(format string, a ...any) {
	if quiet {
		return
	}
	emit("  %s %s", Faint.Render(GlyphHint), Muted.Render(fmt.Sprintf(format, a...)))
}

// Result prints command output meant for capture; it always goes to stdout.
func Result(format string, a ...any) {
	fmt.Fprintln(Out, fmt.Sprintf(format, a...))
}

// Blank separates sections.
func Blank() {
	if !quiet {
		fmt.Fprintln(styledErr)
	}
}

// Heading prints a section heading.
func Heading(s string) {
	if quiet {
		return
	}
	emit("")
	emit("%s", Title.Render(s))
}

// Panel frames text the user must act on, such as a device login code.
func Panel(heading string, rows ...string) {
	if quiet {
		return
	}

	body := Strong.Render(heading)
	for _, r := range rows {
		body += "\n" + r
	}

	if !HasColor() {
		emit("%s", body)
		return
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(0, 2).
		MaxWidth(Width())
	emit("%s", box.Render(body))
}

// Summary prints an aligned key/value block, used for end-of-run reports.
func Summary(pairs ...[2]string) {
	if quiet || len(pairs) == 0 {
		return
	}

	width := 0
	for _, p := range pairs {
		if n := lipgloss.Width(p[0]); n > width {
			width = n
		}
	}

	label := lipgloss.NewStyle().Foreground(colorMuted).Width(width + 1)
	for _, p := range pairs {
		emit("  %s %s", label.Render(p[0]), p[1])
	}
}

// Reason renders an error for a human: one line, no Go type noise, and with
// the wrapped causes flattened into a single arrow-separated trail.
func Reason(err error) string {
	if err == nil {
		return ""
	}
	parts := strings.Split(err.Error(), ": ")
	seen := make(map[string]bool, len(parts))
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return strings.Join(out, " "+GlyphStep+" ")
}
