package gqlexplorer

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
)

const (
	itemPadding   = 4
	detailPadding = 6
	cursorMarker  = "> "

	noMatchesLabel  = "(no matches)"
	helpFilter      = "q: queries | m: mutations | s: subscriptions"
	helpNavigation  = "esc: quit | ↑/↓: navigate | type to filter"
	operationFormat = "%d/%d operations"
)

var badgeColor = map[OperationType]lipgloss.AdaptiveColor{
	TypeQuery:        {Light: "21", Dark: "39"},
	TypeMutation:     {Light: "130", Dark: "214"},
	TypeSubscription: {Light: "30", Dark: "87"},
}

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
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: "filter operations...",
		}),
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
		m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
		return
	}

	var typeFilter OperationType
	searchTerm := strings.ToLower(query)

	if len(query) >= 2 && query[1] == ':' {
		switch query[0] {
		case 'q', 'Q':
			typeFilter = TypeQuery
			searchTerm = strings.ToLower(strings.TrimSpace(query[2:]))
		case 'm', 'M':
			typeFilter = TypeMutation
			searchTerm = strings.ToLower(strings.TrimSpace(query[2:]))
		case 's', 'S':
			typeFilter = TypeSubscription
			searchTerm = strings.ToLower(strings.TrimSpace(query[2:]))
		}
	}

	m.filtered = nil
	for _, op := range m.operations {
		if typeFilter != "" && op.Type != typeFilter {
			continue
		}
		if searchTerm == "" || strings.Contains(strings.ToLower(op.Name), searchTerm) {
			m.filtered = append(m.filtered, op)
		}
	}
	m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
}

func (m Model) View() string {
	search := m.search.ViewTitle()
	filterHint := tui.HelpStyle.Render(" " + helpFilter)
	count := tui.HelpStyle.Render(
		" " + fmt.Sprintf(operationFormat, len(m.filtered), len(m.operations)),
	)

	list := m.renderList()
	help := tui.HelpStyle.Render(helpNavigation)

	content := fmt.Sprintf("%s\n%s\n%s\n\n%s\n\n%s", search, filterHint, count, list, help)

	return tui.BoxStyle.
		Width(m.width - 2 - 4).
		Height(m.height - 2 - 2).
		Render(content)
}

func (m Model) renderList() string {
	itemPrefix := strings.Repeat(" ", itemPadding)
	detailPrefix := strings.Repeat(" ", detailPadding)
	selectedPrefix := strings.Repeat(" ", itemPadding-len(cursorMarker)) + cursorMarker

	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render(
			strings.Repeat(" ", itemPadding-len(cursorMarker)) + noMatchesLabel,
		)
	}

	listHeight := m.height - 16
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
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, tui.RenderBadge(string(currentType), badgeColor[currentType]))
		}
		if i == m.cursor {
			lines = append(lines, tui.SubtitleStyle.Render(selectedPrefix+op.Name))
			if op.Description != "" {
				lines = append(lines, tui.HelpStyle.Render(detailPrefix+op.Description))
			}
			if op.Endpoint != "" {
				lines = append(lines, tui.HelpStyle.Render(detailPrefix+op.Endpoint))
			}
		} else {
			lines = append(lines, itemPrefix+op.Name)
		}
	}
	return strings.Join(lines, "\n")
}

// RunExplorer launches the full-screen explorer TUI.
func RunExplorer(operations []UnifiedOperation) error {
	p := tea.NewProgram(NewModel(operations), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
