package envselect

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

// FormatNoEnvFilesError creates a user-friendly error message when no env files exist.
func FormatNoEnvFilesError() error {
	errMsg := `no '.env' files found in "env/" directory

Possible solutions:
  - Create an env file: echo "KEY=value" > env/dev.env
  - Run "hulak init" to create the env directory structure`

	return utils.ColorError(errMsg)
}

// Model is a lightweight environment selector.
type Model struct {
	items     []string
	filtered  []string
	cursor    int
	textInput tui.TextInput
	Selected  string
	Cancelled bool
}

// NewModel creates a new env selector model if `env/` dir exists and contains `.env` files
func NewModel() Model {
	var items []string
	if files, err := utils.GetEnvFiles(); err == nil {
		for _, file := range files {
			if name, ok := strings.CutSuffix(file, utils.DefaultEnvFileSuffix); ok {
				items = append(items, name)
			}
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
			Prompt:      "Select Environment: ",
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
			lines = append(lines, tui.SubtitleStyle.Render(utils.CursorMarker+" "+item))
		} else {
			lines = append(lines, "   "+item)
		}
	}
	return strings.Join(lines, "\n")
}

// RunEnvSelector runs the environment selector and returns the selected environment.
// Returns ErrNoEnvFiles if no .env files are found in the env directory.
func RunEnvSelector() (string, error) {
	model := NewModel()

	// Check if there are any env files before showing the selector
	if len(model.items) == 0 {
		return "", FormatNoEnvFilesError()
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
