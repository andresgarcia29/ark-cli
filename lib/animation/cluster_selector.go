package animation

import (
	"fmt"
	"strings"

	services_kubernetes "github.com/andresgarcia29/ark-cli/services/kubernetes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ClusterDisplayInfo contains information to show in the interactive list
type ClusterDisplayInfo struct {
	Name        string
	Current     bool
	Status      string
	Profile     string
	Region      string
	ClusterName string
}

// clusterSelectorModel represents the model for the cluster selector with Bubble Tea
type clusterSelectorModel struct {
	clusters         []services_kubernetes.ClusterContext
	filteredClusters []services_kubernetes.ClusterContext
	cursor           int
	offset           int // Index of the first visible cluster
	visibleLines     int // Maximum number of clusters to show
	searchQuery      string
	selected         *services_kubernetes.ClusterContext
	quitting         bool
	searchMode       bool
}

// initialClusterSelectorModel creates the initial model for the selector
func initialClusterSelectorModel(clusters []services_kubernetes.ClusterContext) clusterSelectorModel {
	return clusterSelectorModel{
		clusters:         clusters,
		filteredClusters: clusters,
		cursor:           0,
		offset:           0,
		visibleLines:     10, // Show maximum 10 clusters
		searchQuery:      "",
		searchMode:       true, // Start in search mode
	}
}

// Init implements the tea.Model Init method
func (m clusterSelectorModel) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model Update method
func (m clusterSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "/":
			// Activate search mode
			m.searchMode = true
			m.searchQuery = ""
			return m, nil

		case "esc":
			if m.searchMode {
				// Exit search mode
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
			// Toggle between search mode and full view
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
				// If there are results, select the first one
				m.selected = &m.filteredClusters[m.cursor]
				return m, tea.Quit
			} else if !m.searchMode && len(m.filteredClusters) > 0 {
				// Select cluster
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
				// Adjust offset to keep the cursor visible
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.filteredClusters)-1 {
				m.cursor++
				// Adjust offset to keep the cursor visible
				currentVisibleLines := m.getCurrentVisibleLines()
				if m.cursor >= m.offset+currentVisibleLines {
					m.offset = m.cursor - currentVisibleLines + 1
				}
			}

		default:
			// If in search mode, add characters
			if m.searchMode && len(msg.String()) == 1 {
				m.searchQuery += msg.String()
				m.filterClusters()
			}
		}
	}

	return m, nil
}

// getCurrentVisibleLines calculates how many lines to show currently
func (m clusterSelectorModel) getCurrentVisibleLines() int {
	// Always limit to maximum 10 results
	return min(m.visibleLines, len(m.filteredClusters))
}

// filterClusters filters clusters based on the search query
func (m *clusterSelectorModel) filterClusters() {
	if m.searchQuery == "" {
		m.filteredClusters = m.clusters
		return
	}

	filtered := make([]services_kubernetes.ClusterContext, 0)
	query := strings.ToLower(m.searchQuery)

	for _, cluster := range m.clusters {
		// Search by cluster name
		if strings.Contains(strings.ToLower(cluster.Name), query) {
			filtered = append(filtered, cluster)
		}
	}

	m.filteredClusters = filtered
	// Reset cursor and offset when filtered clusters change
	m.cursor = 0
	m.offset = 0
}

// View implements the tea.Model View method
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
		// Show scroll indicator when there are more clusters
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

	// Calculate display window
	currentVisibleLines := m.getCurrentVisibleLines()
	startDisplay := m.offset
	endDisplay := min(m.offset+currentVisibleLines, len(m.filteredClusters))

	// Show indicator if there are clusters above
	if m.offset > 0 {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more clusters above)"))
		s.WriteString("\n")
	}

	// Render clusters in the visible window
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

	// Show indicator if there are clusters below
	if endDisplay < len(m.filteredClusters) {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more clusters below)"))
		s.WriteString("\n")
	}

	return s.String()
}

// formatClusterDisplay formats the cluster information for display
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

// InteractiveClusterSelector allows selecting a cluster interactively using Bubble Tea
func InteractiveClusterSelector() (*services_kubernetes.ClusterContext, error) {
	// Get all clusters
	clusters, err := services_kubernetes.GetClusterContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster contexts: %w", err)
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no cluster contexts found in kubeconfig")
	}

	// Create and run the Bubble Tea program
	model := initialClusterSelectorModel(clusters)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("error running cluster selector: %w", err)
	}

	// Check if a cluster was selected
	if finalModel.(clusterSelectorModel).selected == nil {
		return nil, fmt.Errorf("no cluster selected")
	}

	return finalModel.(clusterSelectorModel).selected, nil
}
