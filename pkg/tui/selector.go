package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/utils"
)

const (
	selectorViewportDefaultW = 40
	selectorHorizontalPad    = 8
	selectorInputFrameW      = 4
	selectorViewportMaxH     = 3 // fits 3 visible items; keeps the picker compact so it never dominates the terminal
	selectorFrameOverhead    = 8
	defaultHelpMessage       = "enter: select | esc: cancel | arrows: navigate"
)

// SelectorModel is the shared single-list picker engine for simple selection flows.
type SelectorModel struct {
	FilterableList
	Selected    string
	Cancelled   bool
	viewport    viewport.Model
	vpReady     bool
	width       int
	height      int
	helpMessage string
}

func NewSelector(items []string, prompt string, customHelp ...string) SelectorModel {
	var placeholder string
	if len(items) > 0 {
		placeholder = compactPlaceholder(items[0], selectorViewportDefaultW-selectorInputFrameW)
	}

	help := defaultHelpMessage
	if len(customHelp) > 0 && strings.TrimSpace(customHelp[0]) != "" {
		help = customHelp[0]
	}
	m := SelectorModel{
		FilterableList: NewFilterableList(items, prompt, placeholder, false),
		helpMessage:    help,
	}
	m.resizeViewport()
	m.syncViewport()
	return m
}

func compactPlaceholder(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}

	if maxLen <= len(utils.Ellipsis) {
		return strings.Repeat(".", maxLen)
	}

	tailLen := maxLen - len(utils.Ellipsis)
	return utils.Ellipsis + string(runes[len(runes)-tailLen:])
}

func (m *SelectorModel) Init() tea.Cmd {
	return m.TextInput.Init()
}

func (m *SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}

	prev := m.TextInput.Model.Value()
	cmdInput := m.UpdateInput(msg)
	var cmdVP tea.Cmd
	if m.vpReady {
		m.viewport, cmdVP = m.viewport.Update(msg)
	}
	// check if the list has changed, if so then re-render
	if m.TextInput.Model.Value() != prev {
		m.syncViewport()
	}
	return m, tea.Batch(cmdInput, cmdVP)
}

func (m *SelectorModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyQuit:
		m.Cancelled = true
		return m, tea.Quit

	case KeyCancel:
		if m.HasFilterValue() {
			m.ClearFilter()
			m.syncViewport()
			return m, nil
		}
		m.Cancelled = true
		return m, tea.Quit

	case KeyEnter:
		val, ok := m.SelectCurrent()
		if !ok {
			return m, nil
		}
		m.Selected = val
		return m, tea.Quit

	case KeyUp, KeyCtrlP:
		m.Cursor = MoveCursorUp(m.Cursor)
		m.syncViewport()
		return m, nil

	case KeyDown, KeyCtrlN:
		m.Cursor = MoveCursorDown(m.Cursor, len(m.Filtered)-1)
		m.syncViewport()
		return m, nil
	}

	cmd := m.UpdateInput(msg)
	m.syncViewport()
	return m, cmd
}

func (m *SelectorModel) View() string {
	title := m.TextInput.ViewTitle()
	if m.width > 0 {
		title = lipgloss.NewStyle().MaxWidth(max(m.width-selectorHorizontalPad, 1)).Render(title)
	}
	list := ""
	if m.vpReady {
		vp := m.viewport
		content, cursorLine := m.RenderItemsWidth(vp.Width)
		SyncViewport(&vp, content, cursorLine, DefaultScrollMargin)
		list = vp.View()
	} else {
		list, _ = m.RenderItems()
	}
	help := HelpBarStyle.Render(m.helpMessage)
	if m.width > 0 {
		help = lipgloss.NewStyle().MaxWidth(max(m.width-selectorHorizontalPad, 1)).Render(help)
	}

	content := title + "\n\n" + list + "\n" + help
	return "\n" + content + "\n"
}

func (m *SelectorModel) resizeViewport() {
	availableW := selectorViewportDefaultW
	if m.width > 0 {
		availableW = max(m.width-selectorHorizontalPad, 1)
		maxInputW := max(min(availableW, selectorViewportDefaultW)-selectorInputFrameW, 1)
		if m.TextInput.Model.Width > maxInputW {
			m.TextInput.Model.Width = maxInputW
		}
	}
	w := min(availableW, selectorViewportDefaultW)

	h := selectorViewportMaxH
	if m.height > 0 {
		h = min(max(m.height-selectorFrameOverhead, 1), selectorViewportMaxH)
	}

	if !m.vpReady {
		m.viewport = viewport.New(w, h)
		m.viewport.MouseWheelEnabled = true
		m.vpReady = true
		return
	}

	m.viewport.Width = w
	m.viewport.Height = h
}

func (m *SelectorModel) syncViewport() {
	if !m.vpReady {
		return
	}
	content, cursorLine := m.RenderItemsWidth(m.viewport.Width)
	SyncViewport(&m.viewport, content, cursorLine, DefaultScrollMargin)
}

/*
RunSelector runs a generic selector TUI with the given items and prompt.
Returns the selected item, or empty string if cancelled.
Returns emptyErr if no items are available.
*/
func RunSelector(
	items []string,
	prompt string,
	emptyErr error,
	helpMessage ...string,
) (string, error) {
	if len(items) == 0 {
		return "", emptyErr
	}
	model := NewSelector(items, prompt, helpMessage...)
	m, err := tea.NewProgram(&model).Run()
	if err != nil {
		return "", err
	}

	result := m.(*SelectorModel)
	if result.Cancelled {
		return "", nil
	}
	return result.Selected, nil
}
