package animation

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel represents the spinner model
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
	done     bool
}

// NewSpinnerModel creates a new spinner model
func NewSpinnerModel(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return SpinnerModel{
		spinner: s,
		message: message,
	}
}

// Init implements tea.Model
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case tea.QuitMsg:
		m.quitting = true
		return m, nil

	default:
		return m, nil
	}
}

// View implements tea.Model
func (m SpinnerModel) View() string {
	if m.done {
		checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
		return checkStyle.Render(fmt.Sprintf("✓ %s\n", m.message))
	}

	if m.quitting {
		return ""
	}

	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return fmt.Sprintf("%s %s\n", m.spinner.View(), messageStyle.Render(m.message))
}

// doneMsg is a message to indicate that the spinner should terminate
type doneMsg struct{}

// Done returns a command that sends a completion message
func Done() tea.Msg {
	return doneMsg{}
}

// ShowSpinner shows a spinner while executing a function
func ShowSpinner(message string, fn func() error) error {
	p := tea.NewProgram(NewSpinnerModel(message))

	// Channel to handle the function result
	errChan := make(chan error, 1)

	// Execute the function in a goroutine
	go func() {
		err := fn()
		errChan <- err
		// Send completion message to the program
		time.Sleep(100 * time.Millisecond) // Small pause for the spinner to be visible
		p.Send(Done())
	}()

	// Run the program (this will block until it finishes)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running spinner: %w", err)
	}

	// Get the function result
	return <-errChan
}
