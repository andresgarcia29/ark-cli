package animation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/andresgarcia29/ark-cli/lib/ui"
	tea "github.com/charmbracelet/bubbletea"
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
}

// searchText is everything a query is matched against.
func (i Item) searchText() string {
	return strings.ToLower(i.Title + " " + i.Badge + " " + i.Detail)
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
}

func newPicker(heading string, items []Item) *picker {
	p := &picker{heading: heading, items: items, chosen: -1}
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
	for i, item := range p.items {
		if q == "" || strings.Contains(item.searchText(), q) {
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

// Update handles keys. Printable runes always go to the query, so a search for
// "prod-k8s" is not intercepted by navigation shortcuts.
func (p *picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch key.Type {
	case tea.KeyCtrlC:
		return p, tea.Quit
	case tea.KeyEsc:
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
	case tea.KeyRunes, tea.KeySpace:
		p.query += string(key.Runes)
		if key.Type == tea.KeySpace {
			p.query += " "
		}
		p.filter()
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

func (p *picker) View() string {
	if p.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(ui.Accent.Render(p.heading) + "\n")

	if p.query == "" {
		b.WriteString(ui.Muted.Render("type to search · ↑↓ move · enter select · esc cancel") + "\n\n")
	} else {
		b.WriteString(ui.Muted.Render("search: ") + ui.Warned.Render(p.query) + "\n\n")
	}

	if len(p.filtered) == 0 {
		b.WriteString(ui.Bad.Render("no matches") + "\n")
		return b.String()
	}

	if p.offset > 0 {
		b.WriteString(ui.Muted.Render(fmt.Sprintf("  ↑ %d more", p.offset)) + "\n")
	}

	end := min(p.offset+visibleRows, len(p.filtered))
	for n := p.offset; n < end; n++ {
		item := p.items[p.filtered[n]]
		style := ui.Muted
		cursor := "  "
		switch {
		case n == p.cursor:
			style = ui.Good.Bold(true)
			cursor = ui.Accent.Render("❯ ")
		case item.Marked:
			style = ui.Warned
		}

		row := cursor + style.Render(item.Title)
		if item.Badge != "" {
			row += " " + ui.Muted.Render("("+item.Badge+")")
		}
		if item.Marked {
			row += " " + ui.Warned.Render("• active")
		}
		if item.Detail != "" {
			row += "  " + ui.Muted.Render(item.Detail)
		}
		b.WriteString(row + "\n")
	}

	if end < len(p.filtered) {
		b.WriteString(ui.Muted.Render(fmt.Sprintf("  ↓ %d more", len(p.filtered)-end)) + "\n")
	}

	b.WriteString("\n" + ui.Muted.Render(fmt.Sprintf("%d of %d", len(p.filtered), len(p.items))) + "\n")
	return b.String()
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
