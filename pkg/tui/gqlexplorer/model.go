package gqlexplorer

import (
	"fmt"
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

	inputTypes  map[string]graphql.InputType
	enumTypes   map[string]graphql.EnumType
	objectTypes map[string]graphql.ObjectType

	endpoints        []string
	activeEndpoints  map[string]bool
	pickingEndpoints bool
	endpointCursor   int
	pendingEndpoints map[string]bool
	badgeCache       string

	detailPanel   *tui.Panel
	detailForm    *DetailForm
	detailFormKey string
	focus         tui.FocusRing
}

func NewModel(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
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
	dp := &tui.Panel{Number: 2}
	m := Model{
		operations:      operations,
		filtered:        operations,
		filterHint:      buildFilterHint(operations, endpoints),
		endpoints:       endpoints,
		activeEndpoints: active,
		inputTypes:      inputTypes,
		enumTypes:       enumTypes,
		objectTypes:     objectTypes,
		search: tui.NewFilterInput(tui.TextInputOpts{
			Prompt:      "Search: ",
			Placeholder: searchPlaceholderText,
		}),
		detailPanel: dp,
		focus:       tui.NewFocusRing([]*tui.Panel{dp}),
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

func (m *Model) contentHeight() int {
	return max(m.height-_containerStyle.GetVerticalFrameSize(), 1)
}

func (m *Model) detailTopHeight() int {
	return max(m.contentHeight()*tui.DetailTopHeight/100, 1)
}

// responseAreaHeight returns the height allocated to the response area
// (bottom half of the right panel).
func (m *Model) responseAreaHeight() int {
	top := m.detailTopHeight()
	return max(m.contentHeight()-top, 1)
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
	if m.pickingEndpoints || m.focus.LeftFocused() {
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}
	return m.detailPanel.Update(msg)
}

func (m *Model) viewportHeight() int {
	panelW := max(m.leftPanelWidth(), 1)
	headerLines := searchBoxLines
	// Only count the badge row when it will actually be rendered.
	// updateBadgeCache clears badgeCache in narrow terminals, so counting
	// it unconditionally causes a 1-line viewport height mismatch.
	if len(m.activeEndpoints) > 0 && m.hasHeaderContentSpace() {
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
	helpWithScroll := fmt.Sprintf("%s %3.f%%", helpText, m.viewport.ScrollPercent()*100)
	footerLines += wrappedLineCount(helpWithScroll, panelW)
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
		detailOuterW := max(m.rightPanelWidth()*tui.DetailFocusBoxW/100, 1)
		detailOuterH := max(m.detailTopHeight()*tui.DetailFocusBoxH/100, 1)
		m.detailPanel.Resize(detailOuterW, detailOuterH)
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

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pickingEndpoints {
		return m.handleEndpointPickerKey(msg)
	}

	// Global keys that apply regardless of focused panel.
	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit
	case tui.KeyCancel:
		if !m.focus.LeftFocused() && m.detailForm != nil {
			if m.detailForm.ConsumesTextInput() {
				m.detailForm.HandleKey(msg)
				m.syncViewport()
				return m, nil
			}
			m.detailForm.BlurAll()
			m.focus.FocusByNumber(1)
			m.focus.SetTyping(true)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
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
		if !m.focus.LeftFocused() {
			m.focus.FocusByNumber(1)
			m.focus.SetTyping(true)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		return m, tea.Quit
	case tui.KeyTab:
		m.focus.Next()
		if m.focus.LeftFocused() {
			m.focus.SetTyping(true)
		}
		m.syncSearchFocus()
		return m, nil
	case tui.KeyEnter:
		if !m.focus.LeftFocused() && m.detailForm != nil {
			cmd := m.detailForm.HandleKey(msg)
			m.syncViewport()
			return m, cmd
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

	case tui.KeyUp, tui.KeyCtrlP, tui.KeyDown, tui.KeyCtrlN, tui.KeyLeft, tui.KeyRight:
		if !m.focus.LeftFocused() {
			if m.detailForm != nil {
				if m.detailForm.ConsumesTextInput() {
					cmd := m.detailForm.HandleKey(msg)
					m.syncViewport()
					return m, cmd
				}
				switch msg.String() {
				case tui.KeyUp, tui.KeyCtrlP:
					m.detailForm.CursorUp()
				case tui.KeyDown, tui.KeyCtrlN:
					m.detailForm.CursorDown()
				}
				m.syncViewport()
				return m, nil
			}
			cmd := m.detailPanel.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case tui.KeyUp, tui.KeyCtrlP:
			m.cursor = tui.MoveCursorUp(m.cursor)
			m.syncViewport()
		case tui.KeyDown, tui.KeyCtrlN:
			m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
			m.syncViewport()
		}
		return m, nil

	case tui.KeySpace:
		if !m.focus.LeftFocused() && m.detailForm != nil {
			cmd := m.detailForm.HandleKey(msg)
			m.syncViewport()
			return m, cmd
		}
	}

	if !m.focus.Typing() {
		// lazygit style numeber'd border
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
		if m.detailForm != nil && m.detailForm.ConsumesTextInput() {
			cmd := m.detailForm.HandleKey(msg)
			m.syncViewport()
			return m, cmd
		}
		return m, nil
	}

	prevValue := m.search.Model.Value()
	var cmd tea.Cmd
	_, cmd = m.search.Update(msg)
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
		op := &m.filtered[m.cursor]

		formKey := op.Endpoint + "\x1f" + op.Name
		if m.detailFormKey != formKey {
			m.detailForm = buildDetailForm(op, m.enumTypes, m.objectTypes)
			m.detailFormKey = formKey
			m.detailPanel.GotoTop()
		}

		if m.detailForm != nil {
			if m.focus.IsFocused(m.detailPanel) {
				m.detailForm.FocusCurrent()
			} else {
				m.detailForm.BlurAll()
			}
			m.detailPanel.SetContent(m.detailForm.View(op), "")
		} else {
			cacheKey := op.Endpoint + "\x1f" + op.Name + "\x1f" + strconv.Itoa(m.rightPanelWidth())
			if m.detailPanel.SetContent(renderDetail(op, m.inputTypes, m.objectTypes), cacheKey) {
				m.detailPanel.GotoTop()
			}
		}
	} else {
		m.detailForm = nil
		m.detailFormKey = ""
		m.detailPanel.SetContent("", "")
	}
}

func (m *Model) View() string {
	// Compute layout values once per frame instead of calling through
	// method chains repeatedly.
	leftW := m.leftPanelWidth()
	contentH := m.contentHeight()

	leftCol := lipgloss.NewStyle().
		Width(leftW).
		Height(contentH).
		Render(m.renderLeftContent())
	if !m.hasTwoPanelLayout() {
		box := _containerStyle.
			Width(max(m.width-_containerStyle.GetHorizontalFrameSize(), 1)).
			Height(max(m.height-_containerStyle.GetVerticalFrameSize(), 1)).
			Render(leftCol)

		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box)
	}

	rightW := m.rightPanelWidth()
	topH := m.detailTopHeight()
	detailFrameStyle := lipgloss.NewStyle().
		Width(rightW).
		Height(topH)
	var detailView string
	if m.detailPanel.CanRender() {
		detailView = detailFrameStyle.Render(
			m.detailPanel.View(m.focus.IsFocused(m.detailPanel)),
		)
	} else {
		detailView = detailFrameStyle.Render("")
	}

	// Placeholder reserves vertical space for the future response panel.
	// Without it the right column collapses to only the detail viewport height.
	responsePlaceholder := lipgloss.NewStyle().
		Width(rightW).
		Height(m.responseAreaHeight()).
		Render("")

	rightCol := lipgloss.JoinVertical(lipgloss.Left, detailView, responsePlaceholder)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	box := _containerStyle.
		Width(max(m.width-_containerStyle.GetHorizontalFrameSize(), 1)).
		Height(max(m.height-_containerStyle.GetVerticalFrameSize(), 1)).
		Render(combined)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box)
}

func RunExplorer(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
) error {
	model := NewModel(operations, inputTypes, enumTypes, objectTypes)
	p := tea.NewProgram(
		&model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
