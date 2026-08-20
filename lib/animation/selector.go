package animation

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/andresgarcia29/ark-cli/lib/ui"
)

// ErrCancelled means the user dismissed the picker. It is a normal outcome,
// not a failure, and callers should exit quietly.
var ErrCancelled = errors.New("selection cancelled")

const visibleRows = 10

// Item is one row of a picker.
type Item struct {
	// Title is the primary label and the main search target.
	Title string
	// Badge is a short qualifier shown after the title, such as the profile type.
	Badge string
	// Detail is dimmed supporting text.
	Detail string
	// Marked highlights the row as the currently active choice.
	Marked bool

	lowerTitle  string
	lowerSearch string
}

// prepare precomputes the lowercased forms so filtering does not re-allocate
// on every keystroke; a large account can have hundreds of profiles.
func (i *Item) prepare() {
	i.lowerTitle = strings.ToLower(i.Title)
	i.lowerSearch = strings.ToLower(i.Title + " " + i.Badge + " " + i.Detail)
}

type picker struct {
	heading  string
	items    []Item
	filtered []int
	cursor   int
	offset   int
	query    string
	chosen   int
	done     bool
	width    int
}

func newPicker(heading string, items []Item) *picker {
	for i := range items {
		items[i].prepare()
	}

	p := &picker{heading: heading, items: items, chosen: -1, width: ui.Width()}
	p.filter()
	for n, idx := range p.filtered {
		if items[idx].Marked {
			p.cursor = n
			break
		}
	}
	p.scroll()
	return p
}

func (p *picker) filter() {
	p.filtered = p.filtered[:0]
	q := strings.ToLower(strings.TrimSpace(p.query))
	for i := range p.items {
		if q == "" || strings.Contains(p.items[i].lowerSearch, q) {
			p.filtered = append(p.filtered, i)
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
	p.scroll()
}

// scroll keeps the cursor inside the visible window.
func (p *picker) scroll() {
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+visibleRows {
		p.offset = p.cursor - visibleRows + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p *picker) Init() tea.Cmd { return nil }

// Update handles keys. Bubble Tea v2 reports printable input as Key.Text, so
// letters always reach the search box and never collide with shortcuts.
func (p *picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		return p, nil

	case tea.KeyPressMsg:
		key := msg.Key()

		if key.Mod&tea.ModCtrl != 0 {
			switch key.Code {
			case 'c', 'd':
				return p, tea.Quit
			case 'u':
				p.query = ""
				p.filter()
				return p, nil
			}
			return p, nil
		}

		switch key.Code {
		case tea.KeyEscape:
			if p.query != "" {
				p.query = ""
				p.filter()
				return p, nil
			}
			return p, tea.Quit
		case tea.KeyUp:
			p.move(-1)
			return p, nil
		case tea.KeyDown:
			p.move(1)
			return p, nil
		case tea.KeyPgUp:
			p.move(-visibleRows)
			return p, nil
		case tea.KeyPgDown:
			p.move(visibleRows)
			return p, nil
		case tea.KeyHome:
			p.move(-len(p.filtered))
			return p, nil
		case tea.KeyEnd:
			p.move(len(p.filtered))
			return p, nil
		case tea.KeyBackspace:
			if p.query != "" {
				p.query = p.query[:len(p.query)-1]
				p.filter()
			}
			return p, nil
		case tea.KeyEnter:
			if len(p.filtered) > 0 {
				p.chosen = p.filtered[p.cursor]
				p.done = true
				return p, tea.Quit
			}
			return p, nil
		}

		// Any printable input, including accented and non-latin characters,
		// extends the search.
		if key.Text != "" {
			p.query += key.Text
			p.filter()
		}
		return p, nil
	}
	return p, nil
}

func (p *picker) move(delta int) {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor = min(max(p.cursor+delta, 0), len(p.filtered)-1)
	p.scroll()
}

// highlight renders title in base, with the matched span picked out, so it is
// obvious why a row survived the filter without losing the row's own styling.
func highlight(item Item, query string, base lipgloss.Style) string {
	if query == "" {
		return base.Render(item.Title)
	}
	idx := strings.Index(item.lowerTitle, strings.ToLower(query))
	if idx < 0 {
		return base.Render(item.Title)
	}
	end := idx + len(query)
	return base.Render(item.Title[:idx]) +
		ui.Match.Render(item.Title[idx:end]) +
		base.Render(item.Title[end:])
}

func (p *picker) View() tea.View {
	if p.done {
		return tea.NewView("")
	}

	var b strings.Builder
	b.WriteString(ui.Title.Render(p.heading))
	b.WriteString("  ")
	b.WriteString(ui.Faint.Render(fmt.Sprintf("%d/%d", len(p.filtered), len(p.items))))
	b.WriteString("\n")

	// Search line doubles as the key hint until the user starts typing.
	if p.query == "" {
		b.WriteString(ui.Faint.Render("type to filter  " + ui.GlyphDot + "  ↑↓ move  " + ui.GlyphDot + "  enter select  " + ui.GlyphDot + "  esc cancel"))
	} else {
		b.WriteString(ui.Muted.Render("filter ") + ui.Match.Render(p.query) + ui.Faint.Render("▌"))
	}
	b.WriteString("\n\n")

	if len(p.filtered) == 0 {
		b.WriteString(ui.Warned.Render("  nothing matches " + p.query))
		b.WriteString("\n")
		return tea.NewView(b.String())
	}

	if p.offset > 0 {
		b.WriteString(ui.Faint.Render(fmt.Sprintf("   %d more above", p.offset)))
		b.WriteString("\n")
	}

	end := min(p.offset+visibleRows, len(p.filtered))
	for n := p.offset; n < end; n++ {
		item := p.items[p.filtered[n]]
		selected := n == p.cursor

		cursor := "  "
		base := lipgloss.NewStyle()
		switch {
		case selected:
			cursor = ui.Accent.Render(ui.GlyphCurs) + " "
			base = ui.Selected
		case item.Marked:
			base = ui.Warned
		}

		row := cursor + highlight(item, p.query, base)
		if item.Badge != "" {
			row += " " + ui.Badge.Render("["+item.Badge+"]")
		}
		if item.Marked {
			row += " " + ui.Warned.Render(ui.GlyphDot+" active")
		}
		b.WriteString(row + "\n")

		// Detail sits on its own dimmed line under the selected row only, so
		// the list stays scannable but the choice is fully described.
		if selected && item.Detail != "" {
			b.WriteString("    " + ui.Muted.Render(item.Detail) + "\n")
		}
	}

	if end < len(p.filtered) {
		b.WriteString(ui.Faint.Render(fmt.Sprintf("   %d more below", len(p.filtered)-end)))
		b.WriteString("\n")
	}

	return tea.NewView(b.String())
}

// Select renders an interactive picker and returns the chosen index.
// It returns ErrCancelled if the user dismisses it.
func Select(heading string, items []Item) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("nothing to choose from")
	}

	p := newPicker(heading, items)
	final, err := tea.NewProgram(p, tea.WithOutput(ui.Err)).Run()
	if err != nil {
		return 0, fmt.Errorf("could not show selector: %w", err)
	}

	result, ok := final.(*picker)
	if !ok || result.chosen < 0 {
		return 0, ErrCancelled
	}
	return result.chosen, nil
}
