package gqlexplorer

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
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
	helpResponsePanel     = "Navigate: ↑↓ j/k h/l | G/gg: bottom/top | /: search | Ctrl+s: save | Tab/Shift+Tab: switch | Ctrl+y: copy | Esc: back"
	searchPlaceholderText = "filter operations..."
	minHeaderContentWidth = 111
)

var badgeColor = map[OperationType]lipgloss.TerminalColor{
	TypeQuery:        tui.ColorPrimary,
	TypeMutation:     tui.ColorWarn,
	TypeSubscription: tui.ColorSuccess,
}

var typeRank = map[OperationType]int{
	TypeQuery:        0,
	TypeMutation:     1,
	TypeSubscription: 2,
}

type ExplorerData struct {
	Operations      []UnifiedOperation
	InputTypes      map[string]graphql.InputType
	EnumTypes       map[string]graphql.EnumType
	ObjectTypes     map[string]graphql.ObjectType
	UnionTypes      map[string]graphql.UnionType
	InterfaceTypes  map[string]graphql.InterfaceType
	APIInfos        map[string]yamlparser.APIInfo
	SchemaFilePaths map[string]string // endpoint URL → .hk.yaml file path
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

type spinnerTickMsg struct{}

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

	inputTypes      map[string]graphql.InputType
	enumTypes       map[string]graphql.EnumType
	objectTypes     map[string]graphql.ObjectType
	unionTypes      map[string]graphql.UnionType
	interfaceTypes  map[string]graphql.InterfaceType
	apiInfos        map[string]yamlparser.APIInfo
	schemaFilePaths map[string]string

	endpoints       []string
	activeEndpoints map[string]bool
	endpointCursor  int
	badgeCache      string

	detailPanel         *tui.Panel
	variablePanel       *tui.Panel
	detailForm          *DetailForm
	detailFormKey       string
	formCache           map[string]*DetailForm
	responseCache       map[string]*cachedResponse
	queryPanel          *tui.Panel
	responsePanel       *tui.Panel
	responseBody        string
	responseColoredBody string
	responseStatusCode  int
	responseDuration    string
	responseSearch      tui.PanelSearch
	executing           bool
	spinnerFrame        int
	focus               tui.FocusRing
	pendingG            bool
	helpBarH            int
	refreshFn           RefreshFunc
	refreshing          bool
	notification        tui.NotificationCenter
	actionRow           tui.ActionRow
	initCmd             tea.Cmd
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
		detailPanel:    dp,
		variablePanel:  vp,
		formCache:      make(map[string]*DetailForm),
		responseCache:  make(map[string]*cachedResponse),
		queryPanel:     qp,
		responsePanel:  rp,
		responseSearch: tui.NewPanelSearch(),
		focus:          tui.NewFocusRing([]*tui.Panel{dp, qp, vp, rp}),
		helpBarH:       tui.HelpBarHeight,
		notification:   tui.NewNotificationCenter(),
		actionRow:      tui.NewActionRow(),
	}
	m.focus.SetTyping(true)
	m.syncSearchFocus()
	m.updateActionRow()
	return m
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
		m.handleQueryExecuted(&msg)
		return m, nil
	case queryErrorMsg:
		m.executing = false
		m.clearResponse()
		m.updateActionRow()
		cmd := m.enqueueNotification(tui.NotificationError, msg.err.Error())
		return m, cmd
	case spinnerTickMsg:
		if !m.executing {
			return m, nil
		}
		m.spinnerFrame++
		m.responsePanel.SetContent(m.spinnerContent(), "")
		return m, spinnerTick()
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
