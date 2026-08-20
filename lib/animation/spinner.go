package animation

import (
	"context"
	"fmt"

	"github.com/andresgarcia29/ark-cli/lib/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinnerModel struct {
	spinner spinner.Model
	message string
	note    string
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
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
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

func (m spinnerModel) View() string {
	out := m.spinner.View() + " " + m.message
	if m.note != "" {
		out += "\n  " + ui.Muted.Render(m.note)
	}
	return out + "\n"
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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.Accent

	p := tea.NewProgram(
		spinnerModel{spinner: s, message: message},
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
