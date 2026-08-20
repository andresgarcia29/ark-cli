package animation

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func items() []Item {
	return []Item{
		{Title: "prod-k8s-readonly", Badge: "sso"},
		{Title: "staging-jump-admin", Badge: "sso"},
		{Title: "dev-query-role", Badge: "assume_role"},
	}
}

// press sends a special key, e.g. tea.KeyEnter.
func press(p *picker, code rune) (*picker, tea.Cmd) {
	m, cmd := p.Update(tea.KeyPressMsg{Code: code})
	return m.(*picker), cmd
}

// typeRunes types printable text the way a terminal reports it in v2: Key.Text
// carries the character, which is what keeps letters out of the shortcut table.
func typeRunes(p *picker, s string) *picker {
	for _, r := range s {
		m, _ := p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
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

	got, _ := press(p, tea.KeyEnter)
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

	p, cmd := press(p, tea.KeyEscape)
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
	if _, cmd = press(p, tea.KeyEscape); cmd == nil {
		t.Error("esc on an empty search should quit")
	}
}

func TestCancellingReportsNoSelection(t *testing.T) {
	p := newPicker("pick", items())
	m, cmd := p.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
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
	p, _ = press(p, tea.KeyBackspace)

	if p.query != "k8s" {
		t.Errorf("query = %q, want k8s", p.query)
	}
	if len(p.filtered) != 1 {
		t.Errorf("filtered = %d, want 1", len(p.filtered))
	}
}

// highlight slices the original title using offsets found in its lowercased
// form. Those can disagree for multi-byte text, so it must never cut a rune.
func TestHighlightHandlesUnicode(t *testing.T) {
	cases := []struct {
		title, query string
	}{
		{"café-prod-readonly", "prod"},
		{"CAFÉ-PROD", "café"},
		{"münchen-cluster", "münchen"},
		{"ÅNGSTRÖM-eks", "ström"},
		{"日本-クラスタ", "クラスタ"},
		{"plain", ""},
		{"plain", "zzz"},
	}

	for _, c := range cases {
		item := Item{Title: c.title}
		item.prepare()

		got := highlight(item, c.query, lipgloss.NewStyle())
		if !utf8.ValidString(got) {
			t.Errorf("highlight(%q, %q) produced invalid UTF-8: %q", c.title, c.query, got)
		}
		// Stripping styling must give the original title back untouched.
		if plain := stripANSI(got); plain != c.title {
			t.Errorf("highlight(%q, %q) altered the text: %q", c.title, c.query, plain)
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestFilterMatchesBadgeAndDetail(t *testing.T) {
	list := []Item{
		{Title: "alpha", Badge: "sso", Detail: "account 111122223333"},
		{Title: "beta", Badge: "assume_role", Detail: "account 999988887777"},
	}
	p := newPicker("pick", list)

	for _, q := range []string{"assume", "9999", "beta"} {
		np := typeRunes(newPicker("pick", list), q)
		if len(np.filtered) != 1 {
			t.Errorf("query %q matched %d rows, want 1", q, len(np.filtered))
		}
	}
	_ = p
}
