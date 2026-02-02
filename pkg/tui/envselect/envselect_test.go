package envselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

// setupTestEnvDir creates a temp directory with env files and changes to it.
func setupTestEnvDir(t *testing.T, envFiles []string) func() {
	t.Helper()

	tmpDir := t.TempDir()
	envDir := filepath.Join(tmpDir, "env")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range envFiles {
		f, err := os.Create(filepath.Join(envDir, name))
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	return func() { os.Chdir(oldWd) }
}

// newTestModel creates a Model with items for testing.
func newTestModel(items []string) Model {
	return Model{
		items:     items,
		filtered:  items,
		textInput: tui.NewFilterInput(),
	}
}

func TestNewModelWithEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "prod.env", "staging.env"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 3 {
		t.Errorf("expected 3 items, got %d", len(m.items))
	}

	expected := map[string]bool{"dev": true, "prod": true, "staging": true}
	for _, item := range m.items {
		if !expected[item] {
			t.Errorf("unexpected item: %s", item)
		}
	}
}

func TestNewModelWithNoEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 0 {
		t.Errorf("expected 0 items, got %d", len(m.items))
	}
}

func TestNewModelIgnoresNonEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "readme.txt", "config.yaml"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(m.items))
	}
	if m.items[0] != "dev" {
		t.Errorf("expected 'dev', got '%s'", m.items[0])
	}
}

func TestFormatNoEnvFilesError(t *testing.T) {
	err := FormatNoEnvFilesError()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "no '.env' files found") {
		t.Error("error should mention no env files found")
	}
	if !strings.Contains(errStr, "Possible solutions") {
		t.Error("error should include possible solutions")
	}
}

func TestQuit(t *testing.T) {
	m := newTestModel([]string{"dev"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := newModel.(Model)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestCancelWithEmptyFilter(t *testing.T) {
	m := newTestModel([]string{"dev"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(Model)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestCancelClearsFilterFirst(t *testing.T) {
	m := newTestModel([]string{"dev", "prod"})
	m.textInput.SetValue("test")
	m.applyFilter()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(Model)

	if model.Cancelled {
		t.Error("expected Cancelled to be false - esc should clear filter first")
	}
	if model.textInput.Value() != "" {
		t.Errorf("expected filter to be cleared, got '%s'", model.textInput.Value())
	}
	if cmd != nil {
		t.Error("expected no quit command when clearing filter")
	}
}

func TestSelect(t *testing.T) {
	m := newTestModel([]string{"dev", "prod"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "dev" {
		t.Errorf("expected Selected 'dev', got '%s'", model.Selected)
	}
	if model.Cancelled {
		t.Error("expected Cancelled to be false")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestNavigation(t *testing.T) {
	m := newTestModel([]string{"dev", "prod", "staging"})

	if m.cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", m.cursor)
	}

	// Move down with arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}

	// Move down with ctrl+n
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = newModel.(Model)
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}

	// Move up with arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}

	// Move up with ctrl+p
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestTypingFilters(t *testing.T) {
	m := newTestModel([]string{"dev", "prod", "development"})

	// Type "dev"
	for _, r := range "dev" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if m.textInput.Value() != "dev" {
		t.Errorf("expected filter 'dev', got '%s'", m.textInput.Value())
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered items, got %d", len(m.filtered))
	}
}

func TestSelectWithNoMatches(t *testing.T) {
	m := newTestModel([]string{"dev", "prod"})
	m.textInput.SetValue("xyz")
	m.applyFilter()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "" {
		t.Errorf("expected empty Selected, got '%s'", model.Selected)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestNavigationBounds(t *testing.T) {
	m := newTestModel([]string{"dev", "prod"})

	// At top, can't go up
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Go to bottom
	m.cursor = 1
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor should stay at 1, got %d", m.cursor)
	}
}

func TestCursorAdjustsWhenFilterReducesList(t *testing.T) {
	m := newTestModel([]string{"dev", "prod", "staging"})
	m.cursor = 2 // pointing to "staging"

	// Type "d" to filter - only "dev" should remain
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := newModel.(Model)

	if model.cursor >= len(model.filtered) {
		t.Errorf(
			"cursor %d should be less than filtered length %d",
			model.cursor,
			len(model.filtered),
		)
	}
}

func TestSelectAfterFiltering(t *testing.T) {
	m := newTestModel([]string{"dev", "prod", "staging"})

	// Filter to "prod"
	for _, r := range "prod" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "prod" {
		t.Errorf("expected 'prod', got '%s'", model.Selected)
	}
}

func TestSingleItemList(t *testing.T) {
	m := newTestModel([]string{"dev"})

	// Navigate down should stay at 0
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Select should work
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)
	if model.Selected != "dev" {
		t.Errorf("expected 'dev', got '%s'", model.Selected)
	}
}

func TestCaseInsensitiveFiltering(t *testing.T) {
	m := newTestModel([]string{"dev", "PROD", "Staging"})

	// Type uppercase "DEV"
	for _, r := range "DEV" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if len(m.filtered) != 1 || m.filtered[0] != "dev" {
		t.Errorf("case insensitive filter failed: expected [dev], got %v", m.filtered)
	}
}

func TestFilterRestorationAfterEsc(t *testing.T) {
	m := newTestModel([]string{"dev", "prod", "staging"})

	// Apply filter
	for _, r := range "dev" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if len(m.filtered) == 3 {
		t.Error("filter should have reduced the list")
	}

	// Press esc to clear filter
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	if len(m.filtered) != 3 {
		t.Errorf("expected all 3 items after esc, got %d", len(m.filtered))
	}
}

func TestFilterThenNavigateThenSelect(t *testing.T) {
	m := newTestModel([]string{"dev", "development", "prod"})

	// Filter to "dev" items
	for _, r := range "dev" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if len(m.filtered) != 2 {
		t.Fatalf("expected 2 filtered items, got %d", len(m.filtered))
	}

	// Navigate to second item
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	// Select
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "development" {
		t.Errorf("expected 'development', got '%s'", model.Selected)
	}
}

func TestViewContainsHelp(t *testing.T) {
	m := newTestModel([]string{"dev"})
	view := m.View()

	if !strings.Contains(view, "enter: select") {
		t.Error("view should contain help text")
	}
	if !strings.Contains(view, "esc: cancel") {
		t.Error("view should contain esc help")
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := newTestModel([]string{"dev"})
	view := m.View()

	if !strings.Contains(view, "Select Environment") {
		t.Error("view should contain title")
	}
}

func TestViewShowsNoMatchesWhenEmpty(t *testing.T) {
	m := newTestModel([]string{"dev"})
	m.textInput.SetValue("xyz")
	m.applyFilter()

	view := m.View()

	if !strings.Contains(view, "(no matches)") {
		t.Error("view should show 'no matches'")
	}
}

func TestViewHasBorder(t *testing.T) {
	m := newTestModel([]string{"dev"})
	view := m.View()

	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("view should have rounded border")
	}
}

func TestSelectedItemHasArrow(t *testing.T) {
	m := newTestModel([]string{"dev", "prod"})
	m.cursor = 1

	view := m.View()

	if !strings.Contains(view, ">") {
		t.Error("view should contain '>' for selected item")
	}
}

func TestInitReturnsBlinkCmd(t *testing.T) {
	m := newTestModel([]string{"dev"})
	cmd := m.Init()

	if cmd != nil {
		t.Error("Init should return nil")
	}
}
