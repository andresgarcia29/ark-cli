package animation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressModel representa el modelo para la barra de progreso
type ProgressModel struct {
	progress     progress.Model
	total        int
	current      int
	currentItem  string
	items        []string
	errors       []string
	quitting     bool
	done         bool
	successCount int
}

// progressMsg es un mensaje para actualizar el progreso
type progressMsg struct {
	item  string
	error string
}

// NewProgressModel crea un nuevo modelo de barra de progreso
func NewProgressModel(total int) ProgressModel {
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	return ProgressModel{
		progress: prog,
		total:    total,
		current:  0,
		items:    make([]string, 0),
		errors:   make([]string, 0),
	}
}

// Init implementa tea.Model
func (m ProgressModel) Init() tea.Cmd {
	return nil
}

// Update implementa tea.Model
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case progressMsg:
		m.current++
		m.currentItem = msg.item
		if msg.error != "" {
			m.errors = append(m.errors, msg.error)
		} else {
			m.successCount++
		}
		m.items = append(m.items, msg.item)

		if m.current >= m.total {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 4
		if m.progress.Width > 120 {
			m.progress.Width = 120
		}
		return m, nil

	default:
		return m, nil
	}
}

// View implementa tea.Model
func (m ProgressModel) View() string {
	if m.quitting && !m.done {
		return ""
	}

	var s strings.Builder

	// Título
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)

	if m.done {
		s.WriteString(titleStyle.Render("🎉 Configuration Completed!"))
		s.WriteString("\n\n")
	} else {
		s.WriteString(titleStyle.Render("⚙️  Configuring Kubernetes Clusters"))
		s.WriteString("\n\n")
	}

	// Barra de progreso
	percent := float64(m.current) / float64(m.total)
	s.WriteString(m.progress.ViewAs(percent))
	s.WriteString("\n\n")

	// Contador
	counterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	s.WriteString(counterStyle.Render(fmt.Sprintf("Progress: %d/%d clusters", m.current, m.total)))
	s.WriteString("\n\n")

	// Item actual
	if !m.done && m.currentItem != "" {
		currentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
		s.WriteString(currentStyle.Render(fmt.Sprintf("⚡ Configuring: %s", m.currentItem)))
		s.WriteString("\n\n")
	}

	// Resumen final
	if m.done {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
		failStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

		s.WriteString(successStyle.Render(fmt.Sprintf("✓ Successful: %d", m.successCount)))
		s.WriteString("\n")

		if len(m.errors) > 0 {
			s.WriteString(failStyle.Render(fmt.Sprintf("✗ Failed: %d", len(m.errors))))
			s.WriteString("\n\n")

			// Mostrar errores
			errorHeaderStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
			s.WriteString(errorHeaderStyle.Render("Errors:"))
			s.WriteString("\n")

			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)

			for _, err := range m.errors {
				s.WriteString(errorStyle.Render(fmt.Sprintf("  • %s", err)))
				s.WriteString("\n")
			}
		}
	}

	return s.String()
}

// ProgressIncrement retorna un comando para incrementar el progreso
func ProgressIncrement(item string, err error) tea.Cmd {
	return func() tea.Msg {
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		return progressMsg{
			item:  item,
			error: errorMsg,
		}
	}
}

// ShowProgressBar muestra una barra de progreso para múltiples operaciones
func ShowProgressBar(total int, fn func(update func(item string, err error)) error) error {
	model := NewProgressModel(total)
	p := tea.NewProgram(model)

	// Canal para errores
	errChan := make(chan error, 1)

	// Función para actualizar el progreso
	updateProgress := func(item string, err error) {
		p.Send(progressMsg{
			item: item,
			error: func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(),
		})
	}

	// Ejecutar la función en una goroutine
	go func() {
		err := fn(updateProgress)
		errChan <- err
	}()

	// Ejecutar el programa (esto bloqueará hasta que termine)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running progress bar: %w", err)
	}

	// Obtener el resultado de la función
	return <-errChan
}
