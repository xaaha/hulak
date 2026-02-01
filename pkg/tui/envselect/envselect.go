package envselect

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

// ErrNoEnvFiles is returned when no .env files are found in the env directory.
var ErrNoEnvFiles = errors.New("no environment files found")

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
	filter    string
	Selected  string
	Cancelled bool
}

// NewModel creates a new env selector model.
func NewModel() Model {
	items := []string{}
	if files, err := utils.GetEnvFiles(); err == nil {
		for _, file := range files {
			if name, ok := strings.CutSuffix(file, ".env"); ok {
				items = append(items, name)
			}
		}
	}
	return Model{items: items, filtered: items}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		m.Cancelled = true
		return m, tea.Quit

	case tui.KeyCancel:
		if m.filter != "" {
			m.filter = ""
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

	case tui.KeyDown, tui.KeyCtrlN:
		m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)

	case tui.KeyBackspace, tui.KeyCtrlH:
		m.filter = tui.DeleteChar(m.filter)
		m.applyFilter()

	case tui.KeyCtrlW:
		m.filter = tui.DeleteLastWord(m.filter)
		m.applyFilter()

	case tui.KeyCtrlU:
		m.filter = tui.ClearLine()
		m.applyFilter()

	default:
		m.filter = tui.AppendRunes(m.filter, msg.Runes)
		m.applyFilter()
	}
	return m, nil
}

func (m *Model) applyFilter() {
	if m.filter == "" {
		m.filtered = m.items
	} else {
		m.filtered = nil
		lower := strings.ToLower(m.filter)
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item), lower) {
				m.filtered = append(m.filtered, item)
			}
		}
	}
	m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
}

func (m Model) View() string {
	title := m.renderTitle()
	list := m.renderList()
	help := tui.HelpStyle.Render("enter: select • esc: cancel • ↑/↓: navigate")

	content := title + "\n\n" + list + "\n" + help
	return "\n" + tui.BoxStyle.Render(content) + "\n"
}

func (m Model) renderTitle() string {
	title := "Select Environment: " + m.filter + "█"
	return tui.TitleStyle.Render(title)
}

func (m Model) renderList() string {
	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render("   (no matches)")
	}

	var lines []string
	for i, item := range m.filtered {
		if i == m.cursor {
			lines = append(lines, tui.SubtitleStyle.Render(">  "+item))
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
