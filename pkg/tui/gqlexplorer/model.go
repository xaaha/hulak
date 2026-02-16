package gqlexplorer

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
)

const (
	itemPadding   = 4
	detailPadding = 6

	scrollMargin = 3

	// ViewTitle border+padding (4) + len("Search: ") (8)
	searchBoxOverhead = 12

	// Lines the search ViewTitle occupies: top border + input + bottom border
	searchBoxLines = 3

	// Fixed lines around the viewport in View():
	//   above: "\n" + statusLine + "\n\n"  = 2 content + 1 blank
	//   below: "\n\n" + helpText+scrollPct = 1 blank  + 1 content
	//   box:   border top/bottom (2) + outer margin (4)
	viewportFrameLines = 10

	noMatchesLabel  = "(no matches)"
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
	active := make(map[string]bool, len(endpoints))
	for _, ep := range endpoints {
		active[ep] = true
	}
	return Model{
		operations:      operations,
		filtered:        operations,
		filterHint:      buildFilterHint(operations, endpoints),
		endpoints:       endpoints,
		activeEndpoints: active,
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: "filter operations...",
		}),
	}
}

func (m Model) leftPanelWidth() int {
	return tui.LeftPanelWidth(m.width)
}

func (m Model) viewportHeight() int {
	headerLines := searchBoxLines
	if len(m.activeEndpoints) > 0 {
		headerLines++
	}
	if m.filterHint != "" {
		headerLines++
	}
	h := max(m.height-viewportFrameLines-headerLines, 1)
	return h
}

func (m Model) Init() tea.Cmd {
	return m.search.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		panelW := m.leftPanelWidth()
		listHeight := m.viewportHeight()
		m.search.Model.Width = max(panelW-searchBoxOverhead, 10)
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

func (m Model) View() string {
	badges := m.renderBadges()
	search := m.search.ViewTitle()
	filterHint := m.filterHint

	var statusLine string
	if m.pickingEndpoints {
		statusLine = tui.HelpStyle.Render(tui.KeySpace + endpointPickerTitle)
	} else {
		statusLine = tui.HelpStyle.Render(
			tui.KeySpace + fmt.Sprintf(operationFormat, len(m.filtered), len(m.operations)),
		)
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

	var header string
	if badges != "" {
		header += badges + "\n"
	}
	header += search
	if filterHint != "" {
		header += "\n" + filterHint
	}
	content := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s  %s",
		header, statusLine, list, helpText, scrollPct,
	)

	box := tui.BoxStyle.
		Padding(0, 1).
		Width(m.width - 4).
		Height(m.height - 4).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
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
