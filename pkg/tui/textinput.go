package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
)

// NewFilterInput creates a textinput configured for filtering lists.
// Suggestion keys (up/down/ctrl+p/ctrl+n) are disabled so they can be
// used for list navigation instead.
func NewFilterInput(prompt string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.Placeholder = ""
	ti.Focus()

	// Disable suggestion navigation keys so up/down work for list navigation
	km := textinput.DefaultKeyMap
	km.NextSuggestion = key.NewBinding()
	km.PrevSuggestion = key.NewBinding()
	km.AcceptSuggestion = key.NewBinding()
	ti.KeyMap = km

	return ti
}
