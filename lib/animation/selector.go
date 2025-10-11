package animation

import (
	"fmt"
	"strings"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProfileDisplayInfo contiene información para mostrar en la lista interactiva
type ProfileDisplayInfo struct {
	Name        string
	Type        string
	Description string
	AccountID   string
	RoleName    string
	Region      string
}

// profileSelectorModel representa el modelo para el selector de perfiles con Bubble Tea
type profileSelectorModel struct {
	profiles         []services_aws.ProfileConfig
	filteredProfiles []services_aws.ProfileConfig
	cursor           int
	offset           int // Índice del primer perfil visible
	visibleLines     int // Número máximo de perfiles a mostrar
	searchQuery      string
	selected         *services_aws.ProfileConfig
	quitting         bool
	searchMode       bool
}

// initialProfileSelectorModel crea el modelo inicial para el selector
func initialProfileSelectorModel(profiles []services_aws.ProfileConfig) profileSelectorModel {
	return profileSelectorModel{
		profiles:         profiles,
		filteredProfiles: profiles,
		cursor:           0,
		offset:           0,
		visibleLines:     10, // Mostrar máximo 10 perfiles
		searchQuery:      "",
		searchMode:       true, // Iniciar en modo búsqueda
	}
}

// Init implementa el método Init de tea.Model
func (m profileSelectorModel) Init() tea.Cmd {
	return nil
}

// Update implementa el método Update de tea.Model
func (m profileSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "/":
			// Activar modo búsqueda
			m.searchMode = true
			m.searchQuery = ""
			return m, nil

		case "esc":
			if m.searchMode {
				// Salir del modo búsqueda
				m.searchMode = false
				m.searchQuery = ""
				m.filteredProfiles = m.profiles
				m.cursor = 0
				m.offset = 0
			} else {
				m.quitting = true
			}
			return m, tea.Quit

		case "tab":
			// Alternar entre modo búsqueda y vista completa
			if m.searchMode {
				m.searchMode = false
				m.searchQuery = ""
				m.filteredProfiles = m.profiles
				m.cursor = 0
				m.offset = 0
			} else {
				m.searchMode = true
				m.searchQuery = ""
				m.cursor = 0
				m.offset = 0
			}
			return m, nil

		case "enter":
			if m.searchMode && len(m.filteredProfiles) > 0 {
				// Si hay resultados, seleccionar el primero
				m.selected = &m.filteredProfiles[m.cursor]
				return m, tea.Quit
			} else if !m.searchMode && len(m.filteredProfiles) > 0 {
				// Seleccionar perfil
				m.selected = &m.filteredProfiles[m.cursor]
				return m, tea.Quit
			}
			return m, nil

		case "backspace":
			if m.searchMode && len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				m.filterProfiles()
			}
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Ajustar offset para mantener el cursor visible
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.filteredProfiles)-1 {
				m.cursor++
				// Ajustar offset para mantener el cursor visible
				currentVisibleLines := m.getCurrentVisibleLines()
				if m.cursor >= m.offset+currentVisibleLines {
					m.offset = m.cursor - currentVisibleLines + 1
				}
			}

		default:
			// Si estamos en modo búsqueda, agregar caracteres
			if m.searchMode && len(msg.String()) == 1 {
				m.searchQuery += msg.String()
				m.filterProfiles()
			}
		}
	}

	return m, nil
}

// min retorna el mínimo de dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCurrentVisibleLines calcula cuántas líneas mostrar actualmente
func (m profileSelectorModel) getCurrentVisibleLines() int {
	// Siempre limitar a máximo 10 resultados
	return min(m.visibleLines, len(m.filteredProfiles))
}

// filterProfiles filtra los perfiles basado en la consulta de búsqueda
func (m *profileSelectorModel) filterProfiles() {
	if m.searchQuery == "" {
		m.filteredProfiles = m.profiles
		return
	}

	filtered := make([]services_aws.ProfileConfig, 0)
	query := strings.ToLower(m.searchQuery)

	for _, profile := range m.profiles {
		// Buscar en nombre del perfil
		if strings.Contains(strings.ToLower(profile.ProfileName), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Buscar en account ID
		if strings.Contains(strings.ToLower(profile.AccountID), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Buscar en role name
		if strings.Contains(strings.ToLower(profile.RoleName), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Buscar en role ARN
		if strings.Contains(strings.ToLower(profile.RoleARN), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Buscar en source profile
		if strings.Contains(strings.ToLower(profile.SourceProfile), query) {
			filtered = append(filtered, profile)
			continue
		}
	}

	m.filteredProfiles = filtered
	// Resetear cursor y offset cuando cambian los perfiles filtrados
	m.cursor = 0
	m.offset = 0
}

// View implementa el método View de tea.Model
func (m profileSelectorModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)
	s.WriteString(headerStyle.Render("🔍 Select an AWS profile to login:"))
	s.WriteString("\n\n")

	// Search bar
	if m.searchMode {
		searchStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
		s.WriteString(searchStyle.Render("🔎 Search: "))

		queryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
		s.WriteString(queryStyle.Render(m.searchQuery))
		s.WriteString("_") // Cursor
		s.WriteString("\n\n")
	}

	// Instructions
	instructionsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	var instructions string
	if m.searchMode {
		instructions = "Type to search • Enter to select • Tab to view all • Esc to quit"
	} else {
		instructions = "↑/↓ to navigate • / to search • Enter to select • q/esc to quit"
	}

	s.WriteString(instructionsStyle.Render(instructions))
	s.WriteString("\n\n")

	// Results count
	if m.searchQuery != "" {
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(countStyle.Render(fmt.Sprintf("Found %d of %d profiles", len(m.filteredProfiles), len(m.profiles))))
		s.WriteString("\n\n")
	} else if len(m.filteredProfiles) > m.visibleLines {
		// Mostrar indicador de scroll cuando hay más perfiles
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(countStyle.Render(fmt.Sprintf("Showing %d of %d profiles (use ↑/↓ to scroll)", m.getCurrentVisibleLines(), len(m.filteredProfiles))))
		s.WriteString("\n\n")
	}

	// Profile list
	if len(m.filteredProfiles) == 0 {
		noResultsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		s.WriteString(noResultsStyle.Render("No profiles found matching your search"))
		s.WriteString("\n")
		return s.String()
	}

	// Calcular ventana de visualización
	currentVisibleLines := m.getCurrentVisibleLines()
	startDisplay := m.offset
	endDisplay := min(m.offset+currentVisibleLines, len(m.filteredProfiles))

	// Mostrar indicador si hay perfiles arriba
	if m.offset > 0 {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more profiles above)"))
		s.WriteString("\n")
	}

	// Renderizar perfiles en la ventana visible
	for i := startDisplay; i < endDisplay; i++ {
		profile := m.filteredProfiles[i]
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		displayInfo := formatProfileDisplay(profile)

		// Style based on profile type
		var nameStyle lipgloss.Style
		var typeStyle lipgloss.Style

		switch profile.ProfileType {
		case services_aws.ProfileTypeSSO:
			nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
			typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
		case services_aws.ProfileTypeAssumeRole:
			nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
			typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		default:
			nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		}

		// Highlight selected item
		if m.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(">")
			nameStyle = nameStyle.Bold(true)
		}

		line := fmt.Sprintf("%s %s (%s) - %s",
			cursor,
			nameStyle.Render(displayInfo.Name),
			typeStyle.Render(displayInfo.Type),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(displayInfo.Description),
		)

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Mostrar indicador si hay perfiles abajo
	if endDisplay < len(m.filteredProfiles) {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more profiles below)"))
		s.WriteString("\n")
	}

	return s.String()
}

// formatProfileDisplay formatea la información del perfil para mostrar
func formatProfileDisplay(profile services_aws.ProfileConfig) ProfileDisplayInfo {
	var description string
	var accountID, roleName string

	switch profile.ProfileType {
	case services_aws.ProfileTypeSSO:
		accountID = profile.AccountID
		roleName = profile.RoleName
		description = fmt.Sprintf("SSO - Account: %s, Role: %s", accountID, roleName)
	case services_aws.ProfileTypeAssumeRole:
		// Extraer account ID del ARN
		if strings.Contains(profile.RoleARN, ":") {
			parts := strings.Split(profile.RoleARN, ":")
			if len(parts) >= 5 {
				accountID = parts[4]
			}
		}
		// Extraer role name del ARN
		if strings.Contains(profile.RoleARN, "/") {
			parts := strings.Split(profile.RoleARN, "/")
			if len(parts) >= 2 {
				roleName = parts[1]
			}
		}
		description = fmt.Sprintf("Assume Role - Account: %s, Role: %s", accountID, roleName)
	default:
		description = "Unknown profile type"
	}

	return ProfileDisplayInfo{
		Name:        profile.ProfileName,
		Type:        string(profile.ProfileType),
		Description: description,
		AccountID:   accountID,
		RoleName:    roleName,
		Region:      profile.Region,
	}
}

// InteractiveProfileSelector permite seleccionar un perfil de forma interactiva usando Bubble Tea
func InteractiveProfileSelector() (*services_aws.ProfileConfig, error) {
	// Obtener todos los perfiles
	profiles, err := services_aws.ReadAllProfilesFromConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles: %w", err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found in AWS config")
	}

	// Crear y ejecutar el programa Bubble Tea
	model := initialProfileSelectorModel(profiles)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("error running profile selector: %w", err)
	}

	// Verificar si se seleccionó un perfil
	if finalModel.(profileSelectorModel).selected == nil {
		return nil, fmt.Errorf("no profile selected")
	}

	return finalModel.(profileSelectorModel).selected, nil
}
