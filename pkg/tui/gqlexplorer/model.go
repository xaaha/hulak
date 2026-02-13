package gqlexplorer

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
)

// Model is the full-screen GraphQL explorer TUI.
type Model struct {
	operations []UnifiedOperation
	filtered   []UnifiedOperation
	cursor     int
	search     tui.TextInput
	width      int
	height     int
}

// NewModel creates an explorer model from a flat list of operations.
func NewModel(operations []UnifiedOperation) Model {
	sort.Slice(operations, func(i, j int) bool {
		return typeOrder(operations[i].Type) < typeOrder(operations[j].Type)
	})
	return Model{
		operations: operations,
		filtered:   operations,
		search:     tui.NewFilterInput("Search: ", "filter operations..."),
	}
}

func typeOrder(t OperationType) int {
	switch t {
	case TypeQuery:
		return 0
	case TypeMutation:
		return 1
	case TypeSubscription:
		return 2
	default:
		return 3
	}
}

func (m Model) Init() tea.Cmd {
	return m.search.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit
	case tui.KeyCancel:
		if m.search.Model.Value() != "" {
			m.search.Model.Reset()
			m.applyFilter()
			return m, nil
		}
		return m, tea.Quit
	case tui.KeyUp, tui.KeyCtrlP:
		m.cursor = tui.MoveCursorUp(m.cursor)
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN:
		m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
		return m, nil
	}

	prevValue := m.search.Model.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if m.search.Model.Value() != prevValue {
		m.applyFilter()
	}
	return m, cmd
}

func (m *Model) applyFilter() {
	query := m.search.Model.Value()
	if query == "" {
		m.filtered = m.operations
	} else {
		m.filtered = nil
		lower := strings.ToLower(query)
		for _, op := range m.operations {
			if strings.Contains(strings.ToLower(op.Name), lower) {
				m.filtered = append(m.filtered, op)
			}
		}
	}
	m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
}

func (m Model) View() string {
	search := m.search.ViewTitle()
	count := tui.HelpStyle.Render(
		fmt.Sprintf("  %d/%d operations", len(m.filtered), len(m.operations)),
	)

	list := m.renderList()
	help := tui.HelpStyle.Render("esc: quit | arrows: navigate | type to filter")

	content := fmt.Sprintf("%s \n %s \n\n %s \n\n %s", search, count, list, help)

	// TODOs: Reminder, we might have to clean this up
	// Border consumes 2 cols and 2 rows;
	// 	size the box to the terminal.
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorMuted).
		Width(m.width - 2).
		Height(m.height - 2)

	return border.Render(content)
}

func (m Model) renderList() string {
	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render("  (no matches)")
	}

	listHeight := m.height - 9
	if listHeight < 1 {
		listHeight = 10
	}

	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	end := min(start+listHeight, len(m.filtered))

	var lines []string
	var currentType OperationType
	for i := start; i < end; i++ {
		op := m.filtered[i]
		if op.Type != currentType {
			currentType = op.Type
			header := tui.RenderBadge(string(currentType), badgeColorIndex(currentType))
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, header)
		}
		if i == m.cursor {
			lines = append(lines, tui.SubtitleStyle.Render("  > "+op.Name))
		} else {
			lines = append(lines, "    "+op.Name)
		}
	}
	return strings.Join(lines, "\n")
}

func badgeColorIndex(t OperationType) int {
	switch t {
	case TypeQuery:
		return 0
	case TypeMutation:
		return 3
	case TypeSubscription:
		return 5
	default:
		return 0
	}
}

// RunExplorer launches the full-screen explorer TUI.
func RunExplorer(operations []UnifiedOperation) error {
	p := tea.NewProgram(NewModel(operations), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
