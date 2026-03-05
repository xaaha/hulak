package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var testOptions = []string{"ADMIN", "USER", "MODERATOR"}

func TestNewDropdownDefaults(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	if dd.Label != "role" {
		t.Errorf("Label = %q, want %q", dd.Label, "role")
	}
	if dd.Selected != 0 {
		t.Errorf("Selected = %d, want 0", dd.Selected)
	}
	if dd.Value() != "ADMIN" {
		t.Errorf("Value = %q, want %q", dd.Value(), "ADMIN")
	}
}

func TestNewDropdownWithInitialIndex(t *testing.T) {
	dd := NewDropdown("role", testOptions, 2)
	if dd.Selected != 2 || dd.Value() != "MODERATOR" {
		t.Errorf("Selected = %d, Value = %q; want 2, MODERATOR", dd.Selected, dd.Value())
	}
}

func TestNewDropdownClampsNegativeInitial(t *testing.T) {
	dd := NewDropdown("role", testOptions, -1)
	if dd.Selected != 0 {
		t.Errorf("Selected = %d, want 0 for negative initial", dd.Selected)
	}
}

func TestNewDropdownClampsOverflowInitial(t *testing.T) {
	dd := NewDropdown("role", testOptions, 99)
	if dd.Selected != 0 {
		t.Errorf("Selected = %d, want 0 for overflow initial", dd.Selected)
	}
}

func TestNewDropdownEmptyOptions(t *testing.T) {
	dd := NewDropdown("role", nil, 0)
	if dd.Value() != "" {
		t.Errorf("Value = %q, want empty for nil options", dd.Value())
	}
}

func TestDropdownFocusBlur(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	if dd.Focused() {
		t.Error("expected not focused after creation")
	}

	dd.Focus()
	if !dd.Focused() {
		t.Error("expected focused after Focus()")
	}

	dd.Blur()
	if dd.Focused() {
		t.Error("expected not focused after Blur()")
	}
}

func TestDropdownBlurCollapsesExpanded(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !dd.Expanded() {
		t.Fatal("precondition: expected expanded after Enter")
	}

	dd.Blur()
	if dd.Expanded() {
		t.Error("expected collapsed after Blur")
	}
}

func TestDropdownInitReturnsNil(t *testing.T) {
	dd := NewDropdown("x", testOptions, 0)
	if cmd := dd.Init(); cmd != nil {
		t.Error("expected nil cmd from Init")
	}
}

func TestDropdownExpandOnEnter(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !dd.Expanded() {
		t.Error("expected expanded after Enter")
	}
}

func TestDropdownExpandOnSpace(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()

	dd, _ = dd.Update(keyMsg(KeySpace))
	if !dd.Expanded() {
		t.Error("expected expanded after Space")
	}
}

func TestDropdownExpandSetsCursorToSelected(t *testing.T) {
	dd := NewDropdown("role", testOptions, 2)
	dd.Focus()

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if dd.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (matching Selected)", dd.cursor)
	}
}

func TestDropdownExpandNoopWithEmptyOptions(t *testing.T) {
	dd := NewDropdown("role", nil, 0)
	dd.Focus()
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if dd.Expanded() {
		t.Error("expected no expand with empty options")
	}
}

func TestDropdownSelectOnEnter(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if dd.Expanded() {
		t.Error("expected collapsed after selection")
	}
	if dd.Selected != 1 || dd.Value() != "USER" {
		t.Errorf("Selected = %d, Value = %q; want 1, USER", dd.Selected, dd.Value())
	}
}

func TestDropdownSelectOnSpace(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()

	dd, _ = dd.Update(keyMsg(KeySpace))
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	dd, _ = dd.Update(keyMsg(KeySpace))

	if dd.Selected != 2 || dd.Value() != "MODERATOR" {
		t.Errorf("Selected = %d, Value = %q; want 2, MODERATOR", dd.Selected, dd.Value())
	}
}

func TestDropdownEscCollapsesWithoutChanging(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyDown})
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if dd.Expanded() {
		t.Error("expected collapsed after Esc")
	}
	if dd.Selected != 0 || dd.Value() != "ADMIN" {
		t.Errorf("Selected = %d, Value = %q; want 0, ADMIN (unchanged)", dd.Selected, dd.Value())
	}
}

func TestDropdownCursorNavigation(t *testing.T) {
	tests := []struct {
		name       string
		key        tea.Msg
		startAt    int
		wantCursor int
	}{
		{"down from top", tea.KeyMsg{Type: tea.KeyDown}, 0, 1},
		{"up from middle", tea.KeyMsg{Type: tea.KeyUp}, 1, 0},
		{"ctrl+n", tea.KeyMsg{Type: tea.KeyCtrlN}, 0, 1},
		{"ctrl+p", tea.KeyMsg{Type: tea.KeyCtrlP}, 1, 0},
		{"clamps at bottom", tea.KeyMsg{Type: tea.KeyDown}, 2, 2},
		{"clamps at top", tea.KeyMsg{Type: tea.KeyUp}, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dd := NewDropdown("role", testOptions, tc.startAt)
			dd.Focus()
			dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
			dd.cursor = tc.startAt
			dd, _ = dd.Update(tc.key)
			if dd.cursor != tc.wantCursor {
				t.Errorf("cursor = %d, want %d", dd.cursor, tc.wantCursor)
			}
		})
	}
}

func TestDropdownIgnoresKeysWhenBlurred(t *testing.T) {
	keys := []tea.Msg{
		keyMsg(KeySpace),
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
	}

	for _, k := range keys {
		dd := NewDropdown("role", testOptions, 0)
		dd, _ = dd.Update(k)
		if dd.Expanded() {
			t.Errorf("expected no expand when blurred for %v", k)
		}
	}
}

func TestDropdownCollapsedIgnoresNonExpandKeys(t *testing.T) {
	keys := []tea.Msg{
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyEscape},
		keyMsg("a"),
	}

	for _, k := range keys {
		dd := NewDropdown("role", testOptions, 0)
		dd.Focus()
		dd, _ = dd.Update(k)
		if dd.Expanded() {
			t.Errorf("expected no expand for non-expand key %v", k)
		}
	}
}

func TestDropdownIgnoresNonKeyMessages(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()
	dd, _ = dd.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if dd.Expanded() {
		t.Error("expected no state change for WindowSizeMsg")
	}
}

func TestDropdownViewCollapsed(t *testing.T) {
	dd := NewDropdown("role", testOptions, 1)
	view := dd.View()
	if !strings.Contains(view, "USER") {
		t.Errorf("collapsed view should contain selected value, got: %s", view)
	}
	if !strings.Contains(view, dropdownIndicator) {
		t.Errorf("collapsed view should contain indicator, got: %s", view)
	}
}

func TestDropdownViewExpanded(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := dd.View()
	for _, opt := range testOptions {
		if !strings.Contains(view, opt) {
			t.Errorf("expanded view should contain %q, got: %s", opt, view)
		}
	}

	lines := strings.Split(view, "\n")
	if len(lines) != len(testOptions) {
		t.Errorf("expected %d lines in expanded view, got %d", len(testOptions), len(lines))
	}
}

func TestDropdownViewEmptyOptions(t *testing.T) {
	dd := NewDropdown("role", nil, 0)
	view := dd.View()
	if !strings.Contains(view, "(none)") {
		t.Errorf("empty-options view should contain '(none)', got: %s", view)
	}
}

func TestDropdownUpdateReturnsNilCmd(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	dd.Focus()
	_, cmd := dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd from Update")
	}
}

func TestDropdownExpandedReportsState(t *testing.T) {
	dd := NewDropdown("role", testOptions, 0)
	if dd.Expanded() {
		t.Error("expected not expanded after creation")
	}

	dd.Focus()
	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !dd.Expanded() {
		t.Error("expected expanded after Enter")
	}

	dd, _ = dd.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if dd.Expanded() {
		t.Error("expected not expanded after Esc")
	}
}
