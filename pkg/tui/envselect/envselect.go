package envselect

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

var boxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(tui.ColorMuted).
	Padding(1, 2)

// Model is a lightweight environment selector
type Model struct {
	items     []string // all items
	filtered  []string // filtered items (based on filter text)
	cursor    int      // cursor position in filtered list
	filter    string   // current filter text
	Selected  string
	Cancelled bool
}

// NewModel creates a new env selector model
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
	return Model{
		items:    items,
		filtered: items,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
			// Clear filter first
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
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN:
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil
	case "backspace", "ctrl+h":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
		return m, nil
	case "ctrl+w":
		// Delete last word
		m.filter = deleteLastWord(m.filter)
		m.applyFilter()
		return m, nil
	case "ctrl+u":
		// Clear entire filter
		m.filter = ""
		m.applyFilter()
		return m, nil
	default:
		// Add printable characters to filter
		if len(msg.Runes) > 0 {
			m.filter += string(msg.Runes)
			m.applyFilter()
		}
		return m, nil
	}
}

// deleteLastWord removes the last word from the string
func deleteLastWord(s string) string {
	if s == "" {
		return ""
	}
	// Trim trailing spaces first
	s = strings.TrimRight(s, " ")
	// Find last space
	lastSpace := strings.LastIndex(s, " ")
	if lastSpace == -1 {
		return ""
	}
	return s[:lastSpace+1]
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
	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) View() string {
	var b strings.Builder

	// Title with integrated filter input
	title := "Select Environment: "
	if m.filter != "" {
		title += m.filter
	}
	title += "█"
	b.WriteString(tui.TitleStyle.Render(title))
	b.WriteString("\n\n")

	// List items
	for i, item := range m.filtered {
		if i == m.cursor {
			b.WriteString(tui.SubtitleStyle.Render(">  " + item))
		} else {
			b.WriteString("   " + item)
		}
		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(tui.HelpStyle.Render("   (no matches)"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(tui.HelpStyle.Render("enter: select • esc: cancel • ↑/↓: navigate"))

	// Wrap in box
	return "\n" + boxStyle.Render(b.String()) + "\n"
}

// RunEnvSelector runs the environment selector and returns the selected environment
func RunEnvSelector() (string, error) {
	p := tea.NewProgram(NewModel())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	m := final.(Model)
	if m.Cancelled {
		return "", nil
	}
	return m.Selected, nil
}
