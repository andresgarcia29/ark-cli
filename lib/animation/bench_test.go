package animation

import (
	"fmt"
	"testing"
)

func benchItems(n int) []Item {
	items := make([]Item, n)
	for i := range items {
		items[i] = Item{
			Title:  fmt.Sprintf("tml-service-%04d-stg-aws-operations-access", i),
			Badge:  "sso",
			Detail: fmt.Sprintf("account %012d · aws-operations-access", i),
		}
	}
	return items
}

// Filtering runs on every keystroke, so it must stay cheap even for accounts
// with hundreds of profiles.
func BenchmarkFilter(b *testing.B) {
	p := newPicker("pick", benchItems(1000))
	p.query = "service-0500"

	b.ReportAllocs()
	for b.Loop() {
		p.filter()
	}
}

// View is re-rendered on every frame while the picker is open.
func BenchmarkView(b *testing.B) {
	p := newPicker("pick", benchItems(1000))
	p.query = "operations"

	b.ReportAllocs()
	for b.Loop() {
		_ = p.View()
	}
}
