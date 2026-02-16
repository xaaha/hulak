package gqlexplorer

import (
	"fmt"
	"maps"
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

	scrollMargin = 3 // leave some space before and after the cursor

	noMatchesLabel      = "(no matches)"
	helpNavigation      = "esc: quit | ↑/↓: navigate | scroll: mouse | type to filter"
	helpEndpointPicker  = " k↑/j↓: navigate | space: toggle | enter: confirm | esc: cancel"
	operationFormat     = "%d/%d operations"
	checkMark           = "✓ "
	endpointPickerTitle = "Filter Endpoints:"
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
	filterHint string
	search     tui.TextInput
	viewport   viewport.Model
	ready      bool
	width      int
	height     int

	endpoints        []string
	activeEndpoints  map[string]bool
	pickingEndpoints bool
	endpointCursor   int
	pendingEndpoints map[string]bool
}

// NewModel creates an explorer model from a flat list of operations.
func NewModel(operations []UnifiedOperation) Model {
	sort.Slice(operations, func(i, j int) bool {
		return typeRank[operations[i].Type] < typeRank[operations[j].Type]
	})
	endpoints := collectEndpoints(operations)
	return Model{
		operations:      operations,
		filtered:        operations,
		filterHint:      buildFilterHint(operations, endpoints),
		endpoints:       endpoints,
		activeEndpoints: make(map[string]bool),
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: "filter operations...",
			MinWidth:    32,
		}),
	}
}

func collectEndpoints(operations []UnifiedOperation) []string {
	seen := make(map[string]bool)
	var endpoints []string
	for _, op := range operations {
		if op.Endpoint != "" && !seen[op.Endpoint] {
			seen[op.Endpoint] = true
			endpoints = append(endpoints, op.Endpoint)
		}
	}
	sort.Strings(endpoints)
	return endpoints
}

func (m Model) Init() tea.Cmd {
	return m.search.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// controls the viewport height for the operations
		listHeight := m.height - 30
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
		// sync viewport and cursor
		m.syncViewport()
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
	if m.pickingEndpoints {
		return m.handleEndpointPickerKey(msg)
	}

	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit
	case tui.KeyCancel:
		if m.search.Model.Value() != "" {
			m.search.Model.Reset()
			m.applyFilter()
			m.viewport.GotoTop()
			m.syncViewport()
			return m, nil
		}
		return m, tea.Quit
	case tui.KeyUp, tui.KeyCtrlP:
		m.cursor = tui.MoveCursorUp(m.cursor)
		m.syncViewport()
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN:
		m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
		m.syncViewport()
		return m, nil
	}

	prevValue := m.search.Model.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	newValue := m.search.Model.Value()
	if newValue != prevValue {
		if m.shouldEnterEndpointPicker(newValue) {
			m.enterEndpointPicker()
			return m, nil
		}
		m.applyFilter()
		m.viewport.GotoTop()
		m.syncViewport()
	}
	return m, cmd
}

func (m *Model) shouldEnterEndpointPicker(value string) bool {
	return len(m.endpoints) > 1 && len(value) >= 2 &&
		(value[len(value)-2] == 'e' || value[len(value)-2] == 'E') &&
		value[len(value)-1] == ':'
}

func (m *Model) enterEndpointPicker() {
	m.pickingEndpoints = true
	m.endpointCursor = 0
	m.pendingEndpoints = make(map[string]bool)
	maps.Copy(m.pendingEndpoints, m.activeEndpoints)
	m.syncViewport()
}

func (m Model) handleEndpointPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit
	case tui.KeyCancel:
		m.pickingEndpoints = false
		m.pendingEndpoints = nil
		m.stripEndpointPrefix()
		m.syncViewport()
		return m, nil
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyK:
		m.endpointCursor = tui.MoveCursorUp(m.endpointCursor)
		m.syncViewport()
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN, tui.KeyJ:
		m.endpointCursor = tui.MoveCursorDown(m.endpointCursor, len(m.endpoints)-1)
		m.syncViewport()
		return m, nil
	case " ":
		ep := m.endpoints[m.endpointCursor]
		m.pendingEndpoints[ep] = !m.pendingEndpoints[ep]
		if !m.pendingEndpoints[ep] {
			delete(m.pendingEndpoints, ep)
		}
		m.syncViewport()
		return m, nil
	case tui.KeyEnter:
		m.activeEndpoints = make(map[string]bool)
		for k, v := range m.pendingEndpoints {
			if v {
				m.activeEndpoints[k] = true
			}
		}
		m.pickingEndpoints = false
		m.pendingEndpoints = nil
		m.stripEndpointPrefix()
		m.applyFilter()
		m.viewport.GotoTop()
		m.syncViewport()
		return m, nil
	}
	return m, nil
}

func (m *Model) stripEndpointPrefix() {
	val := m.search.Model.Value()
	for {
		idx := strings.LastIndex(strings.ToLower(val), "e:")
		if idx < 0 {
			break
		}
		val = strings.TrimSpace(val[:idx])
	}
	m.search.Model.SetValue(val)
}

func (m *Model) syncViewport() {
	var content string
	var cursorLine int
	if m.pickingEndpoints {
		content, cursorLine = m.renderEndpointPicker()
	} else {
		content, cursorLine = m.renderList()
	}
	m.viewport.SetContent(content)
	h := m.viewport.Height
	if cursorLine < m.viewport.YOffset {
		m.viewport.SetYOffset(max(0, cursorLine-1))
	} else if cursorLine+scrollMargin >= m.viewport.YOffset+h {
		m.viewport.SetYOffset(cursorLine - h + 1 + scrollMargin)
	}
}

func buildFilterHint(operations []UnifiedOperation, endpoints []string) string {
	hasType := make(map[OperationType]bool)
	for _, op := range operations {
		hasType[op.Type] = true
	}
	var parts []string
	if len(hasType) >= 2 {
		if hasType[TypeQuery] {
			parts = append(parts, "q: queries")
		}
		if hasType[TypeMutation] {
			parts = append(parts, "m: mutations")
		}
		if hasType[TypeSubscription] {
			parts = append(parts, "s: subscriptions")
		}
	}
	if len(endpoints) > 1 {
		parts = append(parts, "e: endpoints")
	}
	if len(parts) == 0 {
		return ""
	}
	return tui.HelpStyle.Render(" " + strings.Join(parts, " | "))
}

func (m *Model) applyFilter() {
	query := m.search.Model.Value()
	hasEndpointFilter := len(m.activeEndpoints) > 0

	if query == "" && !hasEndpointFilter {
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
		case 'm', 'M':
			typeFilter = TypeMutation
		case 's', 'S':
			typeFilter = TypeSubscription
		}
		if typeFilter != "" {
			searchTerm = strings.ToLower(strings.TrimSpace(query[2:]))
		}
	}

	m.filtered = nil
	for _, op := range m.operations {
		if typeFilter != "" && op.Type != typeFilter {
			continue
		}
		if hasEndpointFilter && !m.activeEndpoints[op.Endpoint] {
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
	filterHint := m.filterHint

	var statusLine string
	if m.pickingEndpoints {
		statusLine = tui.HelpStyle.Render(" " + endpointPickerTitle)
	} else {
		badges := m.renderBadges()
		count := fmt.Sprintf(operationFormat, len(m.filtered), len(m.operations))
		if badges != "" {
			statusLine = tui.HelpStyle.Render(" "+count) + "  " + badges
		} else {
			statusLine = tui.HelpStyle.Render(" " + count)
		}
	}

	var list string
	if m.ready {
		list = m.viewport.View()
	} else {
		content, _ := m.renderList()
		list = content
	}

	var helpText string
	if m.pickingEndpoints {
		helpText = tui.HelpStyle.Render(helpEndpointPicker)
	} else {
		helpText = tui.HelpStyle.Render(helpNavigation)
	}

	scrollPct := tui.HelpStyle.Render(
		fmt.Sprintf(" %3.f%%", m.viewport.ScrollPercent()*100),
	)

	header := search
	if filterHint != "" {
		header += "\n" + filterHint
	}
	content := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s  %s",
		header, statusLine, list, helpText, scrollPct,
	)

	box := tui.BoxStyle.
		Width(m.width - 4).
		Height(m.height - 4).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderBadges() string {
	var badges []string
	for ep := range m.activeEndpoints {
		badges = append(badges, tui.RenderBadge(shortenEndpoint(ep), tui.ColorPrimary))
	}
	sort.Strings(badges)
	return strings.Join(badges, " ")
}

func shortenEndpoint(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, "/graphql")
	url = strings.TrimSuffix(url, "/gql")
	url = strings.TrimSuffix(url, "/")
	return url
}

func (m Model) renderEndpointPicker() (string, int) {
	itemPrefix := strings.Repeat(" ", itemPadding)
	selectedPrefix := strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + utils.CursorMarker

	if len(m.endpoints) == 0 {
		return tui.HelpStyle.Render(itemPrefix + noMatchesLabel), 0
	}

	var lines []string
	cursorLine := 0
	for i, ep := range m.endpoints {
		prefix := itemPrefix
		if i == m.endpointCursor {
			prefix = selectedPrefix
			cursorLine = len(lines)
		}
		toggle := "  "
		if m.pendingEndpoints[ep] {
			toggle = checkMark
		}
		style := lipgloss.NewStyle()
		if i == m.endpointCursor {
			style = tui.SubtitleStyle
		}
		lines = append(lines, style.Render(prefix+toggle+ep))
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m Model) renderList() (string, int) {
	itemPrefix := strings.Repeat(" ", itemPadding)
	detailPrefix := strings.Repeat(" ", detailPadding)
	selectedPrefix := strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + utils.CursorMarker

	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render(
			strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + noMatchesLabel,
		), 0
	}

	var lines []string
	cursorLine := 0
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
			cursorLine = len(lines)
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
	return strings.Join(lines, "\n"), cursorLine
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
