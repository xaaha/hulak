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

	// Lines the search ViewTitle occupies: top border + input + bottom border
	searchBoxLines        = 3
	noMatchesLabel        = "(no matches)"
	helpNavigation        = "esc: quit | ↑/↓: navigate"
	operationFormat       = "%d/%d operations"
	searchPlaceholderText = "filter operations..."
	// below this width, the ui does not have enough space to render fixed text
	// like searchPlaceholderText and badge.
	minHeaderContentWidth = 111
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
	enumTypes  map[string]graphql.EnumType // TODO: wire into detail panel for enum expansion

	endpoints        []string
	activeEndpoints  map[string]bool
	pickingEndpoints bool
	endpointCursor   int
	pendingEndpoints map[string]bool
	detailCacheKey   string
	detailCacheValue string
	badgeCache       string

	focusedPanel panelFocus
}

type panelFocus uint8

const (
	focusLeft panelFocus = iota
	focusRight
)

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
	m := Model{
		operations:      operations,
		filtered:        operations,
		filterHint:      buildFilterHint(operations, endpoints),
		endpoints:       endpoints,
		activeEndpoints: active,
		inputTypes:      inputTypes,
		enumTypes:       enumTypes,
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: searchPlaceholderText,
		}),
	}
	m.setFocus(focusLeft)
	return m
}

// hasHeaderContentSpace guards optional header UI that is visually noisy in
// narrow terminals (badge row + placeholder hint).
// When space is limited, returning false keeps the search row stable by hiding those extras.
func (m Model) hasHeaderContentSpace() bool {
	return m.width >= minHeaderContentWidth
}

func (m *Model) updateSearchPlaceholder() {
	if m.hasHeaderContentSpace() {
		m.search.Model.Placeholder = searchPlaceholderText
		return
	}
	m.search.Model.Placeholder = ""
}

func (m Model) leftPanelWidth() int {
	contentW := m.contentWidth()
	if !m.hasTwoPanelLayout() {
		return max(contentW, 1)
	}

	leftW := contentW * tui.LeftPanelPct / 100
	leftW = max(leftW, tui.MinLeftPanelWidth)
	maxLeft := max(contentW-tui.MinRightPanelWidth, 1)
	leftW = min(leftW, maxLeft)
	return max(leftW, 1)
}

func (m Model) rightPanelWidth() int {
	if !m.hasTwoPanelLayout() {
		return 0
	}
	return max(m.contentWidth()-m.leftPanelWidth(), 0)
}

func (m Model) hasTwoPanelLayout() bool {
	return m.contentWidth() >= tui.MinLeftPanelWidth+tui.MinRightPanelWidth
}

func containerStyle() lipgloss.Style {
	return tui.BoxStyle.Padding(0, 1)
}

func (m Model) contentWidth() int {
	return max(m.width-containerStyle().GetHorizontalFrameSize(), 1)
}

func (m Model) contentHeight() int {
	return max(m.height-containerStyle().GetVerticalFrameSize(), 1)
}

func detailFocusStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 0)
}

func (m Model) detailOuterWidth() int {
	return max(m.rightPanelWidth()*tui.DetailFocusBoxW/100, 1)
}

func (m Model) detailOuterHeight() int {
	return max(m.detailTopHeight()*tui.DetailFocusBoxH/100, 1)
}

func (m Model) detailViewportSize() (int, int) {
	style := detailFocusStyle()
	w := max(m.detailOuterWidth()-style.GetHorizontalFrameSize(), 0)
	h := max(m.detailOuterHeight()-style.GetVerticalFrameSize(), 0)
	return w, h
}

func (m Model) canRenderDetailBox() bool {
	style := detailFocusStyle()
	return m.detailOuterWidth() > style.GetHorizontalFrameSize() &&
		m.detailOuterHeight() > style.GetVerticalFrameSize()
}

func (m Model) detailHeight() int {
	return max(m.contentHeight(), 1)
}

// detailTopHeight returns the height allocated to the detail viewport
// (top half of the right panel).
func (m Model) detailTopHeight() int {
	return max(m.detailHeight()*tui.DetailTopHeight/100, 1)
}

// responseAreaHeight returns the height allocated to the response area
// (bottom half of the right panel, below the horizontal divider).
func (m Model) responseAreaHeight() int {
	top := m.detailTopHeight()
	return max(m.detailHeight()-top, 1)
}

func (m *Model) updateBadgeCache() {
	if !m.hasHeaderContentSpace() {
		m.badgeCache = ""
		return
	}
	m.badgeCache = m.renderBadges()
}

func (m *Model) toggleFocus() {
	if m.focusedPanel == focusLeft {
		m.setFocus(focusRight)
		return
	}
	m.setFocus(focusLeft)
}

func (m *Model) setFocus(f panelFocus) {
	m.focusedPanel = f
	if f == focusLeft {
		m.search.Model.Focus()
		return
	}
	m.search.Model.Blur()
}

func (m Model) activeScrollPanel() panelFocus {
	if m.pickingEndpoints {
		return focusLeft
	}
	return m.focusedPanel
}

func (m *Model) updateFocusedViewport(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.activeScrollPanel() == focusLeft {
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}
	m.detailVP, cmd = m.detailVP.Update(msg)
	return cmd
}

func (m Model) viewportHeight() int {
	panelW := max(m.leftPanelWidth(), 1)
	headerLines := searchBoxLines
	if len(m.activeEndpoints) > 0 {
		headerLines++
	}
	if m.filterHint != "" {
		headerLines += wrappedLineCount(m.filterHint, panelW)
	}
	// statusLine (always 1) + help line (may wrap)
	footerLines := 1
	helpText := helpNavigation
	if m.pickingEndpoints {
		helpText = helpEndpointPicker
	}
	footerLines += wrappedLineCount(helpText, panelW)
	h := max(m.contentHeight()-headerLines-footerLines, 1)
	return h
}

func wrappedLineCount(text string, width int) int {
	if width <= 0 || text == "" {
		return 0
	}
	return lipgloss.Height(lipgloss.NewStyle().Width(width).Render(text))
}

func (m Model) Init() tea.Cmd {
	return m.search.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSearchPlaceholder()
		panelW := m.leftPanelWidth()
		listHeight := m.viewportHeight()
		detailW := max(m.rightPanelWidth(), 1)
		detailH := max(m.detailTopHeight(), 1)
		if m.canRenderDetailBox() {
			detailW, detailH = m.detailViewportSize()
			detailW = max(detailW, 1)
			detailH = max(detailH, 1)
		}
		searchFrame := tui.InputStyle.GetHorizontalFrameSize()
		m.search.Model.Width = max(panelW-searchFrame-len(m.search.Model.Prompt), 1)
		if !m.ready {
			m.viewport = viewport.New(panelW, listHeight)
			m.viewport.MouseWheelEnabled = true
			m.detailVP = viewport.New(detailW, detailH)
			m.detailVP.MouseWheelEnabled = true
			m.ready = true
		} else {
			m.viewport.Width = panelW
			m.viewport.Height = listHeight
			m.detailVP.Width = detailW
			m.detailVP.Height = detailH
		}
		m.updateBadgeCache()
		m.syncViewport()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	cmds = append(cmds, cmd)
	cmd = m.updateFocusedViewport(msg)
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
		if m.focusedPanel == focusRight {
			var cmd tea.Cmd
			m.detailVP, cmd = m.detailVP.Update(msg)
			return m, cmd
		}
		m.cursor = tui.MoveCursorUp(m.cursor)
		m.syncViewport()
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN:
		if m.focusedPanel == focusRight {
			var cmd tea.Cmd
			m.detailVP, cmd = m.detailVP.Update(msg)
			return m, cmd
		}
		m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
		m.syncViewport()
		return m, nil
	case tui.KeyLeft, tui.KeyRight:
		if m.focusedPanel == focusRight {
			var cmd tea.Cmd
			m.detailVP, cmd = m.detailVP.Update(msg)
			return m, cmd
		}
		return m, nil
	case tui.KeyTab:
		m.toggleFocus()
		return m, nil
	case tui.KeyEnter:
		if m.focusedPanel == focusLeft && m.hasTwoPanelLayout() {
			m.setFocus(focusRight)
		}
		return m, nil
	}

	if m.focusedPanel == focusRight {
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
	tui.SyncViewport(&m.viewport, content, cursorLine, tui.DefaultScrollMargin)

	if m.pickingEndpoints {
		return
	}

	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		op := m.filtered[m.cursor]
		// inputTypes is immutable for the program lifetime, so it's safe
		// to omit from the cache key.
		detailKey := op.Endpoint + "\x1f" + op.Name + "\x1f" + strconv.Itoa(m.rightPanelWidth())
		if detailKey != m.detailCacheKey {
			m.detailCacheValue = renderDetail(op, m.inputTypes)
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
		Height(m.detailHeight()).
		Render(m.renderLeftContent())
	if !m.hasTwoPanelLayout() {
		boxStyle := containerStyle()
		box := boxStyle.
			Width(max(m.width-boxStyle.GetHorizontalFrameSize(), 1)).
			Height(max(m.height-boxStyle.GetVerticalFrameSize(), 1)).
			Render(leftCol)

		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box)
	}

	rightW := m.rightPanelWidth()
	detailFrameStyle := lipgloss.NewStyle().
		Width(rightW).
		Height(m.detailTopHeight())
	var detailView string
	if m.canRenderDetailBox() {
		detailW, detailH := m.detailViewportSize()
		detailStyle := detailFocusStyle().Width(detailW).Height(detailH)
		if m.focusedPanel == focusRight {
			detailStyle = detailStyle.BorderForeground(tui.ColorPrimary)
		} else {
			detailStyle = detailStyle.BorderForeground(tui.ColorMuted)
		}
		detailView = detailFrameStyle.Render(detailStyle.Render(m.detailVP.View()))
	} else {
		detailView = detailFrameStyle.Render(m.detailVP.View())
	}

	responsePlaceholder := lipgloss.NewStyle().
		Width(rightW).
		Height(m.responseAreaHeight()).
		Render("")

	rightCol := lipgloss.JoinVertical(lipgloss.Left, detailView, responsePlaceholder)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	boxStyle := containerStyle()
	box := boxStyle.
		Width(max(m.width-boxStyle.GetHorizontalFrameSize(), 1)).
		Height(max(m.height-boxStyle.GetVerticalFrameSize(), 1)).
		Render(combined)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box)
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
