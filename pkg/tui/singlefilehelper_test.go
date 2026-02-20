package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSingleFileHelperStartsWithEnvFocus(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev", "prod"}, []string{"a.yaml"}, "", false)

	if m.focus != focusEnv {
		t.Fatalf("expected env focus, got %v", m.focus)
	}
}

func TestSingleFileHelperEnterAdvancesFromEnvToFile(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev", "prod"}, []string{"a.yaml"}, "", false)
	for _, r := range "dev" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(singleFileHelperModel)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(singleFileHelperModel)

	if next.selectedEnv != "dev" {
		t.Fatalf("expected selected env dev, got %q", next.selectedEnv)
	}
	if next.focus != focusFile {
		t.Fatalf("expected file focus, got %v", next.focus)
	}
}

func TestSingleFileHelperTabTogglesFocus(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev"}, []string{"a.yaml"}, "", false)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	next := updated.(singleFileHelperModel)
	if next.focus != focusFile {
		t.Fatalf("expected file focus after tab, got %v", next.focus)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyTab})
	next = updated.(singleFileHelperModel)
	if next.focus != focusEnv {
		t.Fatalf("expected env focus after second tab, got %v", next.focus)
	}
}

func TestSingleFileHelperEscFromFileMovesBackToEnv(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev"}, []string{"a.yaml"}, "", false)
	m.setFocus(focusFile)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(singleFileHelperModel)

	if cmd != nil {
		t.Fatal("expected no quit command when moving focus back")
	}
	if next.focus != focusEnv {
		t.Fatalf("expected env focus, got %v", next.focus)
	}
}

func TestSingleFileHelperEscClearsFilterBeforeCancel(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev", "prod"}, []string{"a.yaml"}, "", false)
	for _, r := range "pro" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(singleFileHelperModel)
	}

	if m.envPane.textInput.Model.Value() == "" {
		t.Fatal("expected env filter to be set")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(singleFileHelperModel)

	if cmd != nil {
		t.Fatal("expected no quit command when clearing filter")
	}
	if next.envPane.textInput.Model.Value() != "" {
		t.Fatalf("expected filter cleared, got %q", next.envPane.textInput.Model.Value())
	}
	if next.cancelled {
		t.Fatal("did not expect cancellation")
	}
}

func TestSingleFileHelperEnterOnFileQuitsWithSelection(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev", "prod"}, []string{"a.yaml", "b.yaml"}, "", false)
	for _, r := range "dev" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(singleFileHelperModel)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(singleFileHelperModel)
	for _, r := range "a" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(singleFileHelperModel)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(singleFileHelperModel)

	if cmd == nil {
		t.Fatal("expected quit command on file selection")
	}
	if next.selectedFile != "a.yaml" {
		t.Fatalf("expected selected file a.yaml, got %q", next.selectedFile)
	}
}

func TestSingleFileHelperViewContainsSections(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev"}, []string{"a.yaml"}, "", false)
	view := m.View()

	if !strings.Contains(view, "Hulak") {
		t.Fatal("expected header title")
	}
	if !strings.Contains(view, "Environment") {
		t.Fatal("expected environment section title")
	}
	if !strings.Contains(view, "Request File") {
		t.Fatal("expected request file section title")
	}
	if !strings.Contains(view, "enter: select | tab: switch env/file | esc: clear/back/cancel | arrows: navigate") {
		t.Fatal("expected help text at bottom")
	}
}

func TestSingleFileHelperEnvLockedStartsOnFile(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev", "prod"}, []string{"a.yaml"}, "staging", true)

	if m.focus != focusFile {
		t.Fatalf("expected file focus for locked env, got %v", m.focus)
	}
}

func TestSingleFileHelperFileListStartsEmptyUntilTyping(t *testing.T) {
	m := newSingleFileHelperModel([]string{"dev", "prod"}, []string{"a.yaml", "b.yaml"}, "", false)
	if len(m.envPane.filtered) != 0 {
		t.Fatalf("expected empty env list before typing, got %d", len(m.envPane.filtered))
	}
	if len(m.filePane.filtered) != 0 {
		t.Fatalf("expected empty file list before typing, got %d", len(m.filePane.filtered))
	}

	view := m.View()
	if strings.Contains(view, "type to search") {
		t.Fatal("did not expect search hint text in view")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(singleFileHelperModel)
	if len(m.envPane.filtered) == 0 {
		t.Fatal("expected matching env entries after typing")
	}

	for _, r := range "ev" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(singleFileHelperModel)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(singleFileHelperModel)
	if m.focus != focusFile {
		t.Fatalf("expected file focus, got %v", m.focus)
	}

	view = m.View()
	if strings.Contains(view, "type to search") {
		t.Fatal("did not expect search hint text in view")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(singleFileHelperModel)
	if len(m.filePane.filtered) == 0 {
		t.Fatal("expected matching file entries after typing")
	}
}
