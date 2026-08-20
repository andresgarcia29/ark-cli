package animation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/andresgarcia29/ark-cli/lib/ui"
)

type spinnerModel struct {
	spinner spinner.Model
	message string
	note    string
	started time.Time
	width   int
}

// noteMsg replaces the line under the spinner with fresh progress detail.
type noteMsg string

// finishedMsg ends the spinner once the work returns.
type finishedMsg struct{}

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case finishedMsg:
		return m, tea.Quit
	case noteMsg:
		m.note = string(msg)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		if k := msg.Key(); k.Mod&tea.ModCtrl != 0 && (k.Code == 'c' || k.Code == 'd') {
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.spinner.View())
	b.WriteString(" ")
	b.WriteString(m.message)

	// Elapsed time makes a slow scan legible as progress rather than a hang.
	if elapsed := time.Since(m.started); elapsed > 3*time.Second {
		b.WriteString(" ")
		b.WriteString(ui.Faint.Render(fmt.Sprintf("%s %s", ui.GlyphDot, elapsed.Round(time.Second))))
	}
	if m.note != "" {
		b.WriteString("\n  ")
		b.WriteString(ui.Muted.Render(truncate(m.note, m.width-4)))
	}
	b.WriteString("\n")
	return tea.NewView(b.String())
}

// truncate shortens s to fit width, so a long note never wraps and smears the
// spinner across several lines.
func truncate(s string, width int) string {
	if width <= 1 || len(s) <= width {
		return s
	}
	return s[:width-1] + "…"
}

// Progress reports incremental detail from inside a spinner.
type Progress func(format string, a ...any)

// Spin runs work while showing a spinner, cancelling it if the user hits
// ctrl+c. When stderr is not a terminal it prints one line and runs work
// directly, so logs and CI output stay clean.
func Spin(ctx context.Context, message string, work func(context.Context, Progress) error) error {
	if !ui.Interactive() {
		ui.Step("%s", message)
		return work(ctx, func(string, ...any) {})
	}

	s := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(ui.Accent),
	)

	p := tea.NewProgram(
		spinnerModel{spinner: s, message: message, started: time.Now(), width: ui.Width()},
		tea.WithOutput(ui.Err),
		tea.WithContext(ctx),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- work(ctx, func(format string, a ...any) {
			p.Send(noteMsg(fmt.Sprintf(format, a...)))
		})
		p.Send(finishedMsg{})
	}()

	if _, err := p.Run(); err != nil {
		// The UI stopped, most likely ctrl+c. Cancel the work and report it.
		cancel()
		<-errCh
		return fmt.Errorf("cancelled: %w", err)
	}

	if err := <-errCh; err != nil {
		return err
	}
	ui.Done("%s", message)
	return nil
}
