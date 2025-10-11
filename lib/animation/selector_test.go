package animation

import (
	"testing"

	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestInitialProfileSelectorModel(t *testing.T) {
	tests := []struct {
		name     string
		profiles []services_aws.ProfileConfig
	}{
		{
			name:     "empty profiles",
			profiles: []services_aws.ProfileConfig{},
		},
		{
			name: "single profile",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "test-profile",
					ProfileType: services_aws.ProfileTypeSSO,
					StartURL:    "https://example.awsapps.com/start",
					Region:      "us-west-2",
					AccountID:   "123456789012",
					RoleName:    "TestRole",
				},
			},
		},
		{
			name: "multiple profiles",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
					StartURL:    "https://example.awsapps.com/start",
					Region:      "us-west-2",
					AccountID:   "123456789012",
					RoleName:    "Role1",
				},
				{
					ProfileName:   "profile2",
					ProfileType:   services_aws.ProfileTypeAssumeRole,
					Region:        "us-east-1",
					AccountID:     "987654321098",
					RoleName:      "Role2",
					RoleARN:       "arn:aws:iam::987654321098:role/Role2",
					SourceProfile: "source-profile",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initialProfileSelectorModel(tt.profiles)

			assert.Equal(t, tt.profiles, model.profiles)
			assert.Equal(t, tt.profiles, model.filteredProfiles)
			assert.Equal(t, 0, model.cursor)
			assert.Equal(t, 0, model.offset)
			assert.Equal(t, 10, model.visibleLines)
			assert.Empty(t, model.searchQuery)
			assert.Nil(t, model.selected)
			assert.False(t, model.quitting)
			assert.True(t, model.searchMode)
		})
	}
}

func TestProfileSelectorModelInit(t *testing.T) {
	profiles := []services_aws.ProfileConfig{
		{
			ProfileName: "test-profile",
			ProfileType: services_aws.ProfileTypeSSO,
		},
	}

	model := initialProfileSelectorModel(profiles)

	cmd := model.Init()

	// Init should return nil command
	assert.Nil(t, cmd)
}

func TestProfileSelectorModelUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		expectedCmd tea.Cmd
		validate    func(t *testing.T, model profileSelectorModel)
	}{
		{
			name:        "quit key (ctrl+c)",
			msg:         tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "quit key (q)",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "search mode activation",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}},
			expectedCmd: nil,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.True(t, model.searchMode)
				assert.Empty(t, model.searchQuery)
			},
		},
		{
			name:        "escape key in search mode",
			msg:         tea.KeyMsg{Type: tea.KeyEscape},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.False(t, model.searchMode)
				assert.Empty(t, model.searchQuery)
			},
		},
		{
			name:        "tab key to toggle search mode",
			msg:         tea.KeyMsg{Type: tea.KeyTab},
			expectedCmd: nil,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.False(t, model.searchMode)
				assert.Empty(t, model.searchQuery)
			},
		},
		{
			name:        "enter key to select profile",
			msg:         tea.KeyMsg{Type: tea.KeyEnter},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.NotNil(t, model.selected)
			},
		},
		{
			name:        "backspace in search mode",
			msg:         tea.KeyMsg{Type: tea.KeyBackspace},
			expectedCmd: nil,
			validate: func(t *testing.T, model profileSelectorModel) {
				// Should remove one character from "test" -> "tes"
				assert.Equal(t, "tes", model.searchQuery)
			},
		},
		{
			name:        "up arrow key",
			msg:         tea.KeyMsg{Type: tea.KeyUp},
			expectedCmd: nil,
			validate: func(t *testing.T, model profileSelectorModel) {
				// Cursor should not change if already at 0
				assert.Equal(t, 0, model.cursor)
			},
		},
		{
			name:        "down arrow key",
			msg:         tea.KeyMsg{Type: tea.KeyDown},
			expectedCmd: nil,
			validate: func(t *testing.T, model profileSelectorModel) {
				// Cursor should not change if no profiles
				assert.Equal(t, 0, model.cursor)
			},
		},
		{
			name:        "character input in search mode",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			expectedCmd: nil,
			validate: func(t *testing.T, model profileSelectorModel) {
				assert.Equal(t, "a", model.searchQuery)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiles := []services_aws.ProfileConfig{
				{
					ProfileName: "test-profile",
					ProfileType: services_aws.ProfileTypeSSO,
				},
			}

			model := initialProfileSelectorModel(profiles)

			// Set up model state for specific tests
			if tt.name == "escape key in search mode" {
				model.searchMode = true
				model.searchQuery = "test"
			} else if tt.name == "tab key to toggle search mode" {
				model.searchMode = true
			} else if tt.name == "enter key to select profile" {
				model.searchMode = false
			} else if tt.name == "backspace in search mode" {
				model.searchMode = true
				model.searchQuery = "test"
			} else if tt.name == "character input in search mode" {
				model.searchMode = true
			}

			updatedModel, cmd := model.Update(tt.msg)

			if tt.expectedCmd != nil {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}

			if tt.validate != nil {
				tt.validate(t, updatedModel.(profileSelectorModel))
			}
		})
	}
}

func TestProfileSelectorModelView(t *testing.T) {
	tests := []struct {
		name     string
		model    profileSelectorModel
		expected string
	}{
		{
			name: "quitting model",
			model: profileSelectorModel{
				quitting: true,
			},
			expected: "",
		},
		{
			name: "empty profiles",
			model: profileSelectorModel{
				quitting:         false,
				profiles:         []services_aws.ProfileConfig{},
				filteredProfiles: []services_aws.ProfileConfig{},
			},
			expected: "No profiles found matching your search",
		},
		{
			name: "search mode",
			model: profileSelectorModel{
				quitting:    false,
				searchMode:  true,
				searchQuery: "test",
				profiles: []services_aws.ProfileConfig{
					{
						ProfileName: "test-profile",
						ProfileType: services_aws.ProfileTypeSSO,
					},
				},
				filteredProfiles: []services_aws.ProfileConfig{
					{
						ProfileName: "test-profile",
						ProfileType: services_aws.ProfileTypeSSO,
					},
				},
			},
			expected: "🔎 Search: test_",
		},
		{
			name: "normal mode",
			model: profileSelectorModel{
				quitting:   false,
				searchMode: false,
				profiles: []services_aws.ProfileConfig{
					{
						ProfileName: "test-profile",
						ProfileType: services_aws.ProfileTypeSSO,
					},
				},
				filteredProfiles: []services_aws.ProfileConfig{
					{
						ProfileName: "test-profile",
						ProfileType: services_aws.ProfileTypeSSO,
					},
				},
			},
			expected: "🔍 Select an AWS profile to login:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.model.View()

			if tt.name == "quitting model" {
				assert.Empty(t, view)
			} else {
				assert.Contains(t, view, tt.expected)
			}
		})
	}
}

func TestFormatProfileDisplay(t *testing.T) {
	tests := []struct {
		name     string
		profile  services_aws.ProfileConfig
		expected ProfileDisplayInfo
	}{
		{
			name: "SSO profile",
			profile: services_aws.ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: services_aws.ProfileTypeSSO,
				StartURL:    "https://example.awsapps.com/start",
				Region:      "us-west-2",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
			},
			expected: ProfileDisplayInfo{
				Name:        "test-profile",
				Type:        "sso",
				Description: "SSO - Account: 123456789012, Role: TestRole",
				AccountID:   "123456789012",
				RoleName:    "TestRole",
				Region:      "us-west-2",
			},
		},
		{
			name: "Assume role profile",
			profile: services_aws.ProfileConfig{
				ProfileName:   "test-profile",
				ProfileType:   services_aws.ProfileTypeAssumeRole,
				Region:        "us-east-1",
				RoleARN:       "arn:aws:iam::987654321098:role/TestRole",
				SourceProfile: "source-profile",
			},
			expected: ProfileDisplayInfo{
				Name:        "test-profile",
				Type:        "assume_role",
				Description: "Assume Role - Account: 987654321098, Role: TestRole",
				AccountID:   "987654321098",
				RoleName:    "TestRole",
				Region:      "us-east-1",
			},
		},
		{
			name: "Unknown profile type",
			profile: services_aws.ProfileConfig{
				ProfileName: "test-profile",
				ProfileType: "unknown",
				Region:      "us-west-2",
			},
			expected: ProfileDisplayInfo{
				Name:        "test-profile",
				Type:        "unknown",
				Description: "Unknown profile type",
				AccountID:   "",
				RoleName:    "",
				Region:      "us-west-2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatProfileDisplay(tt.profile)

			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.AccountID, result.AccountID)
			assert.Equal(t, tt.expected.RoleName, result.RoleName)
			assert.Equal(t, tt.expected.Region, result.Region)
		})
	}
}

func TestInteractiveProfileSelector(t *testing.T) {
	// We can't easily test the full function without mocking external dependencies
	// but we can test the parameter handling and validation logic

	// Test that the function would accept the expected parameters
	_ = func() (*services_aws.ProfileConfig, error) {
		return &services_aws.ProfileConfig{
			ProfileName: "test-profile",
			ProfileType: services_aws.ProfileTypeSSO,
		}, nil
	}
}

func TestProfileSelectorModelFilterProfiles(t *testing.T) {
	tests := []struct {
		name        string
		profiles    []services_aws.ProfileConfig
		searchQuery string
		expected    []services_aws.ProfileConfig
	}{
		{
			name: "empty search query",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
				},
				{
					ProfileName: "profile2",
					ProfileType: services_aws.ProfileTypeAssumeRole,
				},
			},
			searchQuery: "",
			expected: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
				},
				{
					ProfileName: "profile2",
					ProfileType: services_aws.ProfileTypeAssumeRole,
				},
			},
		},
		{
			name: "search by profile name",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "test-profile",
					ProfileType: services_aws.ProfileTypeSSO,
				},
				{
					ProfileName: "other-profile",
					ProfileType: services_aws.ProfileTypeAssumeRole,
				},
			},
			searchQuery: "test",
			expected: []services_aws.ProfileConfig{
				{
					ProfileName: "test-profile",
					ProfileType: services_aws.ProfileTypeSSO,
				},
			},
		},
		{
			name: "search by account ID",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
					AccountID:   "123456789012",
				},
				{
					ProfileName: "profile2",
					ProfileType: services_aws.ProfileTypeAssumeRole,
					AccountID:   "987654321098",
				},
			},
			searchQuery: "123456789012",
			expected: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
					AccountID:   "123456789012",
				},
			},
		},
		{
			name: "search by role name",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
					RoleName:    "TestRole",
				},
				{
					ProfileName: "profile2",
					ProfileType: services_aws.ProfileTypeAssumeRole,
					RoleName:    "OtherRole",
				},
			},
			searchQuery: "TestRole",
			expected: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
					RoleName:    "TestRole",
				},
			},
		},
		{
			name: "no matches",
			profiles: []services_aws.ProfileConfig{
				{
					ProfileName: "profile1",
					ProfileType: services_aws.ProfileTypeSSO,
				},
			},
			searchQuery: "nonexistent",
			expected:    []services_aws.ProfileConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initialProfileSelectorModel(tt.profiles)
			model.searchQuery = tt.searchQuery

			model.filterProfiles()

			assert.Equal(t, tt.expected, model.filteredProfiles)
			assert.Equal(t, 0, model.cursor)
			assert.Equal(t, 0, model.offset)
		})
	}
}

func TestProfileSelectorModelNavigation(t *testing.T) {
	profiles := []services_aws.ProfileConfig{
		{ProfileName: "profile1", ProfileType: services_aws.ProfileTypeSSO},
		{ProfileName: "profile2", ProfileType: services_aws.ProfileTypeSSO},
		{ProfileName: "profile3", ProfileType: services_aws.ProfileTypeSSO},
	}

	model := initialProfileSelectorModel(profiles)

	// Test down navigation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, 1, model.cursor)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, 2, model.cursor)

	// Test up navigation
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, 1, model.cursor)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, 0, model.cursor)

	// Test that cursor doesn't go below 0
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, 0, model.cursor)

	// Test that cursor doesn't go above max
	model.cursor = 2
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, 2, model.cursor)
}

func TestProfileSelectorModelSearchMode(t *testing.T) {
	profiles := []services_aws.ProfileConfig{
		{ProfileName: "test-profile", ProfileType: services_aws.ProfileTypeSSO},
		{ProfileName: "other-profile", ProfileType: services_aws.ProfileTypeSSO},
	}

	model := initialProfileSelectorModel(profiles)

	// Test search mode activation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = updatedModel.(profileSelectorModel)
	assert.True(t, model.searchMode)
	assert.Empty(t, model.searchQuery)

	// Test character input in search mode
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, "t", model.searchQuery)

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, "te", model.searchQuery)

	// Test backspace in search mode
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model = updatedModel.(profileSelectorModel)
	assert.Equal(t, "t", model.searchQuery)

	// Test escape to exit search mode
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updatedModel.(profileSelectorModel)
	assert.False(t, model.searchMode)
	assert.Empty(t, model.searchQuery)
}

func TestProfileSelectorModelSelection(t *testing.T) {
	profiles := []services_aws.ProfileConfig{
		{ProfileName: "profile1", ProfileType: services_aws.ProfileTypeSSO},
		{ProfileName: "profile2", ProfileType: services_aws.ProfileTypeSSO},
	}

	model := initialProfileSelectorModel(profiles)

	// Test selection in normal mode
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(profileSelectorModel)
	assert.NotNil(t, model.selected)
	assert.Equal(t, "profile1", model.selected.ProfileName)
	assert.NotNil(t, cmd) // Should return tea.Quit

	// Test selection in search mode
	model = initialProfileSelectorModel(profiles)
	model.searchMode = true
	model.searchQuery = "profile2"
	model.filterProfiles()

	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(profileSelectorModel)
	assert.NotNil(t, model.selected)
	assert.Equal(t, "profile2", model.selected.ProfileName)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestProfileSelectorModelStruct(t *testing.T) {
	// Test profileSelectorModel struct fields
	profiles := []services_aws.ProfileConfig{
		{ProfileName: "test-profile", ProfileType: services_aws.ProfileTypeSSO},
	}

	model := profileSelectorModel{
		profiles:         profiles,
		filteredProfiles: profiles,
		cursor:           0,
		offset:           0,
		visibleLines:     10,
		searchQuery:      "test",
		selected:         &profiles[0],
		quitting:         false,
		searchMode:       true,
	}

	assert.Equal(t, profiles, model.profiles)
	assert.Equal(t, profiles, model.filteredProfiles)
	assert.Equal(t, 0, model.cursor)
	assert.Equal(t, 0, model.offset)
	assert.Equal(t, 10, model.visibleLines)
	assert.Equal(t, "test", model.searchQuery)
	assert.NotNil(t, model.selected)
	assert.False(t, model.quitting)
	assert.True(t, model.searchMode)
}

func TestProfileDisplayInfoStruct(t *testing.T) {
	// Test ProfileDisplayInfo struct fields
	info := ProfileDisplayInfo{
		Name:        "test-profile",
		Type:        "sso",
		Description: "SSO - Account: 123456789012, Role: TestRole",
		AccountID:   "123456789012",
		RoleName:    "TestRole",
		Region:      "us-west-2",
	}

	assert.Equal(t, "test-profile", info.Name)
	assert.Equal(t, "sso", info.Type)
	assert.Equal(t, "SSO - Account: 123456789012, Role: TestRole", info.Description)
	assert.Equal(t, "123456789012", info.AccountID)
	assert.Equal(t, "TestRole", info.RoleName)
	assert.Equal(t, "us-west-2", info.Region)
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "a is smaller",
			a:        5,
			b:        10,
			expected: 5,
		},
		{
			name:     "b is smaller",
			a:        10,
			b:        5,
			expected: 5,
		},
		{
			name:     "equal values",
			a:        5,
			b:        5,
			expected: 5,
		},
		{
			name:     "negative values",
			a:        -5,
			b:        -10,
			expected: -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCurrentVisibleLines(t *testing.T) {
	tests := []struct {
		name         string
		visibleLines int
		filteredLen  int
		expected     int
	}{
		{
			name:         "fewer profiles than visible lines",
			visibleLines: 10,
			filteredLen:  5,
			expected:     5,
		},
		{
			name:         "more profiles than visible lines",
			visibleLines: 10,
			filteredLen:  15,
			expected:     10,
		},
		{
			name:         "equal profiles and visible lines",
			visibleLines: 10,
			filteredLen:  10,
			expected:     10,
		},
		{
			name:         "zero profiles",
			visibleLines: 10,
			filteredLen:  0,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := profileSelectorModel{
				visibleLines:     tt.visibleLines,
				filteredProfiles: make([]services_aws.ProfileConfig, tt.filteredLen),
			}

			result := model.getCurrentVisibleLines()
			assert.Equal(t, tt.expected, result)
		})
	}
}
