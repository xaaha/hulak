package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewFilterInputPrompt(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "Filter: "})

	if ti.Model.Prompt != "Filter: " {
		t.Errorf("expected prompt 'Filter: ', got %q", ti.Model.Prompt)
	}
}

func TestNewFilterInputEmptyPlaceholder(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "Filter: "})

	if ti.Model.Placeholder != "" {
		t.Errorf("expected empty placeholder, got %q", ti.Model.Placeholder)
	}
	if ti.Model.Width != 0 {
		t.Errorf("expected width 0 for empty placeholder, got %d", ti.Model.Width)
	}
}

func TestNewFilterInputCustomPlaceholder(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "Search: ", Placeholder: "type to search"})

	if ti.Model.Placeholder != "type to search" {
		t.Errorf("expected placeholder 'type to search', got %q", ti.Model.Placeholder)
	}
	if ti.Model.Width != len("type to search") {
		t.Errorf("expected width %d, got %d", len("type to search"), ti.Model.Width)
	}
}

func TestNewFilterInputWidthMatchesPlaceholderLength(t *testing.T) {
	tests := []struct {
		name        string
		placeholder string
		wantWidth   int
	}{
		{"empty", "", 0},
		{"short", "abc", 3},
		{"medium", "global", 6},
		{"long", "enter environment name here", 27},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ti := NewFilterInput(TextInputOpts{Prompt: "> ", Placeholder: tc.placeholder})
			if ti.Model.Width != tc.wantWidth {
				t.Errorf("placeholder %q: expected width %d, got %d", tc.placeholder, tc.wantWidth, ti.Model.Width)
			}
		})
	}
}

func TestNewFilterInputMinWidthOverridesPlaceholder(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "> ", Placeholder: "abc", MinWidth: 20})
	if ti.Model.Width != 20 {
		t.Errorf("expected width 20 from MinWidth, got %d", ti.Model.Width)
	}
}

func TestNewFilterInputMinWidthNoEffectWhenSmaller(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "> ", Placeholder: "enter environment name here", MinWidth: 5})
	if ti.Model.Width != len("enter environment name here") {
		t.Errorf("expected width %d from placeholder, got %d", len("enter environment name here"), ti.Model.Width)
	}
}

func TestNewFilterInputIsFocused(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "> "})

	if !ti.Model.Focused() {
		t.Error("expected textinput to be focused")
	}
}

func TestNewFilterInputSuggestionKeysDisabled(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "> "})

	before := ti.Model.Value()

	_, _ = ti.Update(tea.KeyMsg{Type: tea.KeyUp})
	_, _ = ti.Update(tea.KeyMsg{Type: tea.KeyDown})

	after := ti.Model.Value()
	if before != after {
		t.Errorf("suggestion keys should be disabled, but value changed from %q to %q", before, after)
	}
}

func TestNewFilterInputAcceptsTypedText(t *testing.T) {
	ti := NewFilterInput(TextInputOpts{Prompt: "Filter: ", Placeholder: "placeholder"})

	for _, r := range "hello" {
		_, _ = ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if ti.Model.Value() != "hello" {
		t.Errorf("expected value 'hello', got %q", ti.Model.Value())
	}
}
