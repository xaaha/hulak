package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/utils"
)

func TestSelectorQuit(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := newModel.(*SelectorModel)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSelectorCancelWithEmptyFilter(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(*SelectorModel)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSelectorCancelClearsFilterFirst(t *testing.T) {
	m := NewSelector([]string{"item1", "item2"}, "Test: ")
	m.TextInput.Model.SetValue("test")
	m.applyFilter()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(*SelectorModel)

	if model.Cancelled {
		t.Error("expected Cancelled to be false - esc should clear filter first")
	}
	if model.TextInput.Model.Value() != "" {
		t.Errorf("expected filter to be cleared, got '%s'", model.TextInput.Model.Value())
	}
	if cmd != nil {
		t.Error("expected no quit command when clearing filter")
	}
}

func TestSelectorSelect(t *testing.T) {
	m := NewSelector([]string{"item1", "item2"}, "Test: ")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(*SelectorModel)

	if model.Selected != "item1" {
		t.Errorf("expected Selected 'item1', got '%s'", model.Selected)
	}
	if model.Cancelled {
		t.Error("expected Cancelled to be false")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSelectorNavigation(t *testing.T) {
	m := NewSelector([]string{"item1", "item2", "item3"}, "Test: ")

	if m.Cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", m.Cursor)
	}

	// Move down with arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.Cursor)
	}

	// Move down with ctrl+n
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.Cursor)
	}

	// Move up with arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.Cursor)
	}

	// Move up with ctrl+p
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.Cursor)
	}
}

func TestSelectorTypingFilters(t *testing.T) {
	m := NewSelector([]string{"item1", "test1", "item_test"}, "Test: ")

	// Type "item"
	for _, r := range "item" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *newModel.(*SelectorModel)
	}

	if m.TextInput.Model.Value() != "item" {
		t.Errorf("expected filter 'item', got '%s'", m.TextInput.Model.Value())
	}
	if len(m.Filtered) != 2 {
		t.Errorf("expected 2 filtered items, got %d", len(m.Filtered))
	}
}

func TestSelectorSelectWithNoMatches(t *testing.T) {
	m := NewSelector([]string{"item1", "item2"}, "Test: ")
	m.TextInput.Model.SetValue("xyz")
	m.applyFilter()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(*SelectorModel)

	if model.Selected != "" {
		t.Errorf("expected empty Selected, got '%s'", model.Selected)
	}
	if cmd != nil {
		t.Error("expected no quit command when there are no matches")
	}
}

func TestSelectorNavigationBounds(t *testing.T) {
	m := NewSelector([]string{"item1", "item2"}, "Test: ")

	// At top, can't go up
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.Cursor)
	}

	// Go to bottom
	m.Cursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 1 {
		t.Errorf("cursor should stay at 1, got %d", m.Cursor)
	}
}

func TestSelectorCursorAdjustsWhenFilterReducesList(t *testing.T) {
	m := NewSelector([]string{"item1", "item2", "item3"}, "Test: ")
	m.Cursor = 2 // pointing to "item3"

	// Type "1" to filter - only "item1" should remain
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model := newModel.(*SelectorModel)

	if model.Cursor >= len(model.Filtered) {
		t.Errorf(
			"cursor %d should be less than filtered length %d",
			model.Cursor,
			len(model.Filtered),
		)
	}
}

func TestSelectorSelectAfterFiltering(t *testing.T) {
	m := NewSelector([]string{"item1", "item2", "item3"}, "Test: ")

	// Filter to "item2"
	for _, r := range "item2" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *newModel.(*SelectorModel)
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(*SelectorModel)

	if model.Selected != "item2" {
		t.Errorf("expected 'item2', got '%s'", model.Selected)
	}
}

func TestSelectorCaseInsensitiveFiltering(t *testing.T) {
	m := NewSelector([]string{"item1", "ITEM2", "Item3"}, "Test: ")

	// Type uppercase "ITEM"
	for _, r := range "ITEM" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *newModel.(*SelectorModel)
	}

	if len(m.Filtered) != 3 {
		t.Errorf("case insensitive filter failed: expected 3 items, got %d", len(m.Filtered))
	}
}

func TestSelectorFilterRestorationAfterEsc(t *testing.T) {
	m := NewSelector([]string{"item1", "item2", "item3"}, "Test: ")

	// Apply filter
	for _, r := range "item1" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *newModel.(*SelectorModel)
	}

	if len(m.Filtered) == 3 {
		t.Error("filter should have reduced the list")
	}

	// Press esc to clear filter
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = *newModel.(*SelectorModel)

	if len(m.Filtered) != 3 {
		t.Errorf("expected all 3 items after esc, got %d", len(m.Filtered))
	}
}

func TestSelectorFilterThenNavigateThenSelect(t *testing.T) {
	m := NewSelector([]string{"item1", "item1_extra", "item2"}, "Test: ")

	// Filter to "item1" items
	for _, r := range "item1" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *newModel.(*SelectorModel)
	}

	if len(m.Filtered) != 2 {
		t.Fatalf("expected 2 filtered items, got %d", len(m.Filtered))
	}

	// Navigate to second item
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = *newModel.(*SelectorModel)

	// Select
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(*SelectorModel)

	if model.Selected != "item1_extra" {
		t.Errorf("expected 'item1_extra', got '%s'", model.Selected)
	}
}

func TestSelectorViewContainsHelp(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")
	view := m.View()

	if !strings.Contains(view, "enter: select") {
		t.Error("view should contain help text")
	}
	if !strings.Contains(view, "esc: cancel") {
		t.Error("view should contain esc help")
	}
}

func TestSelectorViewShowsNoMatchesWhenEmpty(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")
	m.TextInput.Model.SetValue("xyz")
	m.applyFilter()

	view := m.View()

	if !strings.Contains(view, "(no matches)") {
		t.Error("view should show 'no matches'")
	}
}

func TestSelectorInitReturnsBlinkCmd(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a blink command")
	}
}

func TestSelectorSingleItemList(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")

	// Navigate down should stay at 0
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = *newModel.(*SelectorModel)
	if m.Cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.Cursor)
	}

	// Select should work
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(*SelectorModel)
	if model.Selected != "item1" {
		t.Errorf("expected 'item1', got '%s'", model.Selected)
	}
}

func TestSelectorViewHasBorder(t *testing.T) {
	m := NewSelector([]string{"item1"}, "Test: ")
	view := m.View()

	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("view should have rounded border")
	}
}

func TestSelectorSelectedItemHasArrow(t *testing.T) {
	m := NewSelector([]string{"item1", "item2"}, "Test: ")
	m.Cursor = 1

	view := m.View()

	if !strings.Contains(view, utils.ChevronRight) {
		t.Error("view should contain chevron for selected item")
	}
}

func TestSelectorViewportHeightIsCapped(t *testing.T) {
	m := NewSelector([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}, "Test: ")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 80})
	m = *updated.(*SelectorModel)

	if m.viewport.Height > selectorViewportMaxH {
		t.Fatalf("expected viewport height <= %d, got %d", selectorViewportMaxH, m.viewport.Height)
	}
}

func TestSelectorViewportScrollsOnLongLists(t *testing.T) {
	items := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	m := NewSelector(items, "Test: ")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 10})
	m = *updated.(*SelectorModel)

	for i := 0; i < 10; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = *updated.(*SelectorModel)
	}

	if m.viewport.YOffset == 0 {
		t.Fatal("expected viewport to scroll for long list navigation")
	}
}
