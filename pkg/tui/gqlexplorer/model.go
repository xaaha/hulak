package gqlexplorer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

const (
	itemPadding   = 4
	detailPadding = 6

	noMatchesLabel  = "(no matches)"
	helpFilter      = "q: queries | m: mutations | s: subscriptions"
	helpNavigation  = "esc: quit | ↑/↓: navigate | scroll: mouse | type to filter"
	operationFormat = "%d/%d operations"
)

var badgeColor = map[OperationType]lipgloss.AdaptiveColor{
	TypeQuery:        {Light: "21", Dark: "39"},
	TypeMutation:     {Light: "130", Dark: "214"},
	TypeSubscription: {Light: "30", Dark: "87"},
}

var typeRank = map[OperationType]int{
	TypeQuery:        0,
	TypeMutation:     1,
	TypeSubscription: 2,
}

// Model is the full-screen GraphQL explorer TUI.
type Model struct {
	operations []UnifiedOperation
	filtered   []UnifiedOperation
	cursor     int
	search     tui.TextInput
	viewport   viewport.Model
	ready      bool
	width      int
	height     int
}

// NewModel creates an explorer model from a flat list of operations.
func NewModel(operations []UnifiedOperation) Model {
	sort.Slice(operations, func(i, j int) bool {
		return typeRank[operations[i].Type] < typeRank[operations[j].Type]
	})
	return Model{
		operations: operations,
		filtered:   operations,
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: "filter operations...",
			MinWidth:    32,
		}),
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
		listHeight := m.height - 16
		if listHeight < 1 {
			listHeight = 10
		}
		if !m.ready {
			m.viewport = viewport.New(m.width, listHeight)
			m.viewport.MouseWheelEnabled = true
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = listHeight
		}
		m.viewport.SetContent(m.renderList())
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit
	case tui.KeyCancel:
		if m.search.Model.Value() != "" {
			m.search.Model.Reset()
			m.applyFilter()
			m.viewport.SetContent(m.renderList())
			m.viewport.GotoTop()
			return m, nil
		}
		return m, tea.Quit
	case tui.KeyUp, tui.KeyCtrlP:
		m.cursor = tui.MoveCursorUp(m.cursor)
		m.viewport.SetContent(m.renderList())
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN:
		m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
		m.viewport.SetContent(m.renderList())
		return m, nil
	}

	prevValue := m.search.Model.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if m.search.Model.Value() != prevValue {
		m.applyFilter()
		m.viewport.SetContent(m.renderList())
		m.viewport.GotoTop()
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

	var list string
	if m.ready {
		list = m.viewport.View()
	} else {
		list = m.renderList()
	}
	help := tui.HelpStyle.Render(helpNavigation)

	scrollPct := tui.HelpStyle.Render(
		fmt.Sprintf(" %3.f%%", m.viewport.ScrollPercent()*100),
	)

	content := fmt.Sprintf(
		"%s\n%s\n%s\n\n%s\n\n%s  %s",
		search, filterHint, count, list, help, scrollPct,
	)

	return tui.BoxStyle.
		Width(m.width - 2 - 4).
		Height(m.height - 2 - 2).
		Render(content)
}

func (m Model) renderList() string {
	itemPrefix := strings.Repeat(" ", itemPadding)
	detailPrefix := strings.Repeat(" ", detailPadding)
	selectedPrefix := strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + utils.CursorMarker

	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render(
			strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + noMatchesLabel,
		)
	}

	var lines []string
	var currentType OperationType
	for i, op := range m.filtered {
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
	p := tea.NewProgram(
		NewModel(operations),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
