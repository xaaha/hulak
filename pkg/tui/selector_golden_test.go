package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSelectorInitialRender(t *testing.T) {
	m := NewSelector([]string{"global", "prod", "staging"}, "Environment: ")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestSelectorAfterNavigation(t *testing.T) {
	m := NewSelector([]string{"global", "prod", "staging"}, "Environment: ")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestSelectorAfterFiltering(t *testing.T) {
	m := NewSelector([]string{"global", "prod", "staging"}, "Environment: ")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Type("pro")
	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestSelectorAfterSelection(t *testing.T) {
	m := NewSelector([]string{"global", "prod", "staging"}, "Environment: ")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))

	result := out.(SelectorModel)
	if result.Selected != "prod" {
		t.Errorf("expected 'prod', got '%s'", result.Selected)
	}
	teatest.RequireEqualOutput(t, []byte(out.View()))
}
