package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func TestNewToggleDefaultOff(t *testing.T) {
	tog := NewToggle("enabled", false)
	if tog.Value {
		t.Error("expected Value false")
	}
	if tog.Label != "enabled" {
		t.Errorf("expected label 'enabled', got %q", tog.Label)
	}
}

func TestNewToggleDefaultOn(t *testing.T) {
	tog := NewToggle("enabled", true)
	if !tog.Value {
		t.Error("expected Value true")
	}
}

func TestToggleFocusBlur(t *testing.T) {
	tog := NewToggle("f", false)
	if tog.Focused() {
		t.Error("expected not focused after creation")
	}

	tog.Focus()
	if !tog.Focused() {
		t.Error("expected focused after Focus()")
	}

	tog.Blur()
	if tog.Focused() {
		t.Error("expected not focused after Blur()")
	}
}

func TestToggleInitReturnsNil(t *testing.T) {
	tog := NewToggle("x", false)
	if cmd := tog.Init(); cmd != nil {
		t.Error("expected nil cmd from Init")
	}
}

func TestToggleSpaceWhenFocused(t *testing.T) {
	tog := NewToggle("v", false)
	tog.Focus()

	tog, _ = tog.Update(keyMsg(KeySpace))
	if !tog.Value {
		t.Error("expected Value true after Space")
	}

	tog, _ = tog.Update(keyMsg(KeySpace))
	if tog.Value {
		t.Error("expected Value false after second Space")
	}
}

func TestToggleEnterWhenFocused(t *testing.T) {
	tog := NewToggle("v", false)
	tog.Focus()

	tog, _ = tog.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !tog.Value {
		t.Error("expected Value true after Enter")
	}
}

func TestToggleIgnoresKeysWhenBlurred(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"space", keyMsg(KeySpace)},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"arrow", tea.KeyMsg{Type: tea.KeyUp}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tog := NewToggle("v", false)
			tog, _ = tog.Update(tc.msg)
			if tog.Value {
				t.Errorf("expected no toggle when blurred, got Value=true for %s", tc.name)
			}
		})
	}
}

func TestToggleIgnoresNonToggleKeysWhenFocused(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"up", tea.KeyMsg{Type: tea.KeyUp}},
		{"down", tea.KeyMsg{Type: tea.KeyDown}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}},
		{"letter", keyMsg("a")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tog := NewToggle("v", false)
			tog.Focus()
			tog, _ = tog.Update(tc.msg)
			if tog.Value {
				t.Errorf("expected no toggle for %s, got Value=true", tc.name)
			}
		})
	}
}

func TestToggleIgnoresNonKeyMessages(t *testing.T) {
	tog := NewToggle("v", false)
	tog.Focus()
	tog, _ = tog.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if tog.Value {
		t.Error("expected no toggle for WindowSizeMsg")
	}
}

func TestToggleViewContainsLabel(t *testing.T) {
	tog := NewToggle("assignedOnly", false)
	view := tog.View()
	if !strings.Contains(view, "assignedOnly") {
		t.Errorf("expected label in view, got: %s", view)
	}
}

func TestToggleViewShowsCheckState(t *testing.T) {
	tests := []struct {
		name    string
		value   bool
		wantOn  string
		wantOff string
	}{
		{"off", false, "[x]", "[ ]"},
		{"on", true, "[x]", "[ ]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tog := NewToggle("flag", tc.value)
			view := tog.View()
			if tc.value {
				if !strings.Contains(view, "x") {
					t.Errorf("expected 'x' in on-state view, got: %s", view)
				}
			} else {
				if strings.Contains(view, "x") {
					t.Errorf("expected no 'x' in off-state view, got: %s", view)
				}
			}
		})
	}
}

func TestToggleUpdateReturnsNilCmd(t *testing.T) {
	tog := NewToggle("v", false)
	tog.Focus()
	_, cmd := tog.Update(keyMsg(KeySpace))
	if cmd != nil {
		t.Error("expected nil cmd from Update")
	}
}
