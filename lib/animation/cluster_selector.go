package animation

import (
	"fmt"
	"strings"

	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ClusterDisplayInfo contiene información para mostrar en la lista interactiva
type ClusterDisplayInfo struct {
	Name        string
	Current     bool
	Status      string
	Profile     string
	Region      string
	ClusterName string
}

// clusterSelectorModel representa el modelo para el selector de clusters con Bubble Tea
type clusterSelectorModel struct {
	clusters         []services_kubernetes.ClusterContext
	filteredClusters []services_kubernetes.ClusterContext
	cursor           int
	offset           int // Índice del primer cluster visible
	visibleLines     int // Número máximo de clusters a mostrar
	searchQuery      string
	selected         *services_kubernetes.ClusterContext
	quitting         bool
	searchMode       bool
}

// initialClusterSelectorModel crea el modelo inicial para el selector
func initialClusterSelectorModel(clusters []services_kubernetes.ClusterContext) clusterSelectorModel {
	return clusterSelectorModel{
		clusters:         clusters,
		filteredClusters: clusters,
		cursor:           0,
		offset:           0,
		visibleLines:     10, // Mostrar máximo 10 clusters
		searchQuery:      "",
		searchMode:       true, // Iniciar en modo búsqueda
	}
}

// Init implementa el método Init de tea.Model
func (m clusterSelectorModel) Init() tea.Cmd {
	return nil
}

// Update implementa el método Update de tea.Model
func (m clusterSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.filteredClusters = m.clusters
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
				m.filteredClusters = m.clusters
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
			if m.searchMode && len(m.filteredClusters) > 0 {
				// Si hay resultados, seleccionar el primero
				m.selected = &m.filteredClusters[m.cursor]
				return m, tea.Quit
			} else if !m.searchMode && len(m.filteredClusters) > 0 {
				// Seleccionar cluster
				m.selected = &m.filteredClusters[m.cursor]
				return m, tea.Quit
			}
			return m, nil

		case "backspace":
			if m.searchMode && len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				m.filterClusters()
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
			if m.cursor < len(m.filteredClusters)-1 {
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
				m.filterClusters()
			}
		}
	}

	return m, nil
}

// getCurrentVisibleLines calcula cuántas líneas mostrar actualmente
func (m clusterSelectorModel) getCurrentVisibleLines() int {
	// Siempre limitar a máximo 10 resultados
	return min(m.visibleLines, len(m.filteredClusters))
}

// filterClusters filtra los clusters basado en la consulta de búsqueda
func (m *clusterSelectorModel) filterClusters() {
	if m.searchQuery == "" {
		m.filteredClusters = m.clusters
		return
	}

	filtered := make([]services_kubernetes.ClusterContext, 0)
	query := strings.ToLower(m.searchQuery)

	for _, cluster := range m.clusters {
		// Buscar en nombre del cluster
		if strings.Contains(strings.ToLower(cluster.Name), query) {
			filtered = append(filtered, cluster)
		}
	}

	m.filteredClusters = filtered
	// Resetear cursor y offset cuando cambian los clusters filtrados
	m.cursor = 0
	m.offset = 0
}

// View implementa el método View de tea.Model
func (m clusterSelectorModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)
	s.WriteString(headerStyle.Render("🔍 Select a Kubernetes cluster context:"))
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
		s.WriteString(countStyle.Render(fmt.Sprintf("Found %d of %d clusters", len(m.filteredClusters), len(m.clusters))))
		s.WriteString("\n\n")
	} else if len(m.filteredClusters) > m.visibleLines {
		// Mostrar indicador de scroll cuando hay más clusters
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(countStyle.Render(fmt.Sprintf("Showing %d of %d clusters (use ↑/↓ to scroll)", m.getCurrentVisibleLines(), len(m.filteredClusters))))
		s.WriteString("\n\n")
	}

	// Cluster list
	if len(m.filteredClusters) == 0 {
		noResultsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		s.WriteString(noResultsStyle.Render("No clusters found matching your search"))
		s.WriteString("\n")
		return s.String()
	}

	// Calcular ventana de visualización
	currentVisibleLines := m.getCurrentVisibleLines()
	startDisplay := m.offset
	endDisplay := min(m.offset+currentVisibleLines, len(m.filteredClusters))

	// Mostrar indicador si hay clusters arriba
	if m.offset > 0 {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more clusters above)"))
		s.WriteString("\n")
	}

	// Renderizar clusters en la ventana visible
	for i := startDisplay; i < endDisplay; i++ {
		cluster := m.filteredClusters[i]
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		displayInfo := formatClusterDisplay(cluster)

		// Style based on cluster status
		var nameStyle lipgloss.Style
		var statusStyle lipgloss.Style

		if cluster.Current {
			nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
		} else {
			nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		}

		// Highlight selected item
		if m.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(">")
			nameStyle = nameStyle.Bold(true)
		}

		// Build description with profile and region info
		description := ""
		if displayInfo.Profile != "" {
			description = fmt.Sprintf("Profile: %s", displayInfo.Profile)
		}
		if displayInfo.Region != "" {
			if description != "" {
				description += ", "
			}
			description += fmt.Sprintf("Region: %s", displayInfo.Region)
		}
		if displayInfo.ClusterName != "" {
			if description != "" {
				description += ", "
			}
			description += fmt.Sprintf("Cluster: %s", displayInfo.ClusterName)
		}

		line := fmt.Sprintf("%s %s %s",
			cursor,
			nameStyle.Render(displayInfo.Name),
			statusStyle.Render(displayInfo.Status),
		)

		if description != "" {
			line += fmt.Sprintf(" - %s", lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(description))
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Mostrar indicador si hay clusters abajo
	if endDisplay < len(m.filteredClusters) {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more clusters below)"))
		s.WriteString("\n")
	}

	return s.String()
}

// formatClusterDisplay formatea la información del cluster para mostrar
func formatClusterDisplay(cluster services_kubernetes.ClusterContext) ClusterDisplayInfo {
	status := ""
	if cluster.Current {
		status = "(current)"
	}

	return ClusterDisplayInfo{
		Name:        cluster.Name,
		Current:     cluster.Current,
		Status:      status,
		Profile:     cluster.Profile,
		Region:      cluster.Region,
		ClusterName: cluster.ClusterName,
	}
}

// InteractiveClusterSelector permite seleccionar un cluster de forma interactiva usando Bubble Tea
func InteractiveClusterSelector() (*services_kubernetes.ClusterContext, error) {
	// Obtener todos los clusters
	clusters, err := services_kubernetes.GetClusterContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster contexts: %w", err)
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no cluster contexts found in kubeconfig")
	}

	// Crear y ejecutar el programa Bubble Tea
	model := initialClusterSelectorModel(clusters)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("error running cluster selector: %w", err)
	}

	// Verificar si se seleccionó un cluster
	if finalModel.(clusterSelectorModel).selected == nil {
		return nil, fmt.Errorf("no cluster selected")
	}

	return finalModel.(clusterSelectorModel).selected, nil
}
