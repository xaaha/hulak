package gqlexplorer

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

const (
	itemPadding   = 4
	detailPadding = 6

	// leave some space before and after the cursor
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

	dividerWidth = 3 // " │ " between left and right panels

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

	detailVP   viewport.Model
	inputTypes map[string]graphql.InputType
	enumTypes  map[string]graphql.EnumType

	endpoints        []string
	activeEndpoints  map[string]bool
	pickingEndpoints bool
	endpointCursor   int
	pendingEndpoints map[string]bool
	detailCacheKey   string
	detailCacheValue string
}

func NewModel(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
) Model {
	for i := range operations {
		if operations[i].NameLower == "" {
			operations[i].NameLower = strings.ToLower(operations[i].Name)
		}
		if operations[i].EndpointShort == "" {
			operations[i].EndpointShort = shortenEndpoint(operations[i].Endpoint)
		}
	}

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
		inputTypes:      inputTypes,
		enumTypes:       enumTypes,
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: "filter operations...",
		}),
	}
}

func (m Model) leftPanelWidth() int {
	return max((m.width-6)*tui.LeftPanelPct/100, 0)
}

func (m Model) rightPanelWidth() int {
	return max(m.width-6-m.leftPanelWidth()-dividerWidth, 0)
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
		rightW := m.rightPanelWidth()
		detailH := max(m.height-4, 1)
		m.search.Model.Width = max(panelW-searchBoxOverhead, 10)
		if !m.ready {
			m.viewport = viewport.New(panelW, listHeight)
			m.viewport.MouseWheelEnabled = true
			m.detailVP = viewport.New(rightW, detailH)
			m.ready = true
		} else {
			m.viewport.Width = panelW
			m.viewport.Height = listHeight
			m.detailVP.Width = rightW
			m.detailVP.Height = detailH
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
			m.applyFilterAndReset()
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
		m.applyFilterAndReset()
	}
	return m, cmd
}

func (m *Model) applyFilterAndReset() {
	m.applyFilter()
	m.viewport.GotoTop()
	m.syncViewport()
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

	if m.pickingEndpoints {
		return
	}

	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		op := m.filtered[m.cursor]
		detailKey := op.Endpoint + "\x1f" + op.Name + "\x1f" + strconv.Itoa(m.rightPanelWidth())
		if detailKey != m.detailCacheKey {
			m.detailCacheValue = renderDetail(op, m.rightPanelWidth(), m.inputTypes)
			m.detailCacheKey = detailKey
			m.detailVP.SetContent(m.detailCacheValue)
			m.detailVP.GotoTop()
		}
	} else {
		m.detailCacheKey = ""
		m.detailCacheValue = ""
		m.detailVP.SetContent("")
	}
}

func (m Model) View() string {
	leftCol := lipgloss.NewStyle().
		Width(m.leftPanelWidth()).
		Render(m.renderLeftContent())

	divider := renderDivider(max(m.height-4, 1))

	rightCol := lipgloss.NewStyle().
		Width(m.rightPanelWidth()).
		Render(m.detailVP.View())

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, divider, rightCol)

	box := tui.BoxStyle.
		Padding(0, 1).
		Width(m.width - 4).
		Height(m.height - 4).
		Render(combined)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func RunExplorer(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
) error {
	p := tea.NewProgram(
		NewModel(operations, inputTypes, enumTypes),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
