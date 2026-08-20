// Package ui is the single place ark writes human-facing output.
// Results go to stdout; everything else goes to stderr so piping stays useful.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var (
	// Out receives command results meant to be piped or captured.
	Out io.Writer = os.Stdout
	// Err receives progress, warnings and failures.
	Err io.Writer = os.Stderr

	quiet bool
)

// Palette is shared by every view so the CLI reads as one program.
var (
	Accent = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	Good   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	Bad    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	Warned = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Muted  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	Strong = lipgloss.NewStyle().Bold(true)
)

// SetQuiet silences everything except Result and Fail.
func SetQuiet(q bool) { quiet = q }

// Interactive reports whether stderr is a terminal, so callers can skip
// animations when output is redirected or piped.
func Interactive() bool {
	f, ok := Err.(*os.File)
	return ok && isatty.IsTerminal(f.Fd())
}

func line(w io.Writer, s string) {
	fmt.Fprintln(w, s)
}

// Step announces the start of a unit of work.
func Step(format string, a ...any) {
	if quiet {
		return
	}
	line(Err, Accent.Render("→ ")+fmt.Sprintf(format, a...))
}

// Done reports a completed unit of work.
func Done(format string, a ...any) {
	if quiet {
		return
	}
	line(Err, Good.Render("✓ ")+fmt.Sprintf(format, a...))
}

// Warn reports something the user should know but that does not stop the run.
func Warn(format string, a ...any) {
	if quiet {
		return
	}
	line(Err, Warned.Render("! ")+fmt.Sprintf(format, a...))
}

// Fail reports a failure. It is never silenced by quiet mode.
func Fail(format string, a ...any) {
	line(Err, Bad.Render("✗ ")+fmt.Sprintf(format, a...))
}

// Detail prints supporting context, indented under the last message.
func Detail(format string, a ...any) {
	if quiet {
		return
	}
	line(Err, Muted.Render("  "+fmt.Sprintf(format, a...)))
}

// Hint suggests the user's next command.
func Hint(format string, a ...any) {
	if quiet {
		return
	}
	line(Err, Muted.Render("  ↳ "+fmt.Sprintf(format, a...)))
}

// Result prints command output meant for capture; it always goes to stdout.
func Result(format string, a ...any) {
	line(Out, fmt.Sprintf(format, a...))
}

// Blank separates sections.
func Blank() {
	if !quiet {
		fmt.Fprintln(Err)
	}
}

// Title prints a section heading.
func Title(s string) {
	if quiet {
		return
	}
	line(Err, "")
	line(Err, Accent.Render(s))
}

// Panel frames text that the user must act on, such as a device login code.
func Panel(heading string, rows ...string) {
	if quiet {
		return
	}
	body := Strong.Render(heading)
	for _, r := range rows {
		body += "\n" + r
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 2)
	line(Err, box.Render(body))
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
	return strings.Join(out, " → ")
}
