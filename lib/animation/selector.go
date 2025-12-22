package animation

import (
	"fmt"
	"strings"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProfileDisplayInfo contains information to show in the interactive list
type ProfileDisplayInfo struct {
	Name        string
	Type        string
	Description string
	AccountID   string
	RoleName    string
	Region      string
}

// profileSelectorModel represents the model for the profile selector with Bubble Tea
type profileSelectorModel struct {
	profiles         []services_aws.ProfileConfig
	filteredProfiles []services_aws.ProfileConfig
	cursor           int
	offset           int // Index of the first visible profile
	visibleLines     int // Maximum number of profiles to show
	searchQuery      string
	selected         *services_aws.ProfileConfig
	quitting         bool
	searchMode       bool
}

// initialProfileSelectorModel creates the initial model for the selector
func initialProfileSelectorModel(profiles []services_aws.ProfileConfig) profileSelectorModel {
	return profileSelectorModel{
		profiles:         profiles,
		filteredProfiles: profiles,
		cursor:           0,
		offset:           0,
		visibleLines:     10, // Show maximum 10 profiles
		searchQuery:      "",
		searchMode:       true, // Start in search mode
	}
}

// Init implements the tea.Model Init method
func (m profileSelectorModel) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model Update method
func (m profileSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.filteredProfiles = m.profiles
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
				// If there are results, select the first one
				m.selected = &m.filteredProfiles[m.cursor]
				return m, tea.Quit
			} else if !m.searchMode && len(m.filteredProfiles) > 0 {
				// Select profile
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
				// Adjust offset to keep the cursor visible
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.filteredProfiles)-1 {
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
				m.filterProfiles()
			}
		}
	}

	return m, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCurrentVisibleLines calculates how many lines to show currently
func (m profileSelectorModel) getCurrentVisibleLines() int {
	// Always limit to maximum 10 results
	return min(m.visibleLines, len(m.filteredProfiles))
}

// filterProfiles filters profiles based on the search query
func (m *profileSelectorModel) filterProfiles() {
	if m.searchQuery == "" {
		m.filteredProfiles = m.profiles
		return
	}

	filtered := make([]services_aws.ProfileConfig, 0)
	query := strings.ToLower(m.searchQuery)

	for _, profile := range m.profiles {
		// Search by profile name
		if strings.Contains(strings.ToLower(profile.ProfileName), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Search by account ID
		if strings.Contains(strings.ToLower(profile.AccountID), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Search by role name
		if strings.Contains(strings.ToLower(profile.RoleName), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Search by role ARN
		if strings.Contains(strings.ToLower(profile.RoleARN), query) {
			filtered = append(filtered, profile)
			continue
		}

		// Search by source profile
		if strings.Contains(strings.ToLower(profile.SourceProfile), query) {
			filtered = append(filtered, profile)
			continue
		}
	}

	m.filteredProfiles = filtered
	// Reset cursor and offset when filtered profiles change
	m.cursor = 0
	m.offset = 0
}

// View implements the tea.Model View method
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
		// Show scroll indicator when there are more profiles
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

	// Calculate display window
	currentVisibleLines := m.getCurrentVisibleLines()
	startDisplay := m.offset
	endDisplay := min(m.offset+currentVisibleLines, len(m.filteredProfiles))

	// Show indicator if there are profiles above
	if m.offset > 0 {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more profiles above)"))
		s.WriteString("\n")
	}

	// Render profiles in the visible window
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

	// Show indicator if there are profiles below
	if endDisplay < len(m.filteredProfiles) {
		ellipsisStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		s.WriteString(ellipsisStyle.Render("... (more profiles below)"))
		s.WriteString("\n")
	}

	return s.String()
}

// formatProfileDisplay formats the profile information for display
func formatProfileDisplay(profile services_aws.ProfileConfig) ProfileDisplayInfo {
	var description string
	var accountID, roleName string

	switch profile.ProfileType {
	case services_aws.ProfileTypeSSO:
		accountID = profile.AccountID
		roleName = profile.RoleName
		description = fmt.Sprintf("SSO - Account: %s, Role: %s", accountID, roleName)
	case services_aws.ProfileTypeAssumeRole:
		// Extract account ID from ARN
		if strings.Contains(profile.RoleARN, ":") {
			parts := strings.Split(profile.RoleARN, ":")
			if len(parts) >= 5 {
				accountID = parts[4]
			}
		}
		// Extract role name from ARN
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

// InteractiveProfileSelector allows selecting a profile interactively using Bubble Tea
func InteractiveProfileSelector() (*services_aws.ProfileConfig, error) {
	// Get all profiles
	profiles, err := services_aws.ReadAllProfilesFromConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles: %w", err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found in AWS config")
	}

	// Create and run the Bubble Tea program
	model := initialProfileSelectorModel(profiles)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("error running profile selector: %w", err)
	}

	// Check if a profile was selected
	if finalModel.(profileSelectorModel).selected == nil {
		return nil, fmt.Errorf("no profile selected")
	}

	return finalModel.(profileSelectorModel).selected, nil
}
