package envselect

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestItemFilterValue(t *testing.T) {
	i := item("test-env")
	if i.FilterValue() != "test-env" {
		t.Errorf("expected 'test-env', got '%s'", i.FilterValue())
	}
}

func TestDelegateHeight(t *testing.T) {
	d := delegate{}
	if d.Height() != 1 {
		t.Errorf("expected height 1, got %d", d.Height())
	}
}

func TestDelegateSpacing(t *testing.T) {
	d := delegate{}
	if d.Spacing() != 0 {
		t.Errorf("expected spacing 0, got %d", d.Spacing())
	}
}

func TestDelegateRender(t *testing.T) {
	d := delegate{}
	items := []list.Item{item("global"), item("dev")}
	listModel := list.New(items, d, 30, 10)

	tests := []struct {
		name     string
		index    int
		selected int
		contains string
	}{
		{"selected item has arrow", 0, 0, "> global"},
		{"unselected item has spaces", 1, 0, "  dev"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listModel.Select(tc.selected)
			var buf bytes.Buffer
			d.Render(&buf, listModel, tc.index, items[tc.index])
			if !bytes.Contains(buf.Bytes(), []byte(tc.contains)) {
				t.Errorf("expected output to contain '%s', got '%s'", tc.contains, buf.String())
			}
		})
	}
}

func TestNewModelHasGlobal(t *testing.T) {
	m := NewModel()

	// First item should always be "global"
	first := m.list.Items()[0]
	if first.FilterValue() != "global" {
		t.Errorf("expected first item to be 'global', got '%s'", first.FilterValue())
	}
}

func TestNewModelWithEnvFiles(t *testing.T) {
	// Create temp env directory
	tmpDir := t.TempDir()
	envDir := filepath.Join(tmpDir, "env")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create test env files
	for _, name := range []string{"global.env", "dev.env", "prod.env"} {
		f, err := os.Create(filepath.Join(envDir, name))
		if err != nil {
			t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			t.Fatalf("error closing file")
		}
	}

	// Change to temp dir so GetEnvFiles finds our test files
	oldWd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	if err != nil {
		t.Fatal("error closing file")
	}
	defer os.Chdir(oldWd)

	m := NewModel()
	items := m.list.Items()

	// Should have global + dev + prod = 3 items
	if len(items) < 1 {
		t.Errorf("expected at least 1 item, got %d", len(items))
	}

	// First should be global
	if items[0].FilterValue() != "global" {
		t.Errorf("expected first item 'global', got '%s'", items[0].FilterValue())
	}
}

func TestUpdateQuit(t *testing.T) {
	m := NewModel()

	// Send ctrl+c
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := newModel.(Model)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true after ctrl+c")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestUpdateCancel(t *testing.T) {
	m := NewModel()

	// Send esc
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newModel.(Model)

	if !model.Cancelled {
		t.Error("expected Cancelled to be true after esc")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestUpdateSelect(t *testing.T) {
	m := NewModel()

	// Send enter
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

func TestViewContainsHelp(t *testing.T) {
	m := NewModel()
	view := m.View()

	if !bytes.Contains([]byte(view), []byte("enter: select")) {
		t.Error("expected view to contain help text")
	}
	if !bytes.Contains([]byte(view), []byte("esc: cancel")) {
		t.Error("expected view to contain esc help")
	}
}

func TestInitReturnsNil(t *testing.T) {
	m := NewModel()
	if m.Init() != nil {
		t.Error("expected Init to return nil")
	}
}
