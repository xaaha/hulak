package apicaller

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilePathStartsWithEnvFocus(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod"}, []string{"a.yaml"}, "", false)

	if m.focus != focusEnv {
		t.Fatalf("expected env focus, got %v", m.focus)
	}
}

func TestFilePathInputPromptsAreBlank(t *testing.T) {
	m := newFilePathModel([]string{"global", "dev"}, []string{"a.yaml"}, "", false)

	if m.envPane.TextInput.Model.Prompt != "" {
		t.Fatalf("expected empty env prompt, got %q", m.envPane.TextInput.Model.Prompt)
	}
	if m.filePane.TextInput.Model.Prompt != "" {
		t.Fatalf("expected empty file prompt, got %q", m.filePane.TextInput.Model.Prompt)
	}
}

func TestFilePathEnvPlaceholderPrefersGlobalThenFirst(t *testing.T) {
	m := newFilePathModel([]string{"dev", "global", "prod"}, []string{"a.yaml"}, "", false)
	if m.envPane.TextInput.Model.Placeholder != "global" {
		t.Fatalf("expected env placeholder global, got %q", m.envPane.TextInput.Model.Placeholder)
	}

	m = newFilePathModel([]string{"staging", "prod"}, []string{"a.yaml"}, "", false)
	if m.envPane.TextInput.Model.Placeholder != "staging" {
		t.Fatalf("expected env placeholder first item, got %q", m.envPane.TextInput.Model.Placeholder)
	}
}

func TestFilePathEnterAdvancesFromEnvToFile(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod"}, []string{"a.yaml"}, "", false)
	for _, r := range "dev" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(filePathModel)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(filePathModel)

	if next.selectedEnv != "dev" {
		t.Fatalf("expected selected env dev, got %q", next.selectedEnv)
	}
	if next.focus != focusFile {
		t.Fatalf("expected file focus, got %v", next.focus)
	}
}

func TestFilePathTabTogglesFocus(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "", false)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	next := updated.(filePathModel)
	if next.focus != focusFile {
		t.Fatalf("expected file focus after tab, got %v", next.focus)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyTab})
	next = updated.(filePathModel)
	if next.focus != focusEnv {
		t.Fatalf("expected env focus after second tab, got %v", next.focus)
	}
}

func TestFilePathEscFromFileMovesBackToEnv(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "", false)
	m.setFocus(focusFile)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(filePathModel)

	if cmd != nil {
		t.Fatal("expected no quit command when moving focus back")
	}
	if next.focus != focusEnv {
		t.Fatalf("expected env focus, got %v", next.focus)
	}
}

func TestFilePathEscClearsFilterBeforeCancel(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod"}, []string{"a.yaml"}, "", false)
	for _, r := range "pro" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(filePathModel)
	}

	if m.envPane.TextInput.Model.Value() == "" {
		t.Fatal("expected env filter to be set")
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(filePathModel)

	if cmd != nil {
		t.Fatal("expected no quit command when clearing filter")
	}
	if next.envPane.TextInput.Model.Value() != "" {
		t.Fatalf("expected filter cleared, got %q", next.envPane.TextInput.Model.Value())
	}
	if next.cancelled {
		t.Fatal("did not expect cancellation")
	}
}

func TestFilePathEnterOnFileQuitsWithSelection(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod"}, []string{"a.yaml", "b.yaml"}, "", false)
	for _, r := range "dev" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(filePathModel)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(filePathModel)
	for _, r := range "a" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(filePathModel)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(filePathModel)

	if cmd == nil {
		t.Fatal("expected quit command on file selection")
	}
	if next.selectedFile != "a.yaml" {
		t.Fatalf("expected selected file a.yaml, got %q", next.selectedFile)
	}
}

func TestFilePathViewContainsSections(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "", false)
	view := m.View()

	if !strings.Contains(view, "Environment") {
		t.Fatal("expected environment section title")
	}
	if !strings.Contains(view, "Request File") {
		t.Fatal("expected request file section title")
	}
	if !strings.Contains(
		view,
		"enter: select | tab: switch env/file | esc: clear/back/cancel | arrows: navigate",
	) {
		t.Fatal("expected help text at bottom")
	}
	if strings.Contains(view, "\n\n\n\n") {
		t.Fatal("expected compact stacked spacing without excessive blank gaps")
	}
}

func TestFilePathEnvLockedStartsOnFile(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod"}, []string{"a.yaml"}, "staging", true)

	if m.focus != focusFile {
		t.Fatalf("expected file focus for locked env, got %v", m.focus)
	}
}

func TestFilePathFileListStartsPopulatedBeforeTyping(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod"}, []string{"a.yaml", "b.yaml"}, "", false)
	if len(m.envPane.Filtered) != 2 {
		t.Fatalf("expected populated env list before typing, got %d", len(m.envPane.Filtered))
	}
	if len(m.filePane.Filtered) != 2 {
		t.Fatalf("expected populated file list before typing, got %d", len(m.filePane.Filtered))
	}

	view := m.View()
	if strings.Contains(view, "type to search") {
		t.Fatal("did not expect search hint text in view")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(filePathModel)
	if len(m.envPane.Filtered) == 0 {
		t.Fatal("expected matching env entries after typing")
	}

	for _, r := range "ev" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(filePathModel)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(filePathModel)
	if m.focus != focusFile {
		t.Fatalf("expected file focus, got %v", m.focus)
	}

	view = m.View()
	if strings.Contains(view, "type to search") {
		t.Fatal("did not expect search hint text in view")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(filePathModel)
	if len(m.filePane.Filtered) == 0 {
		t.Fatal("expected matching file entries after typing")
	}
}

func TestFilePathCtrlCQuits(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "", false)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	next := updated.(filePathModel)

	if !next.cancelled {
		t.Fatal("expected cancelled after ctrl+c")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestFilePathArrowNavigation(t *testing.T) {
	m := newFilePathModel([]string{"dev", "prod", "staging"}, []string{"a.yaml"}, "", false)
	for _, r := range "d" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(filePathModel)
	}

	if m.envPane.Cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.envPane.Cursor)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(filePathModel)
	if m.envPane.Cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", m.envPane.Cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(filePathModel)
	if m.envPane.Cursor != 0 {
		t.Fatalf("expected cursor 0 after up, got %d", m.envPane.Cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(filePathModel)
	if m.envPane.Cursor != 0 {
		t.Fatal("cursor should not go below 0")
	}
}

func TestFilePathEnterOnFileWithoutEnvRedirects(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "", false)
	m.setFocus(focusFile)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(filePathModel)

	if cmd != nil {
		t.Fatal("expected no quit command when env not selected")
	}
	if next.focus != focusEnv {
		t.Fatalf("expected redirect to env focus, got %v", next.focus)
	}
}

func TestFilePathEnvLockedEscQuits(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "prod", true)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next := updated.(filePathModel)

	if !next.cancelled {
		t.Fatal("expected cancellation on esc with locked env")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestFilePathEnvLockedTabStaysOnFile(t *testing.T) {
	m := newFilePathModel([]string{"dev"}, []string{"a.yaml"}, "prod", true)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	next := updated.(filePathModel)

	if next.focus != focusFile {
		t.Fatalf("expected tab to stay on file when env locked, got %v", next.focus)
	}
}

func TestFilePathViewportHeightStaysStableWhileFiltering(t *testing.T) {
	m := newFilePathModel(
		[]string{"dev", "prod", "staging"},
		[]string{"a.yaml", "b.yaml", "c.yaml"},
		"",
		false,
	)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = updated.(filePathModel)

	initialEnvHeight := m.envPane.vp.Height
	initialFileHeight := m.filePane.vp.Height

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(filePathModel)

	if m.envPane.vp.Height != initialEnvHeight {
		t.Fatalf("expected env pane height %d, got %d", initialEnvHeight, m.envPane.vp.Height)
	}
	if m.filePane.vp.Height != initialFileHeight {
		t.Fatalf("expected file pane height %d, got %d", initialFileHeight, m.filePane.vp.Height)
	}
}

func TestFilePathViewportHeightsAreCappedForCompactLayout(t *testing.T) {
	m := newFilePathModel(
		[]string{"dev", "prod", "staging"},
		[]string{"a.yaml", "b.yaml", "c.yaml", "d.yaml", "e.yaml", "f.yaml", "g.yaml"},
		"",
		false,
	)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	m = updated.(filePathModel)

	if m.envPane.vp.Height > maxEnvListH {
		t.Fatalf("expected env viewport height <= %d, got %d", maxEnvListH, m.envPane.vp.Height)
	}
	if m.filePane.vp.Height > maxFileListH {
		t.Fatalf("expected file viewport height <= %d, got %d", maxFileListH, m.filePane.vp.Height)
	}
}
