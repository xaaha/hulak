package envselect

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

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
	items := []string{"global"}
	if files, err := utils.GetEnvFiles(); err == nil {
		for _, f := range files {
			name := strings.TrimSuffix(f, ".env")
			if name != "global" {
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
		if m.cursor > 0 {
			m.cursor--
		}

	case tui.KeyDown, tui.KeyCtrlN:
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "backspace", "ctrl+h":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}

	case "ctrl+w":
		m.filter = tui.DeleteLastWord(m.filter)
		m.applyFilter()

	case "ctrl+u":
		m.filter = tui.ClearLine()
		m.applyFilter()

	default:
		if len(msg.Runes) > 0 {
			m.filter += string(msg.Runes)
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *Model) applyFilter() {
	if m.filter == "" {
		m.filtered = m.items
		return
	}

	m.filtered = nil
	lower := strings.ToLower(m.filter)
	for _, item := range m.items {
		if strings.Contains(strings.ToLower(item), lower) {
			m.filtered = append(m.filtered, item)
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
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
func RunEnvSelector() (string, error) {
	m, err := tea.NewProgram(NewModel()).Run()
	if err != nil {
		return "", err
	}

	model := m.(Model)
	if model.Cancelled {
		return "", nil
	}
	return model.Selected, nil
}
