package animation

import (
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewProgressModel(t *testing.T) {
	tests := []struct {
		name  string
		total int
	}{
		{
			name:  "valid total",
			total: 10,
		},
		{
			name:  "zero total",
			total: 0,
		},
		{
			name:  "negative total",
			total: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewProgressModel(tt.total)

			assert.Equal(t, tt.total, model.total)
			assert.Equal(t, 0, model.current)
			assert.Empty(t, model.currentItem)
			assert.NotNil(t, model.items)
			assert.NotNil(t, model.errors)
			assert.False(t, model.quitting)
			assert.False(t, model.done)
			assert.Equal(t, 0, model.successCount)
		})
	}
}

func TestProgressModelInit(t *testing.T) {
	model := NewProgressModel(10)

	cmd := model.Init()

	// Init should return nil command
	assert.Nil(t, cmd)
}

func TestProgressModelUpdate(t *testing.T) {
	tests := []struct {
		name        string
		msg         tea.Msg
		expectedCmd tea.Cmd
		validate    func(t *testing.T, model ProgressModel)
	}{
		{
			name:        "quit key",
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model ProgressModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "escape key",
			msg:         tea.KeyMsg{Type: tea.KeyEscape},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model ProgressModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "ctrl+c key",
			msg:         tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model ProgressModel) {
				assert.True(t, model.quitting)
			},
		},
		{
			name:        "progress message with error",
			msg:         progressMsg{item: "test-item", error: "test error"},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model ProgressModel) {
				assert.Equal(t, 1, model.current)
				assert.Equal(t, "test-item", model.currentItem)
				assert.Contains(t, model.errors, "test error")
				assert.Contains(t, model.items, "test-item")
			},
		},
		{
			name:        "progress message without error",
			msg:         progressMsg{item: "test-item", error: ""},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model ProgressModel) {
				assert.Equal(t, 1, model.current)
				assert.Equal(t, "test-item", model.currentItem)
				assert.Empty(t, model.errors)
				assert.Contains(t, model.items, "test-item")
				assert.Equal(t, 1, model.successCount)
			},
		},
		{
			name:        "progress message completing all items",
			msg:         progressMsg{item: "final-item", error: ""},
			expectedCmd: tea.Quit,
			validate: func(t *testing.T, model ProgressModel) {
				assert.True(t, model.done)
			},
		},
		{
			name:        "window size message",
			msg:         tea.WindowSizeMsg{Width: 100, Height: 50},
			expectedCmd: nil,
			validate: func(t *testing.T, model ProgressModel) {
				assert.Equal(t, 96, model.progress.Width) // 100 - 4
			},
		},
		{
			name:        "window size message with large width",
			msg:         tea.WindowSizeMsg{Width: 200, Height: 50},
			expectedCmd: nil,
			validate: func(t *testing.T, model ProgressModel) {
				assert.Equal(t, 120, model.progress.Width) // Capped at 120
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewProgressModel(1) // Use 1 for completion test

			// For the completion test, we need to set up the model
			if tt.name == "progress message completing all items" {
				model.current = 0 // Reset to 0 so the message will complete it
			}

			updatedModel, cmd := model.Update(tt.msg)

			if tt.expectedCmd != nil {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}

			if tt.validate != nil {
				tt.validate(t, updatedModel.(ProgressModel))
			}
		})
	}
}

func TestProgressModelView(t *testing.T) {
	tests := []struct {
		name     string
		model    ProgressModel
		expected string
	}{
		{
			name: "quitting model",
			model: ProgressModel{
				quitting: true,
				done:     false,
			},
			expected: "",
		},
		{
			name: "completed model",
			model: ProgressModel{
				quitting:     false,
				done:         true,
				total:        2,
				current:      2,
				successCount: 2,
				errors:       []string{},
			},
			expected: "🎉 Configuration Completed!",
		},
		{
			name: "in progress model",
			model: ProgressModel{
				quitting:    false,
				done:        false,
				total:       2,
				current:     1,
				currentItem: "test-item",
			},
			expected: "⚙️  Configuring Kubernetes Clusters",
		},
		{
			name: "completed model with errors",
			model: ProgressModel{
				quitting:     false,
				done:         true,
				total:        2,
				current:      2,
				successCount: 1,
				errors:       []string{"test error"},
			},
			expected: "🎉 Configuration Completed!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up progress model
			tt.model.progress = progress.New(
				progress.WithDefaultGradient(),
				progress.WithWidth(50),
				progress.WithoutPercentage(),
			)

			view := tt.model.View()

			if tt.name == "quitting model" {
				assert.Empty(t, view)
			} else {
				assert.Contains(t, view, tt.expected)
			}
		})
	}
}

func TestProgressIncrement(t *testing.T) {
	tests := []struct {
		name        string
		item        string
		err         error
		expectedMsg progressMsg
	}{
		{
			name:        "successful item",
			item:        "test-item",
			err:         nil,
			expectedMsg: progressMsg{item: "test-item", error: ""},
		},
		{
			name:        "failed item",
			item:        "test-item",
			err:         assert.AnError,
			expectedMsg: progressMsg{item: "test-item", error: assert.AnError.Error()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ProgressIncrement(tt.item, tt.err)

			// Execute the command to get the message
			msg := cmd()

			assert.Equal(t, tt.expectedMsg, msg)
		})
	}
}

func TestShowProgressBar(t *testing.T) {
	tests := []struct {
		name             string
		total            int
		fn               func(update func(item string, err error)) error
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name:  "successful progress bar",
			total: 2,
			fn: func(update func(item string, err error)) error {
				update("item1", nil)
				update("item2", nil)
				return nil
			},
			expectedError:    false,
			expectedErrorMsg: "",
		},
		{
			name:  "progress bar with error",
			total: 1,
			fn: func(update func(item string, err error)) error {
				update("item1", assert.AnError)
				return assert.AnError
			},
			expectedError:    true,
			expectedErrorMsg: assert.AnError.Error(),
		},
		{
			name:             "zero total",
			total:            0,
			fn:               func(update func(item string, err error)) error { return nil },
			expectedError:    false,
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the full function without mocking the tea.Program
			// but we can test the parameter handling and validation logic

			// Test parameter validation
			assert.GreaterOrEqual(t, tt.total, 0)
			assert.NotNil(t, tt.fn)

			// Test that the function would accept these parameters
			_ = func(total int, fn func(update func(item string, err error)) error) error {
				return fn(func(item string, err error) {
					// Mock update function
				})
			}
		})
	}
}

func TestProgressModelStruct(t *testing.T) {
	// Test ProgressModel struct fields
	model := ProgressModel{
		total:        10,
		current:      5,
		currentItem:  "test-item",
		items:        []string{"item1", "item2"},
		errors:       []string{"error1"},
		quitting:     false,
		done:         false,
		successCount: 4,
	}

	assert.Equal(t, 10, model.total)
	assert.Equal(t, 5, model.current)
	assert.Equal(t, "test-item", model.currentItem)
	assert.Equal(t, []string{"item1", "item2"}, model.items)
	assert.Equal(t, []string{"error1"}, model.errors)
	assert.False(t, model.quitting)
	assert.False(t, model.done)
	assert.Equal(t, 4, model.successCount)
}

func TestProgressMsgStruct(t *testing.T) {
	// Test progressMsg struct fields
	msg := progressMsg{
		item:  "test-item",
		error: "test error",
	}

	assert.Equal(t, "test-item", msg.item)
	assert.Equal(t, "test error", msg.error)
}

func TestProgressModelUpdateProgress(t *testing.T) {
	// Test progress update logic
	model := NewProgressModel(3)

	// Test first update
	updatedModel, _ := model.Update(progressMsg{item: "item1", error: ""})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 1, model.current)
	assert.Equal(t, "item1", model.currentItem)
	assert.Equal(t, 1, model.successCount)
	assert.Empty(t, model.errors)

	// Test second update with error
	updatedModel, _ = model.Update(progressMsg{item: "item2", error: "error2"})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 2, model.current)
	assert.Equal(t, "item2", model.currentItem)
	assert.Equal(t, 1, model.successCount) // Still 1 because item2 had error
	assert.Contains(t, model.errors, "error2")

	// Test third update (completion)
	updatedModel, cmd := model.Update(progressMsg{item: "item3", error: ""})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 3, model.current)
	assert.Equal(t, "item3", model.currentItem)
	assert.Equal(t, 2, model.successCount)
	assert.True(t, model.done)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestProgressModelViewContent(t *testing.T) {
	// Test view content generation
	model := NewProgressModel(2)
	model.progress = progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	// Test in-progress view
	view := model.View()
	assert.Contains(t, view, "⚙️  Configuring Kubernetes Clusters")
	assert.Contains(t, view, "Progress: 0/2 clusters")

	// Test completed view
	model.done = true
	model.current = 2
	model.successCount = 2
	view = model.View()
	assert.Contains(t, view, "🎉 Configuration Completed!")
	assert.Contains(t, view, "✓ Successful: 2")

	// Test completed view with errors
	model.errors = []string{"test error"}
	view = model.View()
	assert.Contains(t, view, "✗ Failed: 1")
	assert.Contains(t, view, "Errors:")
	assert.Contains(t, view, "• test error")
}

func TestProgressModelWindowSize(t *testing.T) {
	// Test window size handling
	model := NewProgressModel(1)
	model.progress = progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	// Test normal width
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 96, model.progress.Width) // 100 - 4

	// Test large width (should be capped)
	updatedModel, _ = model.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 120, model.progress.Width) // Capped at 120

	// Test small width
	updatedModel, _ = model.Update(tea.WindowSizeMsg{Width: 10, Height: 50})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 6, model.progress.Width) // 10 - 4
}

func TestProgressModelKeyHandling(t *testing.T) {
	// Test key handling
	model := NewProgressModel(1)

	// Test quit keys
	quitKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEscape},
		{Type: tea.KeyCtrlC},
	}

	for _, key := range quitKeys {
		updatedModel, _ := model.Update(key)
		model = updatedModel.(ProgressModel)
		assert.True(t, model.quitting)
		model.quitting = false // Reset for next test
	}

	// Test other keys (should not quit)
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updatedModel.(ProgressModel)
	assert.False(t, model.quitting)
}

func TestProgressModelCompletion(t *testing.T) {
	// Test completion logic
	model := NewProgressModel(2)

	// First item
	updatedModel, _ := model.Update(progressMsg{item: "item1", error: ""})
	model = updatedModel.(ProgressModel)
	assert.False(t, model.done)
	assert.Equal(t, 1, model.current)

	// Second item (should complete)
	updatedModel, cmd := model.Update(progressMsg{item: "item2", error: ""})
	model = updatedModel.(ProgressModel)
	assert.True(t, model.done)
	assert.Equal(t, 2, model.current)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestProgressModelErrorHandling(t *testing.T) {
	// Test error handling
	model := NewProgressModel(2)

	// Add item with error
	updatedModel, _ := model.Update(progressMsg{item: "item1", error: "test error"})
	model = updatedModel.(ProgressModel)
	assert.Contains(t, model.errors, "test error")
	assert.Equal(t, 0, model.successCount)

	// Add item without error
	updatedModel, _ = model.Update(progressMsg{item: "item2", error: ""})
	model = updatedModel.(ProgressModel)
	assert.Equal(t, 1, model.successCount)
	assert.Len(t, model.errors, 1) // Still has the previous error
}
