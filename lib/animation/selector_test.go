package animation

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func items() []Item {
	return []Item{
		{Title: "prod-k8s-readonly", Badge: "sso"},
		{Title: "staging-jump-admin", Badge: "sso"},
		{Title: "dev-query-role", Badge: "assume_role"},
	}
}

func typeRunes(p *picker, s string) *picker {
	for _, r := range s {
		m, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		p = m.(*picker)
	}
	return p
}

// The old selector matched "q", "k" and "j" as shortcuts before treating input
// as text, so these characters could never be typed into a search.
func TestSearchAcceptsNavigationLetters(t *testing.T) {
	for _, query := range []string{"k8s", "jump", "query", "q"} {
		p := typeRunes(newPicker("pick", items()), query)
		if p.query != query {
			t.Errorf("typing %q produced query %q", query, p.query)
		}
		if len(p.filtered) == 0 {
			t.Errorf("query %q matched nothing", query)
		}
	}
}

func TestSearchFiltersAndSelects(t *testing.T) {
	p := typeRunes(newPicker("pick", items()), "k8s")
	if len(p.filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(p.filtered))
	}

	m, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := m.(*picker)
	if got.chosen < 0 {
		t.Fatal("enter did not select anything")
	}
	if name := items()[got.chosen].Title; name != "prod-k8s-readonly" {
		t.Errorf("selected %q, want prod-k8s-readonly", name)
	}
}

// esc used to quit the whole picker even when it was only meant to clear the
// active search.
func TestEscClearsSearchBeforeQuitting(t *testing.T) {
	p := typeRunes(newPicker("pick", items()), "k8s")

	m, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	p = m.(*picker)
	if p.query != "" {
		t.Errorf("esc left query %q", p.query)
	}
	if cmd != nil {
		t.Error("esc quit the picker instead of clearing the search")
	}
	if len(p.filtered) != len(items()) {
		t.Errorf("filter not reset: %d of %d", len(p.filtered), len(items()))
	}

	// A second esc, with no search active, should quit.
	if _, cmd = p.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Error("esc on an empty search should quit")
	}
}

func TestCancellingReportsNoSelection(t *testing.T) {
	p := newPicker("pick", items())
	m, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should quit")
	}
	if m.(*picker).chosen >= 0 {
		t.Error("ctrl+c must not select anything")
	}
}

func TestCursorStartsOnActiveItem(t *testing.T) {
	list := items()
	list[2].Marked = true

	p := newPicker("pick", list)
	if p.cursor != 2 {
		t.Errorf("cursor = %d, want it on the active item (2)", p.cursor)
	}
}

func TestCursorStaysInRange(t *testing.T) {
	p := newPicker("pick", items())

	p.move(-5)
	if p.cursor != 0 {
		t.Errorf("cursor = %d after moving up past the top", p.cursor)
	}
	p.move(50)
	if want := len(items()) - 1; p.cursor != want {
		t.Errorf("cursor = %d, want %d", p.cursor, want)
	}

	// Narrowing the list must pull the cursor back into bounds.
	p = typeRunes(p, "k8s")
	if p.cursor >= len(p.filtered) {
		t.Errorf("cursor %d out of range for %d results", p.cursor, len(p.filtered))
	}
}

func TestBackspaceEditsQuery(t *testing.T) {
	p := typeRunes(newPicker("pick", items()), "k8sx")
	m, _ := p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	p = m.(*picker)

	if p.query != "k8s" {
		t.Errorf("query = %q, want k8s", p.query)
	}
	if len(p.filtered) != 1 {
		t.Errorf("filtered = %d, want 1", len(p.filtered))
	}
}
