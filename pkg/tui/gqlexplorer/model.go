package gqlexplorer

import (
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

const (
	itemPadding   = 4
	detailPadding = 6

	noMatchesLabel        = "(no matches)"
	operationFormat       = "%d/%d operations"
	helpLeftPanel         = "Navigate: ↑↓ Ctrl+n/p | G/gg: bottom/top | Enter: detail | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: unfocus/quit"
	helpDetailPanel       = "↑↓ j/k Ctrl+n/p | G/gg: bottom/top | /: search | Space: toggle | Enter: edit | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	helpSearchPanel       = "↑↓ Ctrl+n/p: cycle matches | Enter: done | Esc: cancel"
	helpQueryPanel        = "Navigate: ↑↓ j/k h/l | G/gg: bottom/top | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	helpVariablePanel     = "Navigate: ↑↓ j/k h/l | G/gg: bottom/top | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
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

	inputTypes     map[string]graphql.InputType
	enumTypes      map[string]graphql.EnumType
	objectTypes    map[string]graphql.ObjectType
	unionTypes     map[string]graphql.UnionType
	interfaceTypes map[string]graphql.InterfaceType

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
	focus         tui.FocusRing
	pendingG      bool
	helpBarH      int
}

func NewModel(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
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
	dp := &tui.Panel{Number: 2}
	qp := &tui.Panel{Number: 3}
	vp := &tui.Panel{Number: 4}
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
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "[1] Search: ",
			Placeholder: searchPlaceholderText,
		}),
		detailPanel:   dp,
		variablePanel: vp,
		formCache:     make(map[string]*DetailForm),
		queryPanel:    qp,
		focus:         tui.NewFocusRing([]*tui.Panel{dp, qp, vp}),
		helpBarH:      tui.HelpBarHeight,
	}
	m.focus.SetTyping(true)
	m.syncSearchFocus()
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
		helpQueryPanel, helpVariablePanel, helpEndpointFilter,
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

func (m *Model) variablePanelHeight() int {
	return max(m.contentHeight()*tui.VariablePanelPct/100, 1)
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
	return m.search.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.CopiedMsg:
		// TODO: surface clipboard errors via a status flash once one exists.
		return m, nil
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
		detailW := max(rightW*tui.DetailPanelWPct/100, 1)
		detailH := max(topH*tui.DetailPanelHPct/100, 1)
		m.detailPanel.Resize(detailW, detailH)
		m.queryPanel.Resize(max(rightW-detailW, 1), detailH)
		m.variablePanel.Resize(max(rightW-detailW, 1), variableH)
		m.updateBadgeCache()
		m.syncViewport()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmds []tea.Cmd
	_, cmd := m.search.Update(msg)
	cmds = append(cmds, cmd)
	cmd = m.updateFocusedViewport(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
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
			content, cursorLine := m.detailForm.View(op)
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

		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box)
	}

	rightW := m.rightPanelWidth()

	var detailView, queryView, variableView string
	if m.detailPanel.CanRender() {
		detailView = m.detailPanel.View(m.focus.IsFocused(m.detailPanel))
	}
	if m.queryPanel.CanRender() {
		queryView = m.queryPanel.View(m.focus.IsFocused(m.queryPanel))
	}
	if m.variablePanel.CanRender() {
		variableView = m.variablePanel.View(m.focus.IsFocused(m.variablePanel))
	}

	topRight := lipgloss.JoinHorizontal(lipgloss.Top, detailView, queryView)
	detailW := max(rightW*tui.DetailPanelWPct/100, 1)
	middleRight := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(detailW).Height(m.variablePanelHeight()).Render(""),
		variableView,
	)

	responsePlaceholder := lipgloss.NewStyle().
		Width(rightW).
		Height(m.callAreaHeight()).
		Render("")

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRight, middleRight, responsePlaceholder)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
	body := lipgloss.JoinVertical(lipgloss.Left, combined, helpBar)

	boxH := max(m.height-_containerStyle.GetVerticalFrameSize(), 1)

	box := _containerStyle.
		Height(boxH).
		Render(body)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box)
}

func RunExplorer(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
) error {
	model := NewModel(operations, inputTypes, enumTypes, objectTypes, unionTypes, interfaceTypes)
	p := tea.NewProgram(
		&model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
