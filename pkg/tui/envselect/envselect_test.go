package envselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelHasGlobal(t *testing.T) {
	m := NewModel()

	if len(m.items) == 0 {
		t.Fatal("expected at least one item")
	}
	if m.items[0] != "global" {
		t.Errorf("expected first item to be 'global', got '%s'", m.items[0])
	}
}

func TestNewModelWithEnvFiles(t *testing.T) {
	tmpDir := t.TempDir()
	envDir := filepath.Join(tmpDir, "env")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"global.env", "dev.env", "prod.env"} {
		f, err := os.Create(filepath.Join(envDir, name))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	m := NewModel()

	if len(m.items) < 1 {
		t.Errorf("expected at least 1 item, got %d", len(m.items))
	}
	if m.items[0] != "global" {
		t.Errorf("expected first item 'global', got '%s'", m.items[0])
	}
}

func TestUpdateQuit(t *testing.T) {
	m := NewModel()

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
	m := NewModel()

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
	m := NewModel()
	m.filter = "test"
	m.filtered = []string{}

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
	m := NewModel()

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newModel.(Model)

	if model.Selected != "global" {
		t.Errorf("expected Selected to be 'global', got '%s'", model.Selected)
	}
	if model.Cancelled {
		t.Error("expected Cancelled to be false")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestUpdateNavigation(t *testing.T) {
	m := NewModel()
	m.items = []string{"global", "dev", "prod"}
	m.filtered = m.items

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
	m := NewModel()
	m.items = []string{"global", "dev", "prod", "development"}
	m.filtered = m.items

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
	m := NewModel()
	m.filter = "test"
	m.items = []string{"global", "test"}
	m.applyFilter()

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(Model)

	if m.filter != "tes" {
		t.Errorf("expected filter 'tes', got '%s'", m.filter)
	}
}

func TestCtrlWDeletesLastWord(t *testing.T) {
	m := NewModel()
	m.items = []string{"global", "hello world test"}
	m.filter = "hello world"
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
	m := NewModel()
	m.items = []string{"global", "test"}
	m.filter = "hello world"
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
	m := NewModel()
	view := m.View()

	if !strings.Contains(view, "enter: select") {
		t.Error("expected view to contain help text")
	}
	if !strings.Contains(view, "esc: cancel") {
		t.Error("expected view to contain esc help")
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := NewModel()
	view := m.View()

	if !strings.Contains(view, "Select Environment") {
		t.Error("expected view to contain title")
	}
}

func TestViewShowsCursor(t *testing.T) {
	m := NewModel()
	view := m.View()

	if !strings.Contains(view, "█") {
		t.Error("expected view to show cursor")
	}
}

func TestViewShowsFilterText(t *testing.T) {
	m := NewModel()
	m.filter = "test"
	view := m.View()

	if !strings.Contains(view, "test") {
		t.Error("expected view to show filter text")
	}
}

func TestViewHasBorder(t *testing.T) {
	m := NewModel()
	view := m.View()

	// Rounded border uses these characters
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("expected view to have rounded border")
	}
}

func TestInitReturnsNil(t *testing.T) {
	m := NewModel()
	if m.Init() != nil {
		t.Error("expected Init to return nil")
	}
}
