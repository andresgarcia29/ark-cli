package ui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// The palette is defined once at startup rather than per render: lipgloss v2
// styles are values, so rebuilding them inside View costs an allocation on
// every frame.
var (
	// profile is the terminal's colour capability, detected once. It also
	// honours NO_COLOR and TERM=dumb through colorprofile.
	profile = colorprofile.Detect(os.Stderr, os.Environ())

	// dark reports whether the terminal background is dark, so the palette can
	// pick contrast that stays readable on light themes too.
	dark = detectDarkBackground()
)

func detectDarkBackground() bool {
	if f, ok := any(os.Stderr).(*os.File); ok {
		return lipgloss.HasDarkBackground(os.Stdin, f)
	}
	return true
}

// adaptive picks between a light-background and a dark-background colour.
func adaptive(light, dark_ string) color.Color {
	if dark {
		return lipgloss.Color(dark_)
	}
	return lipgloss.Color(light)
}

// Palette. Hues are shared across every surface so the CLI reads as one tool:
// violet marks ark's own voice, green success, red failure, amber caution.
var (
	colorAccent = adaptive("#7C3AED", "#B69BFF")
	colorGood   = adaptive("#047857", "#4ADE80")
	colorBad    = adaptive("#BE123C", "#FF6B81")
	colorWarn   = adaptive("#B45309", "#FBBF24")
	colorMuted  = adaptive("#6B7280", "#8B8B9E")
	colorFaint  = adaptive("#9CA3AF", "#5C5C6E")
)

// Styles used across the CLI. Exported so pickers and progress views share
// exactly one definition of each role.
var (
	Accent = lipgloss.NewStyle().Foreground(colorAccent)
	Good   = lipgloss.NewStyle().Foreground(colorGood)
	Bad    = lipgloss.NewStyle().Foreground(colorBad)
	Warned = lipgloss.NewStyle().Foreground(colorWarn)
	Muted  = lipgloss.NewStyle().Foreground(colorMuted)
	Faint  = lipgloss.NewStyle().Foreground(colorFaint)
	Strong = lipgloss.NewStyle().Bold(true)

	// Match highlights the part of a row matching the user's search.
	Match = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	// Selected marks the row under the cursor.
	Selected = lipgloss.NewStyle().Foreground(colorGood).Bold(true)

	// Badge renders a small qualifier chip, such as a profile type.
	Badge = lipgloss.NewStyle().Foreground(colorFaint)

	// Title heads a section.
	Title = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
)

// Glyphs. One symbol per meaning, chosen to stay legible in terminals without
// emoji fonts, with ASCII fallbacks when the terminal has no colour at all.
var (
	GlyphStep = "→"
	GlyphGood = "✓"
	GlyphBad  = "✗"
	GlyphWarn = "!"
	GlyphHint = "↳"
	GlyphCurs = "❯"
	GlyphDot  = "·"
)

func init() {
	if profile == colorprofile.NoTTY || profile == colorprofile.Ascii {
		GlyphStep, GlyphGood, GlyphBad = "-", "OK", "FAIL"
		GlyphWarn, GlyphHint, GlyphCurs, GlyphDot = "!", ">", ">", "-"
	}
}

// HasColor reports whether the terminal can render colour, so callers can skip
// decorative work entirely when it would be discarded.
func HasColor() bool {
	return profile != colorprofile.NoTTY && profile != colorprofile.Ascii
}

// ProgressColors is the gradient used by progress bars, kept here so bars match
// the rest of the palette instead of shipping their own colours.
func ProgressColors() []color.Color {
	return []color.Color{colorAccent, colorGood}
}

// Scheme exposes the palette to callers that theme third-party components,
// so cobra's help output matches everything ark prints itself.
type Scheme struct {
	Accent, Good, Bad, Warn, Muted, Faint color.Color
}

// Colors returns ark's palette.
func Colors() Scheme {
	return Scheme{
		Accent: colorAccent,
		Good:   colorGood,
		Bad:    colorBad,
		Warn:   colorWarn,
		Muted:  colorMuted,
		Faint:  colorFaint,
	}
}
