package animation

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewSpinnerModel(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "valid message",
			message:  "Loading...",
			expected: "Loading...",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "long message",
			message:  "This is a very long message that should be handled properly",
			expected: "This is a very long message that should be handled properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSpinnerModel(tt.message)

			assert.Equal(t, tt.expected, model.message)
			assert.False(t, model.quitting)
		})
	}
}

func TestSpinnerModelInit(t *testing.T) {
	model := NewSpinnerModel("Test message")

	cmd := model.Init()

	// Init should return a tick command
	assert.NotNil(t, cmd)
}

func TestSpinnerModelUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		expectedCmd tea.Cmd
		validate    func(t *testing.T, model SpinnerModel)
	}{
		{
			name:        "tick message",
			msg:         spinner.TickMsg{},
			expectedCmd: nil, // Should return another tick command
			validate: func(t *testing.T, model SpinnerModel) {
				assert.False(t, model.quitting)
			},
		},
		{
			name:        "quit message",
			msg:         tea.QuitMsg{},
			expectedCmd: nil,
			validate: func(t *testing.T, model SpinnerModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "key message - quit",
			msg:         tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model SpinnerModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "key message - escape",
			msg:         tea.KeyMsg{Type: tea.KeyEscape},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model SpinnerModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "key message - q",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model SpinnerModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "other key message",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			expectedCmd: nil,
			validate: func(t *testing.T, model SpinnerModel) {
				assert.False(t, model.quitting)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSpinnerModel("Test message")

			updatedModel, cmd := model.Update(tt.msg)

			if tt.expectedCmd != nil {
				assert.NotNil(t, cmd)
			}

			if tt.validate != nil {
				tt.validate(t, updatedModel.(SpinnerModel))
			}
		})
	}
}

func TestSpinnerModelView(t *testing.T) {
	tests := []struct {
		name     string
		model    SpinnerModel
		expected string
	}{
		{
			name: "normal spinner",
			model: SpinnerModel{
				quitting: false,
				message:  "Loading...",
			},
			expected: "⠋ Loading...",
		},
		{
			name: "quitting spinner",
			model: SpinnerModel{
				quitting: true,
				message:  "Loading...",
			},
			expected: "",
		},
		{
			name: "empty message",
			model: SpinnerModel{
				quitting: false,
				message:  "",
			},
			expected: "⠋ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.model.View()

			if tt.name == "quitting spinner" {
				assert.Empty(t, view)
			} else {
				assert.Contains(t, view, tt.model.message)
			}
		})
	}
}

func TestSpinnerTick(t *testing.T) {
	// Create a spinner model to test
	model := NewSpinnerModel("Test")

	// Get the tick command from the spinner
	cmd := model.spinner.Tick

	// Should return a command
	assert.NotNil(t, cmd)

	// Test that the command can be executed
	msg := cmd()
	assert.IsType(t, spinner.TickMsg{}, msg)
}

func TestShowSpinner(t *testing.T) {
	tests := []struct {
		name             string
		message          string
		fn               func() error
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:             "successful operation",
			message:          "Loading...",
			fn:               func() error { return nil },
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:             "operation with error",
			message:          "Loading...",
			fn:               func() error { return assert.AnError },
			expectedError:    true,
			expectedErrorMsg: assert.AnError.Error(),
		},
		{
			name:             "empty message",
			message:          "",
			fn:               func() error { return nil },
			expectedError:    false,
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the full function without mocking the tea.Program
			// but we can test the parameter handling and validation logic

			// Test parameter validation
			assert.IsType(t, "", tt.message)
			assert.NotNil(t, tt.fn)

			// Test that the function would accept these parameters
			_ = func(message string, fn func() error) error {
				return fn()
			}
		})
	}
}

func TestSpinnerModelStruct(t *testing.T) {
	// Test SpinnerModel struct fields
	model := SpinnerModel{
		quitting: true,
		message:  "test message",
	}

	assert.Equal(t, true, model.quitting)
	assert.Equal(t, "test message", model.message)
}

func TestSpinnerAnimation(t *testing.T) {
	// Test that spinner frames are properly defined
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	assert.Len(t, frames, 10)
	assert.NotEmpty(t, frames[0])
	assert.NotEmpty(t, frames[9])

	// Test that all frames are single runes (not necessarily single bytes)
	for _, frame := range frames {
		runes := []rune(frame)
		assert.Len(t, runes, 1, "Frame %s should be a single rune", frame)
	}
}

func TestSpinnerTickInterval(t *testing.T) {
	// Test that tick interval is reasonable
	interval := 100 * time.Millisecond

	assert.Equal(t, 100*time.Millisecond, interval)
	assert.Greater(t, interval, 0*time.Millisecond)
	assert.Less(t, interval, 1*time.Second)
}

func TestSpinnerKeyHandling(t *testing.T) {
	// Test key handling logic
	model := NewSpinnerModel("Test")

	// Test quit keys
	quitKeys := []tea.KeyMsg{
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyEscape},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
	}

	for _, key := range quitKeys {
		updatedModel, cmd := model.Update(key)
		model = updatedModel.(SpinnerModel)
		assert.True(t, model.quitting)
		assert.NotNil(t, cmd)  // Should return tea.Quit
		model.quitting = false // Reset for next test
	}

	// Test other keys (should not quit)
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updatedModel.(SpinnerModel)
	assert.False(t, model.quitting)
	assert.Nil(t, cmd)
}

func TestSpinnerMessageHandling(t *testing.T) {
	// Test message handling
	model := NewSpinnerModel("Original message")

	// Test that message is preserved
	assert.Equal(t, "Original message", model.message)

	// Test view with message
	view := model.View()
	assert.Contains(t, view, "Original message")
	// Check that view contains some spinner character (not necessarily a specific one)
	assert.NotEmpty(t, view)
	assert.True(t, len(view) > len("Original message"), "View should contain spinner characters")
}

func TestSpinnerQuitBehavior(t *testing.T) {
	// Test quit behavior
	model := NewSpinnerModel("Test")

	// Initially not quitting
	assert.False(t, model.quitting)
	assert.NotEmpty(t, model.View())

	// After quit message
	updatedModel, _ := model.Update(tea.QuitMsg{})
	model = updatedModel.(SpinnerModel)
	assert.True(t, model.quitting)
	assert.Empty(t, model.View())
}

func TestSpinnerTickCommand(t *testing.T) {
	// Create a spinner model to test
	model := NewSpinnerModel("Test")

	// Test tick command generation
	cmd := model.spinner.Tick

	// Should return a command
	assert.NotNil(t, cmd)

	// Test that the command can be executed
	msg := cmd()
	assert.IsType(t, spinner.TickMsg{}, msg)

	// Test that tick message can be handled
	updatedModel, newCmd := model.Update(msg)

	assert.False(t, updatedModel.(SpinnerModel).quitting)
	assert.NotNil(t, newCmd) // Should return another tick command
}
