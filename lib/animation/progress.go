package animation

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"github.com/andresgarcia29/ark-cli/lib/ui"
)

// Outcome is the result of processing one item in a batch.
type Outcome struct {
	Item string
	Err  error
}

type progressModel struct {
	bar     progress.Model
	title   string
	total   int
	done    int
	current string
	failed  []Outcome
	width   int
}

type stepMsg Outcome

func (m progressModel) Init() tea.Cmd { return nil }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stepMsg:
		m.done++
		m.current = msg.Item
		if msg.Err != nil {
			m.failed = append(m.failed, Outcome(msg))
		}
		if m.done >= m.total {
			return m, tea.Quit
		}
		return m, nil
	case tea.KeyPressMsg:
		if k := msg.Key(); k.Mod&tea.ModCtrl != 0 && (k.Code == 'c' || k.Code == 'd') {
			return m, tea.Quit
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.bar.SetWidth(min(msg.Width-4, 48))
		return m, nil
	}
	return m, nil
}

func (m progressModel) View() tea.View {
	var b strings.Builder
	b.WriteString(ui.Title.Render(m.title))
	b.WriteString("\n\n")
	b.WriteString(m.bar.ViewAs(float64(m.done) / float64(m.total)))
	b.WriteString("  ")
	b.WriteString(ui.Muted.Render(fmt.Sprintf("%d/%d", m.done, m.total)))
	if len(m.failed) > 0 {
		b.WriteString(ui.Bad.Render(fmt.Sprintf("  %s %d failed", ui.GlyphDot, len(m.failed))))
	}
	b.WriteString("\n")
	if m.current != "" && m.done < m.total {
		b.WriteString("  ")
		b.WriteString(ui.Faint.Render(truncate(m.current, m.width-4)))
		b.WriteString("\n")
	}
	return tea.NewView(b.String())
}

// RunBatch applies work to every item, showing a progress bar, and returns the
// outcome of each. Work runs sequentially; use it for steps that share a file.
func RunBatch(ctx context.Context, title string, items []string, work func(context.Context, string) error) []Outcome {
	outcomes := make([]Outcome, 0, len(items))

	if !ui.Interactive() {
		ui.Step("%s (%d)", title, len(items))
		for _, item := range items {
			outcomes = append(outcomes, Outcome{Item: item, Err: work(ctx, item)})
		}
		return outcomes
	}

	bar := progress.New(
		progress.WithColors(ui.ProgressColors()...),
		progress.WithWidth(48),
		progress.WithoutPercentage(),
	)

	p := tea.NewProgram(
		progressModel{bar: bar, title: title, total: len(items), width: ui.Width()},
		tea.WithOutput(ui.Err),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan []Outcome, 1)
	go func() {
		out := make([]Outcome, 0, len(items))
		for _, item := range items {
			if ctx.Err() != nil {
				break
			}
			o := Outcome{Item: item, Err: work(ctx, item)}
			out = append(out, o)
			p.Send(stepMsg(o))
		}
		resultCh <- out
		p.Quit()
	}()

	_, _ = p.Run()
	cancel()
	return <-resultCh
}
