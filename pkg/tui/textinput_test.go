package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewFilterInputPrompt(t *testing.T) {
	ti := NewFilterInput("Filter: ", "")

	if ti.Model.Prompt != "Filter: " {
		t.Errorf("expected prompt 'Filter: ', got %q", ti.Model.Prompt)
	}
}

func TestNewFilterInputEmptyPlaceholder(t *testing.T) {
	ti := NewFilterInput("Filter: ", "")

	if ti.Model.Placeholder != "" {
		t.Errorf("expected empty placeholder, got %q", ti.Model.Placeholder)
	}
	if ti.Model.Width != 0 {
		t.Errorf("expected width 0 for empty placeholder, got %d", ti.Model.Width)
	}
}

func TestNewFilterInputCustomPlaceholder(t *testing.T) {
	ti := NewFilterInput("Search: ", "type to search")

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
			ti := NewFilterInput("> ", tc.placeholder)
			if ti.Model.Width != tc.wantWidth {
				t.Errorf("placeholder %q: expected width %d, got %d", tc.placeholder, tc.wantWidth, ti.Model.Width)
			}
		})
	}
}

func TestNewFilterInputIsFocused(t *testing.T) {
	ti := NewFilterInput("> ", "")

	if !ti.Model.Focused() {
		t.Error("expected textinput to be focused")
	}
}

func TestNewFilterInputSuggestionKeysDisabled(t *testing.T) {
	ti := NewFilterInput("> ", "")

	before := ti.Model.Value()

	ti, _ = ti.Update(tea.KeyMsg{Type: tea.KeyUp})
	ti, _ = ti.Update(tea.KeyMsg{Type: tea.KeyDown})

	after := ti.Model.Value()
	if before != after {
		t.Errorf("suggestion keys should be disabled, but value changed from %q to %q", before, after)
	}
}

func TestNewFilterInputAcceptsTypedText(t *testing.T) {
	ti := NewFilterInput("Filter: ", "placeholder")

	for _, r := range "hello" {
		ti, _ = ti.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if ti.Model.Value() != "hello" {
		t.Errorf("expected value 'hello', got %q", ti.Model.Value())
	}
}
