package cmd

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/andresgarcia29/ark-cli/lib/ui"
	"github.com/charmbracelet/fang"
)

// helpScheme dresses cobra's generated help and error output in ark's palette,
// so `ark --help` looks like the rest of the CLI rather than stock cobra.
func helpScheme(c lipgloss.LightDarkFunc) fang.ColorScheme {
	p := ui.Colors()
	base := c(lipgloss.Color("#31333B"), lipgloss.Color("#DDDDDD"))

	return fang.ColorScheme{
		Base:           base,
		Title:          p.Accent,
		Description:    base,
		Codeblock:      c(lipgloss.Color("#F2F2F2"), lipgloss.Color("#2C2C34")),
		Program:        p.Accent,
		Command:        p.Good,
		DimmedArgument: p.Faint,
		Comment:        p.Faint,
		Flag:           p.Good,
		FlagDefault:    p.Faint,
		QuotedString:   p.Warn,
		Argument:       base,
		Help:           p.Muted,
		Dash:           p.Faint,
		ErrorHeader:    [2]color.Color{lipgloss.Color("#FFFFFF"), p.Bad},
		ErrorDetails:   p.Muted,
	}
}
