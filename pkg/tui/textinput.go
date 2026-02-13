package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// FilterInput wraps a textinput.Model with reusable behaviors:
// cursor blinking, message forwarding, and bordered title rendering.
// Access the inner Model directly for Value(), Reset(), SetValue(), etc.
type FilterInput struct {
	Model textinput.Model
}

// NewFilterInput creates a FilterInput configured for filtering lists.
// Suggestion keys (up/down/ctrl+p/ctrl+n) are disabled so they can be
// used for list navigation instead.
func NewFilterInput(prompt string, placeholder string) FilterInput {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.Placeholder = placeholder
	ti.Width = len(placeholder)
	ti.Focus()

	// Disable suggestion navigation keys so up/down work for list navigation
	km := textinput.DefaultKeyMap
	km.NextSuggestion = key.NewBinding()
	km.PrevSuggestion = key.NewBinding()
	km.AcceptSuggestion = key.NewBinding()
	ti.KeyMap = km

	return FilterInput{Model: ti}
}

// Init returns the blink command to start cursor animation.
func (f FilterInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update forwards messages to the inner textinput model.
// This ensures blink ticks and other non-key messages are handled.
func (f FilterInput) Update(msg tea.Msg) (FilterInput, tea.Cmd) {
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

// ViewTitle renders the textinput inside a bordered title bar.
func (f FilterInput) ViewTitle() string {
	return BorderStyle.Padding(0, 1).Render(f.Model.View())
}
