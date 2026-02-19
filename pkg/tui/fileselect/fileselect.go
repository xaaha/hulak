package fileselect

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

// FormatNoFilesError creates a user-friendly error message when no yaml/yml files exist.
func FormatNoFilesError() error {
	errMsg := `no '.yaml' or '.yml' files found in current directory

Possible solutions:
  - Create an API file: echo "method: GET" > api.yaml
  - Check that files are not inside the 'env/' directory`

	return utils.ColorError(errMsg)
}

// Model is a lightweight file selector.
type Model struct {
	items     []string
	filtered  []string
	cursor    int
	textInput tui.TextInput
	Selected  string
	Cancelled bool
}

// NewModel creates a new file selector model with yaml/yml files from current directory
func NewModel() Model {
	var items []string

	// Get all files from current directory
	if files, err := utils.ListFiles("."); err == nil {
		cwd, _ := os.Getwd()
		for _, file := range files {
			// Convert absolute path to relative
			relPath, err := filepath.Rel(cwd, file)
			if err != nil {
				continue
			}

			// Filter: only .yaml and .yml files
			lower := strings.ToLower(relPath)
			if !strings.HasSuffix(lower, utils.YAML) && !strings.HasSuffix(lower, utils.YML) {
				continue
			}

			// Filter: exclude _response files
			if strings.Contains(relPath, utils.ResponseBase) {
				continue
			}

			// Filter: exclude files in env/ directory
			if strings.HasPrefix(relPath, utils.EnvironmentFolder+string(filepath.Separator)) {
				continue
			}

			items = append(items, relPath)
		}
	}

	var placeholder string
	if len(items) > 0 {
		placeholder = items[0]
	}

	return Model{
		items:    items,
		filtered: items,
		textInput: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Select File: ",
			Placeholder: placeholder,
			MinWidth:    20,
		}),
	}
}

func (m Model) Init() tea.Cmd {
	return m.textInput.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		m.Cancelled = true
		return m, tea.Quit

	case tui.KeyCancel:
		if m.textInput.Model.Value() != "" {
			m.textInput.Model.Reset()
			m.applyFilter()
			return m, nil
		}
		m.Cancelled = true
		return m, tea.Quit

	case tui.KeyEnter:
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			m.Selected = m.filtered[m.cursor]
		}
		return m, tea.Quit

	case tui.KeyUp, tui.KeyCtrlP:
		m.cursor = tui.MoveCursorUp(m.cursor)
		return m, nil

	case tui.KeyDown, tui.KeyCtrlN:
		m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
		return m, nil
	}

	// Delegate all other keys to textinput
	prevValue := m.textInput.Model.Value()
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if m.textInput.Model.Value() != prevValue {
		m.applyFilter()
	}

	return m, cmd
}

// applyFilter matches the user's input text against list items
func (m *Model) applyFilter() {
	userInput := m.textInput.Model.Value()
	if userInput == "" {
		m.filtered = m.items
	} else {
		m.filtered = nil
		lower := strings.ToLower(userInput)
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item), lower) {
				m.filtered = append(m.filtered, item)
			}
		}
	}
	m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
}

func (m Model) View() string {
	title := m.textInput.ViewTitle()
	list := m.renderList()
	help := tui.HelpStyle.Render("enter: select | esc: cancel | arrows: navigate")

	content := title + "\n\n" + list + "\n" + help
	return "\n" + tui.BoxStyle.Render(content) + "\n"
}

func (m Model) renderList() string {
	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render("   (no matches)")
	}

	var lines []string
	for i, item := range m.filtered {
		if i == m.cursor {
			lines = append(lines, tui.SubtitleStyle.Render(utils.ChevronRight+tui.KeySpace+item))
		} else {
			lines = append(lines, strings.Repeat(tui.KeySpace, 3)+item)
		}
	}
	return strings.Join(lines, "\n")
}

// RunFileSelector runs the file selector and returns the selected file path.
// Returns FormatNoFilesError if no .yaml/.yml files are found.
func RunFileSelector() (string, error) {
	model := NewModel()

	// Check if there are any files before showing the selector
	if len(model.items) == 0 {
		return "", FormatNoFilesError()
	}

	m, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}

	result := m.(Model)
	if result.Cancelled {
		return "", nil
	}
	return result.Selected, nil
}
