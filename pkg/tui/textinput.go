package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TextInput wraps a textinput.Model with reusable behaviors:
// cursor blinking, message forwarding, and bordered title rendering.
// Access the inner Model directly for Value(), Reset(), SetValue(), etc.
type TextInput struct {
	Model textinput.Model
}

// NewTextInput creates a focused TextInput with cursor blink support.
func NewTextInput(prompt string, placeholder string) TextInput {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.Placeholder = placeholder
	ti.Width = len(placeholder)
	ti.Focus()

	return TextInput{Model: ti}
}

// NewFilterInput creates a TextInput configured for filtering lists.
// Suggestion keys (up/down/ctrl+p/ctrl+n) are disabled so they can be
// used for list navigation instead.
func NewFilterInput(prompt string, placeholder string) TextInput {
	f := NewTextInput(prompt, placeholder)

	km := textinput.DefaultKeyMap
	km.NextSuggestion = key.NewBinding()
	km.PrevSuggestion = key.NewBinding()
	km.AcceptSuggestion = key.NewBinding()
	f.Model.KeyMap = km

	return f
}

// Init returns the blink command to start cursor animation.
func (f TextInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update forwards messages to the inner textinput model.
func (f TextInput) Update(msg tea.Msg) (TextInput, tea.Cmd) {
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

// ViewTitle renders the textinput inside a bordered title bar.
func (f TextInput) ViewTitle() string {
	return BorderStyle.Padding(0, 1).Render(f.Model.View())
}
