package envselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// setupTestEnvDir creates a temp directory with env files and changes to it.
// Returns a cleanup function that restores the original working directory.
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

func TestNewModelWithEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "prod.env", "staging.env"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 3 {
		t.Errorf("expected 3 items, got %d", len(m.items))
	}

	// Items should be the env file names without .env suffix
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
		t.Errorf("expected 0 items when no env files exist, got %d", len(m.items))
	}
}

func TestNewModelIgnoresNonEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "readme.txt", "config.yaml"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 1 {
		t.Errorf("expected 1 item (only .env files), got %d", len(m.items))
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
		t.Error("error should mention no environment files found")
	}
	if !strings.Contains(errStr, "Possible solutions") {
		t.Error("error should include possible solutions")
	}
	if !strings.Contains(errStr, "env/dev.env") {
		t.Error("error should suggest creating an env file")
	}
}

func TestUpdateQuit(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := newModel.(Model)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true after ctrl+c")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestUpdateCancelWithEmptyFilter(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(Model)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true after esc with empty filter")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestUpdateCancelClearsFilterFirst(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod"},
		filtered: []string{},
		filter:   "test",
	}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(Model)

	if model.Cancelled {
		t.Error("expected Cancelled to be false - esc should clear filter first")
	}
	if model.filter != "" {
		t.Errorf("expected filter to be cleared, got '%s'", model.filter)
	}
	if cmd != nil {
		t.Error("expected no quit command when clearing filter")
	}
}

func TestUpdateSelect(t *testing.T) {
	m := Model{items: []string{"dev", "prod"}, filtered: []string{"dev", "prod"}}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "dev" {
		t.Errorf("expected Selected to be 'dev', got '%s'", model.Selected)
	}
	if model.Cancelled {
		t.Error("expected Cancelled to be false")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestUpdateNavigation(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod", "staging"},
		filtered: []string{"dev", "prod", "staging"},
	}

	if m.cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", m.cursor)
	}

	// Move down with arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after down, got %d", m.cursor)
	}

	// Move down with ctrl+n
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = newModel.(Model)
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 after ctrl+n, got %d", m.cursor)
	}

	// Move up with arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after up, got %d", m.cursor)
	}

	// Move up with ctrl+p
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 after ctrl+p, got %d", m.cursor)
	}
}

func TestTypingFilters(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod", "development"},
		filtered: []string{"dev", "prod", "development"},
	}

	// Type "dev" - should filter immediately
	for _, r := range "dev" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if m.filter != "dev" {
		t.Errorf("expected filter 'dev', got '%s'", m.filter)
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered items (dev, development), got %d", len(m.filtered))
	}
}

func TestBackspaceRemovesFilterChar(t *testing.T) {
	m := Model{
		items:  []string{"dev", "test"},
		filter: "test",
	}
	m.applyFilter()

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(Model)

	if m.filter != "tes" {
		t.Errorf("expected filter 'tes', got '%s'", m.filter)
	}
}

func TestCtrlWDeletesLastWord(t *testing.T) {
	m := Model{
		items:  []string{"dev", "hello world test"},
		filter: "hello world",
	}
	m.applyFilter()

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	m = newModel.(Model)

	if m.filter != "hello " {
		t.Errorf("expected filter 'hello ', got '%s'", m.filter)
	}

	// Delete another word
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	m = newModel.(Model)

	if m.filter != "" {
		t.Errorf("expected filter '', got '%s'", m.filter)
	}
}

func TestCtrlUClearsFilter(t *testing.T) {
	m := Model{
		items:  []string{"dev", "test"},
		filter: "hello world",
	}
	m.applyFilter()

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = newModel.(Model)

	if m.filter != "" {
		t.Errorf("expected filter to be empty, got '%s'", m.filter)
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected all items after clearing filter, got %d", len(m.filtered))
	}
}

func TestViewContainsHelp(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}}
	view := m.View()

	if !strings.Contains(view, "enter: select") {
		t.Error("expected view to contain help text")
	}
	if !strings.Contains(view, "esc: cancel") {
		t.Error("expected view to contain esc help")
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}}
	view := m.View()

	if !strings.Contains(view, "Select Environment") {
		t.Error("expected view to contain title")
	}
}

func TestViewShowsCursor(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}}
	view := m.View()

	if !strings.Contains(view, "█") {
		t.Error("expected view to show cursor")
	}
}

func TestViewShowsFilterText(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}, filter: "test"}
	view := m.View()

	if !strings.Contains(view, "test") {
		t.Error("expected view to show filter text")
	}
}

func TestViewHasBorder(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{"dev"}}
	view := m.View()

	// Rounded border uses these characters
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("expected view to have rounded border")
	}
}

func TestViewShowsNoMatchesWhenFilteredEmpty(t *testing.T) {
	m := Model{items: []string{"dev"}, filtered: []string{}, filter: "xyz"}
	view := m.View()

	if !strings.Contains(view, "(no matches)") {
		t.Error("expected view to show 'no matches' when filtered list is empty")
	}
}

func TestInitReturnsNil(t *testing.T) {
	m := Model{}
	if m.Init() != nil {
		t.Error("expected Init to return nil")
	}
}

// Edge case tests

func TestSelectWithNoMatches(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod"},
		filtered: []string{},
		filter:   "xyz",
	}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	// Should quit but with empty Selected
	if model.Selected != "" {
		t.Errorf("expected empty Selected when no matches, got '%s'", model.Selected)
	}
	if cmd == nil {
		t.Error("expected quit command even with no matches")
	}
}

func TestNavigationBoundsAtTop(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod"},
		filtered: []string{"dev", "prod"},
		cursor:   0,
	}

	// Try to move up when already at top
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := newModel.(Model)

	if model.cursor != 0 {
		t.Errorf("cursor should stay at 0 when at top, got %d", model.cursor)
	}
}

func TestNavigationBoundsAtBottom(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod"},
		filtered: []string{"dev", "prod"},
		cursor:   1, // at bottom
	}

	// Try to move down when already at bottom
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := newModel.(Model)

	if model.cursor != 1 {
		t.Errorf("cursor should stay at 1 when at bottom, got %d", model.cursor)
	}
}

func TestCursorAdjustsWhenFilterReducesList(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod", "staging"},
		filtered: []string{"dev", "prod", "staging"},
		cursor:   2, // pointing to "staging"
	}

	// Type "d" to filter - only "dev" should remain
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := newModel.(Model)

	// Cursor should adjust to valid position
	if model.cursor >= len(model.filtered) {
		t.Errorf("cursor %d should be less than filtered length %d", model.cursor, len(model.filtered))
	}
}

func TestSelectCorrectItemAfterFiltering(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod", "staging"},
		filtered: []string{"dev", "prod", "staging"},
	}

	// Filter to show only "prod"
	for _, r := range "prod" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	// Select
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "prod" {
		t.Errorf("expected 'prod' to be selected, got '%s'", model.Selected)
	}
}

func TestSingleItemList(t *testing.T) {
	m := Model{
		items:    []string{"dev"},
		filtered: []string{"dev"},
	}

	// Navigate down should stay at 0
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := newModel.(Model)
	if model.cursor != 0 {
		t.Errorf("cursor should stay at 0 for single item, got %d", model.cursor)
	}

	// Select should work
	newModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newModel.(Model)
	if model.Selected != "dev" {
		t.Errorf("expected 'dev', got '%s'", model.Selected)
	}
}

func TestCaseInsensitiveFiltering(t *testing.T) {
	m := Model{
		items:    []string{"dev", "PROD", "Staging"},
		filtered: []string{"dev", "PROD", "Staging"},
	}

	// Type uppercase "DEV"
	for _, r := range "DEV" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if len(m.filtered) != 1 || m.filtered[0] != "dev" {
		t.Errorf("case insensitive filter failed: expected [dev], got %v", m.filtered)
	}

	// Clear and try lowercase on uppercase item
	m.filter = ""
	m.applyFilter()

	for _, r := range "prod" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if len(m.filtered) != 1 || m.filtered[0] != "PROD" {
		t.Errorf("case insensitive filter failed: expected [PROD], got %v", m.filtered)
	}
}

func TestBackspaceOnEmptyFilter(t *testing.T) {
	m := Model{
		items:    []string{"dev"},
		filtered: []string{"dev"},
		filter:   "",
	}

	// Should not panic or change anything
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model := newModel.(Model)

	if model.filter != "" {
		t.Errorf("filter should remain empty, got '%s'", model.filter)
	}
}

func TestCtrlWOnEmptyFilter(t *testing.T) {
	m := Model{
		items:    []string{"dev"},
		filtered: []string{"dev"},
		filter:   "",
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	model := newModel.(Model)

	if model.filter != "" {
		t.Errorf("filter should remain empty, got '%s'", model.filter)
	}
}

func TestCtrlUOnEmptyFilter(t *testing.T) {
	m := Model{
		items:    []string{"dev"},
		filtered: []string{"dev"},
		filter:   "",
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	model := newModel.(Model)

	if model.filter != "" {
		t.Errorf("filter should remain empty, got '%s'", model.filter)
	}
}

func TestFilterRestorationAfterEsc(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod", "staging"},
		filtered: []string{"dev", "prod", "staging"},
	}

	// Apply a filter
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

	// All items should be visible again
	if len(m.filtered) != 3 {
		t.Errorf("expected all 3 items after esc, got %d", len(m.filtered))
	}
	if m.filter != "" {
		t.Errorf("filter should be empty after esc, got '%s'", m.filter)
	}
}

func TestSelectedItemHasArrowInView(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod"},
		filtered: []string{"dev", "prod"},
		cursor:   1, // prod selected
	}

	view := m.View()

	// The selected item (prod) should have ">" prefix
	if !strings.Contains(view, ">") {
		t.Error("view should contain '>' for selected item")
	}
}

func TestNavigateAndSelectSecondItem(t *testing.T) {
	m := Model{
		items:    []string{"dev", "prod", "staging"},
		filtered: []string{"dev", "prod", "staging"},
	}

	// Move down to prod
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	// Select
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "prod" {
		t.Errorf("expected 'prod', got '%s'", model.Selected)
	}
}

func TestFilterThenNavigateThenSelect(t *testing.T) {
	m := Model{
		items:    []string{"dev", "development", "prod"},
		filtered: []string{"dev", "development", "prod"},
	}

	// Filter to "dev" items
	for _, r := range "dev" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	// Should have 2 items: dev, development
	if len(m.filtered) != 2 {
		t.Fatalf("expected 2 filtered items, got %d", len(m.filtered))
	}

	// Navigate to second item (development)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	// Select
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "development" {
		t.Errorf("expected 'development', got '%s'", model.Selected)
	}
}
