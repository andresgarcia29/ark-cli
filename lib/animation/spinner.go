package animation

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel representa el modelo para el spinner
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
	done     bool
}

// NewSpinnerModel crea un nuevo modelo de spinner
func NewSpinnerModel(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return SpinnerModel{
		spinner: s,
		message: message,
	}
}

// Init implementa tea.Model
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implementa tea.Model
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

	default:
		return m, nil
	}
}

// View implementa tea.Model
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

// doneMsg es un mensaje para indicar que el spinner debe terminar
type doneMsg struct{}

// Done retorna un comando que envía un mensaje de finalización
func Done() tea.Msg {
	return doneMsg{}
}

// ShowSpinner muestra un spinner mientras ejecuta una función
func ShowSpinner(message string, fn func() error) error {
	p := tea.NewProgram(NewSpinnerModel(message))

	// Canal para manejar el resultado de la función
	errChan := make(chan error, 1)

	// Ejecutar la función en una goroutine
	go func() {
		err := fn()
		errChan <- err
		// Enviar mensaje de finalización al programa
		time.Sleep(100 * time.Millisecond) // Pequeña pausa para que se vea el spinner
		p.Send(Done())
	}()

	// Ejecutar el programa (esto bloqueará hasta que termine)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running spinner: %w", err)
	}

	// Obtener el resultado de la función
	return <-errChan
}
