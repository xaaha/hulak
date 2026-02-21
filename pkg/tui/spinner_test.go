package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSpinnerMessage(t *testing.T) {
	s := newSpinner("Loading...")

	if s.Message != "Loading..." {
		t.Errorf("expected message 'Loading...', got %q", s.Message)
	}
}

func TestNewSpinnerEmptyMessage(t *testing.T) {
	s := newSpinner("")

	if s.Message != "" {
		t.Errorf("expected empty message, got %q", s.Message)
	}
}

func TestNewSpinnerUsesDotStyle(t *testing.T) {
	s := newSpinner("test")

	dotSpinner := spinner.Dot
	if s.Model.Spinner.FPS != dotSpinner.FPS {
		t.Errorf("expected Dot spinner FPS %v, got %v", dotSpinner.FPS, s.Model.Spinner.FPS)
	}
}

func TestSpinnerInitReturnsCmd(t *testing.T) {
	s := newSpinner("Loading...")
	cmd := s.Init()

	if cmd == nil {
		t.Error("Init should return a tick command")
	}
}

func TestSpinnerUpdateReturnsCmd(t *testing.T) {
	s := newSpinner("Loading...")

	updated, cmd := s.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Error("Update with TickMsg should return a command")
	}
	if updated.Message != "Loading..." {
		t.Errorf("message should be preserved after update, got %q", updated.Message)
	}
}

func TestSpinnerViewContainsMessage(t *testing.T) {
	s := newSpinner("Fetching schemas...")
	view := s.View()

	if !strings.Contains(view, "Fetching schemas...") {
		t.Errorf("view should contain message, got %q", view)
	}
}

func TestSpinnerViewDifferentMessages(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"short", "Loading..."},
		{"long", "Fetching GraphQL schemas from remote endpoints..."},
		{"empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newSpinner(tc.message)
			view := s.View()
			if !strings.Contains(view, tc.message) {
				t.Errorf("view should contain %q, got %q", tc.message, view)
			}
		})
	}
}

func TestSpinnerTaskModelInitReturnsBatchCmd(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("test"),
		task:    func() (any, error) { return "done", nil },
	}
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a batched command")
	}
}

func TestSpinnerTaskModeltaskDoneMsgSetsResult(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("test"),
		task:    func() (any, error) { return nil, nil },
	}

	updated, cmd := m.Update(taskDoneMsg{Result: "hello", Err: nil})
	model := updated.(spinnerTaskModel)

	if model.result != "hello" {
		t.Errorf("expected result 'hello', got %v", model.result)
	}
	if model.err != nil {
		t.Errorf("expected nil error, got %v", model.err)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSpinnerTaskModeltaskDoneMsgSetsError(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("test"),
		task:    func() (any, error) { return nil, nil },
	}
	taskErr := fmt.Errorf("connection failed")

	updated, cmd := m.Update(taskDoneMsg{Result: nil, Err: taskErr})
	model := updated.(spinnerTaskModel)

	if model.err == nil || model.err.Error() != "connection failed" {
		t.Errorf("expected error 'connection failed', got %v", model.err)
	}
	if model.result != nil {
		t.Errorf("expected nil result, got %v", model.result)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSpinnerTaskModelCtrlCInterrupts(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("test"),
		task:    func() (any, error) { return nil, nil },
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := updated.(spinnerTaskModel)

	if model.err == nil || model.err.Error() != "interrupted" {
		t.Errorf("expected 'interrupted' error, got %v", model.err)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSpinnerTaskModelNonQuitKeyDoesNotQuit(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("test"),
		task:    func() (any, error) { return nil, nil },
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model := updated.(spinnerTaskModel)

	if model.err != nil {
		t.Errorf("non-quit key should not set error, got %v", model.err)
	}
	if model.result != nil {
		t.Errorf("non-quit key should not set result, got %v", model.result)
	}
}

func TestSpinnerTaskModelTickUpdatesSpinner(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("test"),
		task:    func() (any, error) { return nil, nil },
	}

	_, cmd := m.Update(spinner.TickMsg{})

	if cmd == nil {
		t.Error("tick message should produce a follow-up command")
	}
}

func TestSpinnerTaskModelViewContainsMessage(t *testing.T) {
	m := spinnerTaskModel{
		spinner: newSpinner("Fetching schemas..."),
		task:    func() (any, error) { return nil, nil },
	}
	view := m.View()

	if !strings.Contains(view, "Fetching schemas...") {
		t.Errorf("view should contain spinner message, got %q", view)
	}
}
