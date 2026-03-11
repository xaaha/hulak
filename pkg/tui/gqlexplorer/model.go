package gqlexplorer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

const (
	itemPadding   = 4
	detailPadding = 6

	noMatchesLabel        = "(no matches)"
	operationFormat       = "%d/%d operations"
	helpLeftPanel         = "Navigate: ↑↓ Ctrl+n/p | Enter: detail | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: unfocus/quit"
	helpDetailPanel       = "↑↓ j/k Ctrl+n/p | G/gg: bottom/top | /: search | Space: toggle | Enter: edit | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	helpSearchPanel       = "↑↓ Ctrl+n/p: cycle matches | Enter: done | Esc: cancel"
	helpQueryPanel        = "Navigate: ↑↓ j/k h/l | G/gg: bottom/top | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	helpVariablePanel     = "Navigate: ↑↓ j/k h/l | G/gg: bottom/top | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	helpResponsePanel     = "Navigate: ↑↓ j/k h/l | G/gg: bottom/top | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	searchPlaceholderText = "filter operations..."
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

// Cached styles — these never change at runtime, so building them once
// at package init avoids repeated allocations per View() frame.
var _containerStyle = tui.BoxStyle.Padding(0, 1)

type ExplorerData struct {
	Operations     []UnifiedOperation
	InputTypes     map[string]graphql.InputType
	EnumTypes      map[string]graphql.EnumType
	ObjectTypes    map[string]graphql.ObjectType
	UnionTypes     map[string]graphql.UnionType
	InterfaceTypes map[string]graphql.InterfaceType
	APIInfos       map[string]yamlparser.APIInfo
}

type RefreshPayload struct {
	Data     ExplorerData
	Warnings []string
}

type RefreshFunc func() (RefreshPayload, error)

type refreshLoadedMsg struct {
	payload RefreshPayload
	err     error
}

type queryExecutedMsg struct {
	resp apicalls.CustomResponse
}

type queryErrorMsg struct {
	err error
}

// Model is the full-screen GraphQL explorer TUI.
type Model struct {
	operations []UnifiedOperation
	filtered   []UnifiedOperation
	cursor     int
	filterHint string
	mouse      tui.MouseZone
	search     tui.TextInput
	viewport   viewport.Model
	ready      bool
	width      int
	height     int

	inputTypes     map[string]graphql.InputType
	enumTypes      map[string]graphql.EnumType
	objectTypes    map[string]graphql.ObjectType
	unionTypes     map[string]graphql.UnionType
	interfaceTypes map[string]graphql.InterfaceType
	apiInfos       map[string]yamlparser.APIInfo

	endpoints       []string
	activeEndpoints map[string]bool
	endpointCursor  int
	badgeCache      string

	detailPanel   *tui.Panel
	variablePanel *tui.Panel
	detailForm    *DetailForm
	detailFormKey string
	formCache     map[string]*DetailForm
	queryPanel    *tui.Panel
	responsePanel *tui.Panel
	responseBody  string
	executing     bool
	focus         tui.FocusRing
	pendingG      bool
	helpBarH      int
	refreshFn     RefreshFunc
	refreshing    bool
	notification  tui.NotificationCenter
	actionRow     tui.ActionRow
	initCmd       tea.Cmd
}

func NewModel(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
	apiInfos map[string]yamlparser.APIInfo,
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
	// numbers for navigation.
	dp := &tui.Panel{Number: 2, Label: "Form"}
	qp := &tui.Panel{Number: 3, Label: "Query"}
	vp := &tui.Panel{Number: 4, Label: "Variables"}
	rp := &tui.Panel{Number: 5, Label: "Response"}
	m := Model{
		operations:      operations,
		filtered:        operations,
		filterHint:      buildFilterHint(operations, endpoints),
		endpoints:       endpoints,
		activeEndpoints: active,
		inputTypes:      inputTypes,
		enumTypes:       enumTypes,
		objectTypes:     objectTypes,
		unionTypes:      unionTypes,
		interfaceTypes:  interfaceTypes,
		apiInfos:        apiInfos,
		mouse:           tui.NewMouseZone(),
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "[1] Search: ",
			Placeholder: searchPlaceholderText,
		}),
		detailPanel:   dp,
		variablePanel: vp,
		formCache:     make(map[string]*DetailForm),
		queryPanel:    qp,
		responsePanel: rp,
		focus:         tui.NewFocusRing([]*tui.Panel{dp, qp, vp, rp}),
		helpBarH:      tui.HelpBarHeight,
		notification:  tui.NewNotificationCenter(),
		actionRow:     tui.NewActionRow(),
	}
	m.focus.SetTyping(true)
	m.syncSearchFocus()
	m.updateActionRow()
	return m
}

// hasHeaderContentSpace guards optional header UI that is visually noisy in
// narrow terminals (badge row + placeholder hint).
// When space is limited, returning false keeps the search row stable by hiding those extras.
func (m *Model) hasHeaderContentSpace() bool {
	return m.width >= minHeaderContentWidth
}

func (m *Model) updateSearchPlaceholder() {
	if m.hasHeaderContentSpace() {
		m.search.Model.Placeholder = searchPlaceholderText
		return
	}
	m.search.Model.Placeholder = ""
}

func (m *Model) leftPanelWidth() int {
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

func (m *Model) rightPanelWidth() int {
	if !m.hasTwoPanelLayout() {
		return 0
	}
	return max(m.contentWidth()-m.leftPanelWidth(), 0)
}

func (m *Model) hasTwoPanelLayout() bool {
	return m.contentWidth() >= tui.MinLeftPanelWidth+tui.MinRightPanelWidth
}

func (m *Model) contentWidth() int {
	return max(m.width-_containerStyle.GetHorizontalFrameSize(), 1)
}

func (m *Model) updateHelpBarHeight() {
	contentW := m.contentWidth()
	m.helpBarH = 1
	for _, h := range []string{
		helpLeftPanel, helpDetailPanel, helpSearchPanel,
		helpQueryPanel, helpVariablePanel, helpResponsePanel, helpEndpointFilter,
	} {
		rendered := tui.HelpBarStyle.Render(
			lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center).Render(h),
		)
		if lines := lipgloss.Height(rendered); lines > m.helpBarH {
			m.helpBarH = lines
		}
	}
}

func (m *Model) contentHeight() int {
	return max(m.height-_containerStyle.GetVerticalFrameSize()-m.helpBarH, 1)
}

func (m *Model) detailTopHeight() int {
	return max(m.contentHeight()*tui.DetailTopPct/100, 1)
}

func (m *Model) detailPanelWidth(rightW int) int {
	return max(rightW/2, 1)
}

// gql variable panel height, 2/3 of the remaining height below the top row
func (m *Model) variablePanelHeight() int {
	remaining := max(m.contentHeight()-m.detailTopHeight(), 1)
	return max((remaining*2)/3, 1)
}

// callAreaHeight returns the remaining height allocated to extras
// below the variable panel. I am calling it callAreaheight
func (m *Model) callAreaHeight() int {
	return max(m.contentHeight()-m.detailTopHeight()-m.variablePanelHeight(), 1)
}

func (m *Model) updateBadgeCache() {
	if !m.hasHeaderContentSpace() {
		m.badgeCache = ""
		return
	}
	m.badgeCache = m.renderBadges()
}

func (m *Model) syncSearchFocus() {
	if m.focus.LeftFocused() && m.focus.Typing() {
		m.search.Model.Focus()
		return
	}
	m.search.Model.Blur()
}

func (m *Model) updateFocusedViewport(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.focus.LeftFocused() {
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}
	if m.focus.IsFocused(m.queryPanel) {
		return m.queryPanel.Update(msg)
	}
	if m.focus.IsFocused(m.variablePanel) {
		return m.variablePanel.Update(msg)
	}
	if m.focus.IsFocused(m.responsePanel) {
		return m.responsePanel.Update(msg)
	}
	return m.detailPanel.Update(msg)
}

func (m *Model) viewportHeight() int {
	panelW := max(m.leftPanelWidth(), 1)
	headerLines := tui.SearchBoxHeight
	// Only count the badge row when it will actually be rendered.
	// updateBadgeCache clears badgeCache in narrow terminals, so counting
	// it unconditionally causes a 1-line viewport height mismatch.
	if len(m.activeEndpoints) > 0 && m.hasHeaderContentSpace() {
		headerLines++
	}
	if m.filterHint != "" {
		headerLines += wrappedLineCount(m.filterHint, panelW)
	}
	footerLines := tui.StatusRowHeight
	h := max(m.contentHeight()-headerLines-footerLines, 1)
	return h
}

// wrappedLineCount returns how many visual lines text occupies at the given
// width. It performs a full lipgloss render internally, which is fine for
// short strings (help text, filter hint) but would be a concern for longer content.
func wrappedLineCount(text string, width int) int {
	if width <= 0 || text == "" {
		return 0
	}
	return lipgloss.Height(lipgloss.NewStyle().Width(width).Render(text))
}

func (m *Model) Init() tea.Cmd {
	if m.initCmd != nil {
		return tea.Batch(m.search.Init(), m.initCmd)
	}
	return m.search.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.CopiedMsg:
		// TODO: surface clipboard errors via a status flash once one exists.
		return m, nil
	case refreshLoadedMsg:
		m.refreshing = false
		m.updateActionRow()
		if msg.err != nil {
			cmd := m.enqueueNotification(tui.NotificationError, msg.err.Error())
			return m, cmd
		}
		m.applyRefreshPayload(&msg.payload)
		if len(msg.payload.Warnings) > 0 {
			cmd := m.enqueueNotification(tui.NotificationWarn, joinWarnings(msg.payload.Warnings))
			return m, cmd
		}
		return m, nil
	case queryExecutedMsg:
		m.executing = false
		m.updateActionRow()
		m.handleQueryExecuted(msg)
		return m, nil
	case queryErrorMsg:
		m.executing = false
		m.updateActionRow()
		cmd := m.enqueueNotification(tui.NotificationError, msg.err.Error())
		return m, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateHelpBarHeight()
		m.updateSearchPlaceholder()
		panelW := m.leftPanelWidth()
		listHeight := m.viewportHeight()
		searchFrame := tui.InputStyle.GetHorizontalFrameSize()
		m.search.Model.Width = max(panelW-searchFrame-len(m.search.Model.Prompt), 1)
		if !m.ready {
			m.viewport = viewport.New(panelW, listHeight)
			m.viewport.MouseWheelEnabled = true
			m.ready = true
		} else {
			m.viewport.Width = panelW
			m.viewport.Height = listHeight
		}
		rightW := m.rightPanelWidth()
		topH := m.detailTopHeight()
		variableH := m.variablePanelHeight()
		detailW := m.detailPanelWidth(rightW) // split the top row evenly in half
		detailH := topH                       // detail panel height uses the full top-row height
		m.detailPanel.Resize(detailW, detailH)
		m.queryPanel.Resize(max(rightW-detailW, 1), detailH)
		m.variablePanel.Resize(max(rightW-detailW, 1), variableH)
		m.responsePanel.Resize(detailW, max(m.contentHeight()-topH, 1))
		m.updateBadgeCache()
		m.updateActionRow()
		m.syncViewport()
		return m, nil
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if cmd := m.notification.Update(msg); cmd != nil {
		m.updateActionRow()
		return m, cmd
	}

	var cmds []tea.Cmd
	_, cmd := m.search.Update(msg)
	cmds = append(cmds, cmd)
	cmd = m.updateFocusedViewport(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.notification.Visible() {
		return m, nil
	}
	if tui.IsLeftClick(msg) {
		if cmd, ok := m.handleBottomRowClick(msg); ok {
			return m, cmd
		}
		if m.handleLeftPanelClick(msg) {
			return m, nil
		}
		if m.handleDetailFormClick(msg) {
			return m, nil
		}
	}

	var cmds []tea.Cmd
	_, cmd := m.search.Update(msg)
	cmds = append(cmds, cmd)
	cmd = m.updateFocusedViewport(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleBottomRowClick(msg tea.MouseMsg) (tea.Cmd, bool) {
	id, ok := m.actionRow.HandleMouse(msg)
	if !ok {
		return nil, false
	}
	return m.handleBottomAction(id), true
}

func (m *Model) handleLeftPanelClick(msg tea.MouseMsg) bool {
	if !m.ready {
		return false
	}

	if tui.Hit(m.searchZoneID(), msg) {
		m.focus.FocusByNumber(1)
		m.focus.SetTyping(true)
		m.syncSearchFocus()
		m.syncViewport()
		return true
	}

	if m.isEndpointMode() {
		eps := m.filteredEndpoints()
		for i := range eps {
			if !tui.Hit(m.endpointZoneID(i), msg) {
				continue
			}
			m.focus.FocusByNumber(1)
			m.focus.SetTyping(false)
			m.endpointCursor = i
			m.syncSearchFocus()
			if m.isNegatedEndpointSearch() {
				keep := make(map[string]bool, len(eps))
				for _, ep := range eps {
					keep[ep] = true
				}
				m.activeEndpoints = keep
			} else {
				ep := eps[i]
				if m.activeEndpoints[ep] {
					delete(m.activeEndpoints, ep)
				} else {
					m.activeEndpoints[ep] = true
				}
			}
			m.updateBadgeCache()
			m.applyFilter()
			m.syncViewport()
			return true
		}
		return false
	}

	for i := range m.filtered {
		if !tui.Hit(m.operationZoneID(i), msg) {
			continue
		}
		m.focus.FocusByNumber(1)
		m.focus.SetTyping(false)
		m.syncSearchFocus()
		m.cursor = i
		m.syncViewport()
		return true
	}
	return false
}

func (m *Model) handleDetailFormClick(msg tea.MouseMsg) bool {
	if m.detailForm == nil || m.isEndpointMode() {
		return false
	}
	if !m.detailForm.HandleMouse(m.detailMousePrefix(), msg) {
		return false
	}
	m.focus.FocusByNumber(m.detailPanel.Number)
	m.syncSearchFocus()
	m.syncViewport()
	return true
}

func (m *Model) operationZoneID(index int) string {
	return m.mouse.ID("operation", strconv.Itoa(index))
}

func (m *Model) endpointZoneID(index int) string {
	return m.mouse.ID("endpoint", strconv.Itoa(index))
}

func (m *Model) detailMousePrefix() string {
	return m.mouse.ID("detail")
}

func (m *Model) searchZoneID() string {
	return m.mouse.ID("search")
}

var vimToArrowMap = map[string]tea.KeyType{
	tui.KeyJ: tea.KeyDown,
	tui.KeyK: tea.KeyUp,
	tui.KeyH: tea.KeyLeft,
	tui.KeyL: tea.KeyRight,
}

func vimToArrow(msg tea.KeyMsg) tea.KeyMsg {
	if arrow, ok := vimToArrowMap[msg.String()]; ok {
		return tea.KeyMsg{Type: arrow}
	}
	return msg
}

func (m *Model) forwardKeyToForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd := m.detailForm.HandleKey(msg)
	m.syncViewport()
	return m, cmd
}

func (m *Model) handleDetailFormNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.detailForm.hasExpandedDropdown() {
		return m.forwardKeyToForm(msg)
	}
	switch msg.String() {
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyK:
		m.detailForm.CursorUp()
	case tui.KeyDown, tui.KeyCtrlN, tui.KeyJ:
		m.detailForm.CursorDown()
	case tui.KeyLeft, tui.KeyRight:
		cmd := m.detailPanel.Update(msg)
		return m, cmd
	}
	m.syncViewport()
	return m, nil
}

func (m *Model) jumpToEdge(top bool) {
	switch {
	case m.focus.IsFocused(m.queryPanel):
		if top {
			m.queryPanel.GotoTop()
		} else {
			m.queryPanel.GotoBottom()
		}
	case m.focus.IsFocused(m.variablePanel):
		if top {
			m.variablePanel.GotoTop()
		} else {
			m.variablePanel.GotoBottom()
		}
	case m.focus.IsFocused(m.responsePanel):
		if top {
			m.responsePanel.GotoTop()
		} else {
			m.responsePanel.GotoBottom()
		}
	case m.focus.IsFocused(m.detailPanel) && m.detailForm != nil:
		if top {
			m.detailForm.CursorToTop()
		} else {
			m.detailForm.CursorToBottom()
		}
		m.syncViewport()
	case m.focus.LeftFocused():
		if top {
			m.cursor = 0
		} else {
			m.cursor = max(len(m.filtered)-1, 0)
		}
		m.syncViewport()
	}
}

func (m *Model) switchPanel(key string) {
	if key == tui.KeyShiftTab {
		m.focus.Prev()
	} else {
		m.focus.Next()
	}
	if m.focus.LeftFocused() {
		m.focus.SetTyping(true)
	}
	m.syncSearchFocus()
	m.syncViewport()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.notification.Visible() {
		switch msg.String() {
		case tui.KeyAt, tui.KeyCancel, "q":
			if _, handled := m.actionRow.HandleKey(msg.String()); handled {
				cmd := m.handleBottomAction("badge")
				return m, cmd
			}
			if m.notification.ToggleLast() {
				m.updateActionRow()
			}
			return m, nil
		case tui.KeyYank:
			if text := m.notification.CopyText(); text != "" {
				return m, tui.CopyToClipboard(text)
			}
			return m, nil
		default:
			return m, nil
		}
	}
	if msg.String() == tui.KeyRefresh {
		cmd := m.startRefresh()
		return m, cmd
	}
	if msg.String() == tui.KeySend {
		cmd := m.executeQuery()
		return m, cmd
	}
	if msg.String() == tui.KeyAt && !m.search.Model.Focused() &&
		(m.detailForm == nil || !m.detailForm.ConsumesTextInput()) {
		if _, handled := m.actionRow.HandleKey(msg.String()); handled {
			cmd := m.handleBottomAction("badge")
			return m, cmd
		}
	}

	if m.pendingG {
		m.pendingG = false
		if msg.String() == tui.KeyG {
			m.jumpToEdge(true)
			return m, nil
		}
	}

	if m.focus.LeftFocused() && m.isEndpointMode() {
		if m.handleEndpointKey(msg) {
			return m, nil
		}
	}

	if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil && m.detailForm.IsSearching() {
		if msg.String() == tui.KeyQuit {
			return m, tea.Quit
		}
		cmd := m.detailForm.HandleSearchKey(msg)
		m.syncViewport()
		return m, cmd
	}

	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit

		// Esc: step backward one panel at a time
		// variable panel → query panel → detail panel → search (left) → quit
	case tui.KeyCancel:
		// Detail panel: close dropdown first, exit text editing, then step back.
		if m.focus.IsFocused(m.detailPanel) {
			if m.detailForm != nil && m.detailForm.hasExpandedDropdown() {
				m.detailForm.HandleKey(msg)
				m.syncViewport()
				return m, nil
			}
			if m.detailForm != nil && m.detailForm.ConsumesTextInput() {
				m.detailForm.HandleKey(msg)
				m.syncViewport()
				return m, nil
			}
			if m.detailForm != nil {
				m.detailForm.BlurAll()
			}
			m.focus.FocusByNumber(1)
			m.focus.SetTyping(true)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Response panel: step back to variable panel.
		if m.focus.IsFocused(m.responsePanel) {
			m.focus.FocusByNumber(m.variablePanel.Number)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Variable panel: step back to query panel.
		if m.focus.IsFocused(m.variablePanel) {
			m.focus.FocusByNumber(m.queryPanel.Number)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Query panel: step back to detail panel.
		if m.focus.IsFocused(m.queryPanel) {
			m.focus.FocusByNumber(m.detailPanel.Number)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Left panel (search): clear text → blur → quit.
		if m.focus.Typing() {
			if m.search.Model.Value() != "" {
				m.search.Model.Reset()
				m.applyFilterAndReset()
				return m, nil
			}
			m.focus.SetTyping(false)
			m.syncSearchFocus()
			return m, nil
		}
		return m, tea.Quit

	// ── Tab / Shift+Tab: cycle panels ───────────────────────────
	case tui.KeyTab, tui.KeyShiftTab:
		m.switchPanel(msg.String())
		return m, nil
	// ── Enter: detail panel form input / left panel → detail ────
	case tui.KeyEnter:
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil {
			return m.forwardKeyToForm(msg)
		}
		if m.focus.LeftFocused() {
			if !m.focus.Typing() {
				m.focus.SetTyping(true)
				m.syncSearchFocus()
				return m, nil
			}
			if m.hasTwoPanelLayout() {
				m.focus.FocusByNumber(m.detailPanel.Number)
				m.syncSearchFocus()
				m.syncViewport()
			}
		}
		return m, nil

	// ── Arrow / vim keys: per-panel navigation ──────────────────
	// Query panel: scroll viewport (j/k vertical, h/l horizontal).
	// Detail panel: navigate form items or scroll.
	// Left panel: move operation cursor.
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyDown, tui.KeyCtrlN, tui.KeyLeft, tui.KeyRight,
		tui.KeyK, tui.KeyJ, tui.KeyH, tui.KeyL, tui.KeyG, tui.KeyShiftG:
		if (msg.String() == tui.KeyLeft || msg.String() == tui.KeyRight) &&
			m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
			m.detailForm.ConsumesTextInput() {
			return m.forwardKeyToForm(msg)
		}
		if (msg.String() == tui.KeyLeft || msg.String() == tui.KeyRight) &&
			m.focus.LeftFocused() && m.focus.Typing() {
			var cmd tea.Cmd
			_, cmd = m.search.Update(msg)
			return m, cmd
		}
		if msg.String() == tui.KeyJ || msg.String() == tui.KeyK ||
			msg.String() == tui.KeyH || msg.String() == tui.KeyL ||
			msg.String() == tui.KeyG || msg.String() == tui.KeyShiftG {
			if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
				m.detailForm.ConsumesTextInput() {
				return m.forwardKeyToForm(msg)
			}
			if m.focus.LeftFocused() && m.focus.Typing() {
				break
			}
		}
		if msg.String() == tui.KeyShiftG {
			m.jumpToEdge(false)
			return m, nil
		}
		if msg.String() == tui.KeyG {
			m.pendingG = true
			return m, nil
		}
		// Query panel: scroll viewport. Vim keys are mapped to arrows
		// because the bubbles viewport only understands arrow key types.
		if m.focus.IsFocused(m.queryPanel) {
			cmd := m.queryPanel.Update(vimToArrow(msg))
			return m, cmd
		}
		// Variable panel: scroll viewport. Vim keys are mapped to arrows
		// because the bubbles viewport only understands arrow key types.
		if m.focus.IsFocused(m.variablePanel) {
			cmd := m.variablePanel.Update(vimToArrow(msg))
			return m, cmd
		}
		if m.focus.IsFocused(m.responsePanel) {
			cmd := m.responsePanel.Update(vimToArrow(msg))
			return m, cmd
		}
		// Detail panel: navigate form or scroll.
		if !m.focus.LeftFocused() {
			if m.detailForm != nil {
				return m.handleDetailFormNavigation(msg)
			}
			cmd := m.detailPanel.Update(msg)
			return m, cmd
		}
		// Left panel: move operation list cursor.
		switch msg.String() {
		case tui.KeyUp, tui.KeyCtrlP, tui.KeyK:
			m.cursor = tui.MoveCursorUp(m.cursor)
			m.syncViewport()
		case tui.KeyDown, tui.KeyCtrlN, tui.KeyJ:
			m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
			m.syncViewport()
		}
		return m, nil

	// Space: detail panel field toggle
	case tui.KeySpace:
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil {
			return m.forwardKeyToForm(msg)
		}

	// Slash: vim-style search in detail form
	case tui.KeySlash:
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
			!m.detailForm.ConsumesTextInput() {
			m.detailForm.StartSearch()
			m.syncViewport()
			return m, nil
		}

	// Yank: copy focused panel content to system clipboard
	case tui.KeyYank:
		if text := m.notification.CopyText(); text != "" {
			return m, tui.CopyToClipboard(text)
		}
		if text := m.yankText(); text != "" {
			return m, tui.CopyToClipboard(text)
		}
		return m, nil
	}

	if !m.focus.Typing() && (m.detailForm == nil || !m.detailForm.ConsumesTextInput()) {
		if key := msg.String(); len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			num := int(key[0] - '0')
			if m.focus.FocusByNumber(num) {
				if m.focus.LeftFocused() {
					m.focus.SetTyping(true)
				}
				m.syncSearchFocus()
			}
			return m, nil
		}
	}

	if !m.focus.LeftFocused() {
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
			m.detailForm.ConsumesTextInput() {
			return m.forwardKeyToForm(msg)
		}
		return m, nil
	}

	prevValue := m.search.Model.Value()
	var cmd tea.Cmd
	_, cmd = m.search.Update(msg)
	newValue := m.search.Model.Value()
	if newValue != prevValue {
		if m.isEndpointMode() {
			m.endpointCursor = 0
		}
		m.applyFilterAndReset()
	}
	return m, cmd
}

func (m *Model) SetRefresh(fn RefreshFunc) {
	m.refreshFn = fn
	m.updateActionRow()
}

func (m *Model) SetInitialWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	m.initCmd = m.enqueueNotification(tui.NotificationWarn, joinWarnings(warnings))
}

func (m *Model) startRefresh() tea.Cmd {
	if m.refreshFn == nil || m.refreshing {
		return nil
	}
	m.refreshing = true
	m.updateActionRow()
	refreshFn := m.refreshFn
	return func() tea.Msg {
		payload, err := refreshFn()
		return refreshLoadedMsg{payload: payload, err: err}
	}
}

func (m *Model) executeQuery() tea.Cmd {
	if m.executing {
		return nil
	}
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	op := &m.filtered[m.cursor]

	if op.Type == TypeSubscription {
		return m.enqueueNotification(tui.NotificationWarn, "Subscriptions cannot be executed from the explorer")
	}

	info, ok := m.apiInfos[op.Endpoint]
	if !ok {
		return m.enqueueNotification(tui.NotificationError, "No API configuration found for "+op.Endpoint)
	}

	query := BuildQueryString(op, m.detailForm)
	if query == "" {
		return m.enqueueNotification(tui.NotificationError, "Empty query")
	}

	varsMap := BuildVariablesMap(op, m.detailForm)
	apiInfo := yamlparser.CloneAPIInfo(info)

	body, err := yamlparser.EncodeGraphQlBody(query, varsMap)
	if err != nil {
		return m.enqueueNotification(tui.NotificationError, "Failed to encode query: "+err.Error())
	}
	apiInfo.Body = body

	m.executing = true
	m.responsePanel.SetContent(tui.HelpStyle.Render("Executing..."), "")
	m.responsePanel.Footer = ""
	m.updateActionRow()

	return func() tea.Msg {
		resp, err := apicalls.StandardCall(apiInfo, false)
		if err != nil {
			return queryErrorMsg{err: err}
		}
		return queryExecutedMsg{resp: resp}
	}
}

func (m *Model) handleQueryExecuted(msg queryExecutedMsg) {
	var bodyJSON []byte
	if msg.resp.Response != nil && msg.resp.Response.Body != nil {
		bodyJSON, _ = json.MarshalIndent(msg.resp.Response.Body, "", "  ")
	}
	if len(bodyJSON) == 0 {
		bodyJSON, _ = json.MarshalIndent(msg.resp, "", "  ")
	}
	m.responseBody = string(bodyJSON)

	colored, err := utils.FormatJSONColored(bodyJSON, utils.LipglossColorProvider{})
	if err != nil {
		m.responsePanel.SetContent(m.responseBody, "")
	} else {
		m.responsePanel.SetContent(colored, "")
	}
	m.responsePanel.GotoTop()

	var parts []string
	if msg.resp.Response != nil {
		parts = append(parts, msg.resp.Response.Status)
	}
	if msg.resp.Duration != "" {
		parts = append(parts, msg.resp.Duration)
	}
	if len(parts) > 0 {
		m.responsePanel.Footer = strings.Join(parts, "  ")
	}
}

func (m *Model) handleBottomAction(id string) tea.Cmd {
	switch id {
	case "badge":
		if m.notification.ToggleLast() {
			m.updateActionRow()
		}
		return nil
	case "refresh":
		return m.startRefresh()
	case "send":
		return m.executeQuery()
	default:
		return nil
	}
}

func (m *Model) enqueueNotification(severity tui.NotificationSeverity, message string) tea.Cmd {
	cmd := m.notification.Enqueue(severity, message)
	m.updateActionRow()
	return cmd
}

func (m *Model) applyRefreshPayload(payload *RefreshPayload) {
	m.operations = payload.Data.Operations
	m.filtered = payload.Data.Operations
	m.inputTypes = payload.Data.InputTypes
	m.enumTypes = payload.Data.EnumTypes
	m.objectTypes = payload.Data.ObjectTypes
	m.unionTypes = payload.Data.UnionTypes
	m.interfaceTypes = payload.Data.InterfaceTypes
	m.apiInfos = payload.Data.APIInfos
	m.endpoints = collectEndpoints(m.operations)
	m.activeEndpoints = make(map[string]bool, len(m.endpoints))
	for _, ep := range m.endpoints {
		m.activeEndpoints[ep] = true
	}
	m.filterHint = buildFilterHint(m.operations, m.endpoints)
	m.cursor = 0
	m.endpointCursor = 0
	m.formCache = make(map[string]*DetailForm)
	m.detailForm = nil
	m.detailFormKey = ""
	m.updateBadgeCache()
	m.applyFilterAndReset()
}

func (m *Model) canSend() bool {
	if m.executing {
		return false
	}
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return false
	}
	op := &m.filtered[m.cursor]
	if op.Type == TypeSubscription {
		return false
	}
	_, ok := m.apiInfos[op.Endpoint]
	return ok
}

func (m *Model) updateActionRow() {
	items := []tui.ActionItem{
		{
			ID:      "refresh",
			Label:   "Refresh  ctrl+r",
			Key:     tui.KeyRefresh,
			Enabled: m.refreshFn != nil && !m.refreshing,
		},
		{
			ID:      "send",
			Label:   "Send     ctrl+g",
			Key:     tui.KeySend,
			Enabled: m.canSend(),
		},
		{
			ID:      "save",
			Label:   "Save     ctrl+s",
			Key:     tui.KeySave,
			Enabled: false,
		},
	}
	m.actionRow.SetItems(items)
	m.actionRow.SetBadge(tui.ActionBadge{
		Label:    "Notification @",
		Key:      tui.KeyAt,
		Severity: m.notification.Severity(),
		Visible:  m.notification.HasLast(),
	})
}

func joinWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	if len(warnings) == 1 {
		return warnings[0]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d schema warnings:\n", len(warnings))
	for i, warning := range warnings {
		fmt.Fprintf(&b, "%d. %s", i+1, warning)
		if i < len(warnings)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m *Model) yankText() string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	op := &m.filtered[m.cursor]
	switch {
	case m.focus.IsFocused(m.queryPanel):
		return BuildQueryString(op, m.detailForm)
	case m.focus.IsFocused(m.variablePanel):
		return BuildVariablesString(op, m.detailForm)
	case m.focus.IsFocused(m.responsePanel):
		return m.responseBody
	case m.focus.IsFocused(m.detailPanel):
		return m.detailPanelPlainText(op)
	case m.focus.LeftFocused():
		return formatOperationSummary(op)
	}
	return ""
}

func (m *Model) detailPanelPlainText(op *UnifiedOperation) string {
	var styled string
	if m.detailForm != nil {
		styled, _ = m.detailForm.View(op)
	} else {
		styled = renderDetail(op, m.inputTypes, m.objectTypes, m.unionTypes, m.interfaceTypes)
	}
	return ansi.Strip(styled)
}

func formatOperationSummary(op *UnifiedOperation) string {
	var b strings.Builder
	b.WriteString(op.Name)
	if op.Description != "" {
		b.WriteString("\n  ")
		b.WriteString(op.Description)
	}
	if op.Endpoint != "" {
		b.WriteString("\n  ")
		b.WriteString(op.Endpoint)
	}
	return b.String()
}

func (m *Model) applyFilterAndReset() {
	m.applyFilter()
	m.viewport.GotoTop()
	m.syncViewport()
}

func (m *Model) syncViewport() {
	var content string
	var cursorLine int
	if m.isEndpointMode() {
		content, cursorLine = m.renderEndpointPicker()
	} else {
		content, cursorLine = m.renderList()
	}
	tui.SyncViewport(&m.viewport, content, cursorLine, tui.DefaultScrollMargin)

	if m.isEndpointMode() {
		return
	}

	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		op := &m.filtered[m.cursor]

		formKey := op.Endpoint + "\x1f" + op.Name
		if m.detailFormKey != formKey {
			if m.detailForm != nil && m.detailFormKey != "" {
				m.formCache[m.detailFormKey] = m.detailForm
			}
			if cached, ok := m.formCache[formKey]; ok {
				m.detailForm = cached
			} else {
				m.detailForm = buildDetailForm(op, m.inputTypes, m.enumTypes, m.objectTypes, m.unionTypes, m.interfaceTypes)
			}
			m.detailFormKey = formKey
			m.detailPanel.GotoTop()
		}

		if m.detailForm != nil {
			if m.focus.IsFocused(m.detailPanel) {
				m.detailForm.FocusCurrent()
			} else {
				m.detailForm.BlurAll()
			}
			content, cursorLine := m.detailForm.ViewMarked(op, m.detailMousePrefix(), m.mouse.Mark)
			m.detailPanel.SyncContent(content, cursorLine)
			m.detailPanel.Footer = m.detailForm.SearchFooter()
		} else {
			cacheKey := op.Endpoint + "\x1f" + op.Name + "\x1f" + strconv.Itoa(m.rightPanelWidth())
			if m.detailPanel.SetContent(renderDetail(op, m.inputTypes, m.objectTypes, m.unionTypes, m.interfaceTypes), cacheKey) {
				m.detailPanel.GotoTop()
			}
		}

		m.queryPanel.SetContent(BuildQueryString(op, m.detailForm), "")
		m.variablePanel.SetContent(BuildVariablesString(op, m.detailForm), "")
	} else {
		m.detailForm = nil
		m.detailFormKey = ""
		m.detailPanel.Footer = ""
		m.detailPanel.SetContent("", "")
		m.queryPanel.SetContent("", "")
		m.variablePanel.SetContent("", "")
		m.responsePanel.SetContent("", "")
	}
}

func (m *Model) renderHelpBar(width int) string {
	var raw string
	switch {
	case m.focus.LeftFocused() && m.isEndpointMode():
		raw = helpEndpointFilter
	case m.focus.IsFocused(m.queryPanel):
		raw = helpQueryPanel
	case m.focus.IsFocused(m.variablePanel):
		raw = helpVariablePanel
	case m.focus.IsFocused(m.responsePanel):
		raw = helpResponsePanel
	case m.focus.IsFocused(m.detailPanel) && m.detailForm != nil && m.detailForm.IsSearching():
		raw = helpSearchPanel
	case m.focus.IsFocused(m.detailPanel):
		raw = helpDetailPanel
	default:
		raw = helpLeftPanel
	}
	return tui.HelpBarStyle.Render(
		lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(raw),
	)
}

func (m *Model) View() string {
	leftW := m.leftPanelWidth()
	contentW := m.contentWidth()
	contentH := m.contentHeight()

	helpBar := m.renderHelpBar(contentW)

	leftCol := lipgloss.NewStyle().
		Width(leftW).
		Height(contentH).
		Render(m.renderLeftContent())
	if !m.hasTwoPanelLayout() {
		body := lipgloss.JoinVertical(lipgloss.Left, leftCol, helpBar)
		box := _containerStyle.
			Width(max(m.width-_containerStyle.GetHorizontalFrameSize(), 1)).
			Height(max(m.height-_containerStyle.GetVerticalFrameSize(), 1)).
			Render(body)

		return tui.ScanMouseZones(
			lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box),
		)
	}

	rightW := m.rightPanelWidth()

	var detailView, queryView, variableView, responseView string
	if m.detailPanel.CanRender() {
		detailView = m.detailPanel.View(m.focus.IsFocused(m.detailPanel))
	}
	if m.queryPanel.CanRender() {
		queryView = m.queryPanel.View(m.focus.IsFocused(m.queryPanel))
	}
	if m.variablePanel.CanRender() {
		variableView = m.variablePanel.View(m.focus.IsFocused(m.variablePanel))
	}
	if m.responsePanel.CanRender() {
		responseView = m.responsePanel.View(m.focus.IsFocused(m.responsePanel))
	}

	topRight := lipgloss.JoinHorizontal(lipgloss.Top, detailView, queryView)
	detailW := m.detailPanelWidth(rightW)
	queryW := max(rightW-detailW, 1)

	actionsView := lipgloss.NewStyle().
		Width(queryW).
		Height(m.callAreaHeight()).
		Render(m.renderActionsPanel(queryW, m.callAreaHeight()))
	rightBottom := lipgloss.JoinVertical(lipgloss.Left, variableView, actionsView)
	bottomSection := lipgloss.JoinHorizontal(lipgloss.Top, responseView, rightBottom)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRight, bottomSection)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
	body := lipgloss.JoinVertical(lipgloss.Left, combined, helpBar)

	boxH := max(m.height-_containerStyle.GetVerticalFrameSize(), 1)

	box := _containerStyle.
		Height(boxH).
		Render(body)

	if m.notification.Visible() {
		box = tui.OverlayCenter(
			m.notification.RenderModal(max(m.width-8, 1), max(m.height-6, 1)),
			m.width,
			m.height,
		)
	}

	return tui.ScanMouseZones(lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box))
}

func RunExplorer(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
) error {
	model := NewModel(operations, inputTypes, enumTypes, objectTypes, unionTypes, interfaceTypes, make(map[string]yamlparser.APIInfo))
	return runExplorerModel(&model)
}

func RunExplorerWithRefresh(
	data ExplorerData,
	refreshFn RefreshFunc,
	initialWarnings []string,
) error {
	model := NewModel(
		data.Operations,
		data.InputTypes,
		data.EnumTypes,
		data.ObjectTypes,
		data.UnionTypes,
		data.InterfaceTypes,
		data.APIInfos,
	)
	model.SetRefresh(refreshFn)
	model.SetInitialWarnings(initialWarnings)
	return runExplorerModel(&model)
}

func runExplorerModel(model *Model) error {
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
