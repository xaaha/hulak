package fileselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

// setupTestDir creates a temp directory with files and changes to it.
func setupTestDir(t *testing.T, files []string) func() {
	t.Helper()

	tmpDir := t.TempDir()

	for _, name := range files {
		// Create parent directories if needed
		dir := filepath.Join(tmpDir, filepath.Dir(name))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create the file
		f, err := os.Create(filepath.Join(tmpDir, name))
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return func() {
		err := os.Chdir(oldWd)
		if err != nil {
			t.Errorf("error on setupTestDir: %v", err)
		}
	}
}

// newTestModel creates a Model with items for testing.
func newTestModel(items []string) Model {
	return Model{
		items:    items,
		filtered: items,
		textInput: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Select File: ",
			Placeholder: "",
		}),
	}
}

func TestNewModelWithFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{"collection/get_users.yaml", "collection/post_data.yml"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(m.items))
	}

	expected := map[string]bool{
		filepath.Join("collection", "get_users.yaml"): true,
		filepath.Join("collection", "post_data.yml"):  true,
	}
	for _, item := range m.items {
		if !expected[item] {
			t.Errorf("unexpected item: %s", item)
		}
	}
}

func TestNewModelWithNoFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 0 {
		t.Errorf("expected 0 items, got %d", len(m.items))
	}
}

func TestNewModelFiltersResponseFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{"api.yaml", "api_response.json"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(m.items))
	}
	if m.items[0] != "api.yaml" {
		t.Errorf("expected 'api.yaml', got '%s'", m.items[0])
	}
}

func TestNewModelFiltersJsonFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{"api.yaml", "data.json"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(m.items))
	}
	if m.items[0] != "api.yaml" {
		t.Errorf("expected 'api.yaml', got '%s'", m.items[0])
	}
}

func TestNewModelFiltersEnvDir(t *testing.T) {
	cleanup := setupTestDir(t, []string{"api.yaml", "env/global.env"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(m.items))
	}
	if m.items[0] != "api.yaml" {
		t.Errorf("expected 'api.yaml', got '%s'", m.items[0])
	}
}

func TestNewModelShowsRelativePaths(t *testing.T) {
	cleanup := setupTestDir(t, []string{"collection/get_users.yaml"})
	defer cleanup()

	m := NewModel()

	if len(m.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(m.items))
	}

	if strings.HasPrefix(m.items[0], "/") {
		t.Errorf("expected relative path, got absolute: %s", m.items[0])
	}
	if m.items[0] != filepath.Join("collection", "get_users.yaml") {
		t.Errorf("expected 'collection/get_users.yaml', got '%s'", m.items[0])
	}
}

func TestFormatNoFilesError(t *testing.T) {
	err := FormatNoFilesError()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "no '.yaml' or '.yml' files found") {
		t.Error("error should mention no yaml/yml files found")
	}
	if !strings.Contains(errStr, "Possible solutions") {
		t.Error("error should include possible solutions")
	}
}

func TestQuit(t *testing.T) {
	m := newTestModel([]string{"api.yaml"})

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
	m := newTestModel([]string{"api.yaml"})

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
	m := newTestModel([]string{"api.yaml", "test.yaml"})
	m.textInput.Model.SetValue("test")
	m.applyFilter()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(Model)

	if model.Cancelled {
		t.Error("expected Cancelled to be false - esc should clear filter first")
	}
	if model.textInput.Model.Value() != "" {
		t.Errorf("expected filter to be cleared, got '%s'", model.textInput.Model.Value())
	}
	if cmd != nil {
		t.Error("expected no quit command when clearing filter")
	}
}

func TestSelect(t *testing.T) {
	m := newTestModel([]string{"api.yaml", "test.yaml"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "api.yaml" {
		t.Errorf("expected Selected 'api.yaml', got '%s'", model.Selected)
	}
	if model.Cancelled {
		t.Error("expected Cancelled to be false")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestNavigation(t *testing.T) {
	m := newTestModel([]string{"api.yaml", "test.yaml", "users.yaml"})

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
	m := newTestModel([]string{"api.yaml", "test.yaml", "api_users.yaml"})

	// Type "api"
	for _, r := range "api" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if m.textInput.Model.Value() != "api" {
		t.Errorf("expected filter 'api', got '%s'", m.textInput.Model.Value())
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered items, got %d", len(m.filtered))
	}
}

func TestSelectWithNoMatches(t *testing.T) {
	m := newTestModel([]string{"api.yaml", "test.yaml"})
	m.textInput.Model.SetValue("xyz")
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
	m := newTestModel([]string{"api.yaml", "test.yaml"})

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
	m := newTestModel([]string{"api.yaml", "test.yaml", "users.yaml"})
	m.cursor = 2 // pointing to "users.yaml"

	// Type "a" to filter - only "api.yaml" should remain
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
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
	m := newTestModel([]string{"api.yaml", "test.yaml", "users.yaml"})

	// Filter to "test"
	for _, r := range "test" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "test.yaml" {
		t.Errorf("expected 'test.yaml', got '%s'", model.Selected)
	}
}

func TestCaseInsensitiveFiltering(t *testing.T) {
	m := newTestModel([]string{"api.yaml", "TEST.yaml", "Users.yaml"})

	// Type uppercase "API"
	for _, r := range "API" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	if len(m.filtered) != 1 || m.filtered[0] != "api.yaml" {
		t.Errorf("case insensitive filter failed: expected [api.yaml], got %v", m.filtered)
	}
}

func TestFilterRestorationAfterEsc(t *testing.T) {
	m := newTestModel([]string{"api.yaml", "test.yaml", "users.yaml"})

	// Apply filter
	for _, r := range "api" {
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
	m := newTestModel([]string{"api.yaml", "api_users.yaml", "test.yaml"})

	// Filter to "api" items
	for _, r := range "api" {
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

	if model.Selected != "api_users.yaml" {
		t.Errorf("expected 'api_users.yaml', got '%s'", model.Selected)
	}
}

func TestViewContainsHelp(t *testing.T) {
	m := newTestModel([]string{"api.yaml"})
	view := m.View()

	if !strings.Contains(view, "enter: select") {
		t.Error("view should contain help text")
	}
	if !strings.Contains(view, "esc: cancel") {
		t.Error("view should contain esc help")
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := newTestModel([]string{"api.yaml"})
	view := m.View()

	if !strings.Contains(view, "Select File") {
		t.Error("view should contain title")
	}
}

func TestViewShowsNoMatchesWhenEmpty(t *testing.T) {
	m := newTestModel([]string{"api.yaml"})
	m.textInput.Model.SetValue("xyz")
	m.applyFilter()

	view := m.View()

	if !strings.Contains(view, "(no matches)") {
		t.Error("view should show 'no matches'")
	}
}

func TestInitReturnsBlinkCmd(t *testing.T) {
	m := newTestModel([]string{"api.yaml"})
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a blink command")
	}
}
