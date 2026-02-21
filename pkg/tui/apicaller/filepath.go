package apicaller

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

type helperFocus int

const (
	focusEnv helperFocus = iota
	focusFile
)

type helperPane struct {
	tui.FilterableList
	title       string
	vp          viewport.Model
	vpReady     bool
	vpAllocated int
}

func (p *helperPane) syncViewport() {
	if !p.vpReady {
		return
	}
	content, cursorLine := p.RenderItemsWidth(p.vp.Width)
	p.vp.Height = max(p.vpAllocated, 1)
	tui.SyncViewport(&p.vp, content, cursorLine, tui.DefaultScrollMargin)
}

type SingleFileSelection struct {
	Env       string
	File      string
	Cancelled bool
}

func newHelperPane(
	title, prompt string,
	items []string,
	requireInput bool,
) helperPane {
	return helperPane{
		FilterableList: tui.NewFilterableList(items, prompt, "", requireInput),
		title:          title,
	}
}

func (p helperPane) renderSection(isFocused bool, isLocked bool, lockedValue string) string {
	var title, inputLine string

	switch {
	case isLocked:
		title = tui.MutedTitleStyle.Render(p.title)
		inputLine = tui.InputStyle.Render(p.TextInput.Model.Prompt + lockedValue)
	case isFocused:
		title = tui.TitleStyle.Render(p.title)
		inputLine = tui.FocusedInputStyle.Render(p.TextInput.Model.View())
	default:
		title = tui.MutedTitleStyle.Render(p.title)
		inputLine = p.TextInput.ViewTitle()
	}

	list := p.renderList(isLocked, lockedValue)
	return title + "\n" + inputLine + "\n" + list
}

func (p helperPane) renderList(isLocked bool, lockedValue string) string {
	if isLocked {
		return tui.SubtitleStyle.Render(utils.ChevronRight + tui.KeySpace + lockedValue)
	}
	if p.vpReady {
		return p.vp.View()
	}
	content, _ := p.RenderItemsWidth(0)
	return content
}

const (
	paneOverhead  = 5
	frameOverhead = 4
	maxEnvListH   = 1 // controls the length of the items shown in initial screen, for env, 1 is enough
	maxFileListH  = 5 // same as above, at least 5 so that list does not jitter
)

type filePathModel struct {
	envPane      helperPane
	filePane     helperPane
	focus        helperFocus
	cancelled    bool
	envLocked    bool
	selectedEnv  string
	selectedFile string
	width        int
	height       int
}

func (m *filePathModel) resizeViewports() {
	overhead := frameOverhead + 2*paneOverhead
	if m.envLocked {
		overhead++
	}
	available := max(m.height-overhead, 2)

	var envH, fileH int
	if m.envLocked {
		envH = 1
		fileH = min(max(available-1, 1), maxFileListH)
	} else {
		envH = min(max(available/3, 1), maxEnvListH)
		fileH = min(max(available-envH, 1), maxFileListH)
	}

	initOrResizeVP(&m.envPane, m.width, envH)
	initOrResizeVP(&m.filePane, m.width, fileH)
	m.envPane.syncViewport()
	m.filePane.syncViewport()
}

func initOrResizeVP(p *helperPane, w, h int) {
	p.vpAllocated = h
	if !p.vpReady {
		p.vp = viewport.New(w, h)
		p.vp.MouseWheelEnabled = true
		p.vpReady = true
	} else {
		p.vp.Width = w
		p.vp.Height = h
	}
}

func newFilePathModel(
	envItems []string,
	fileItems []string,
	initialEnv string,
	envLocked bool,
) filePathModel {
	m := filePathModel{
		envPane: newHelperPane(
			"Environment",
			"Select Environment: ",
			envItems,
			true,
		),
		filePane: newHelperPane(
			"Request File",
			"Select File: ",
			fileItems,
			true,
		),
		envLocked:   envLocked,
		selectedEnv: initialEnv,
	}

	if envLocked {
		m.setFocus(focusFile)
	} else {
		m.setFocus(focusEnv)
	}

	return m
}

func (m filePathModel) Init() tea.Cmd {
	return tea.Batch(m.envPane.TextInput.Init(), m.filePane.TextInput.Init())
}

func (m filePathModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewports()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	p := m.focusedPane()
	cmdInput := p.UpdateInput(msg)
	var cmdVP tea.Cmd
	if p.vpReady {
		p.vp, cmdVP = p.vp.Update(msg)
	}
	p.syncViewport()
	return m, tea.Batch(cmdInput, cmdVP)
}

func (m *filePathModel) focusedPane() *helperPane {
	if m.focus == focusEnv {
		return &m.envPane
	}
	return &m.filePane
}

func (m filePathModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		m.cancelled = true
		return m, tea.Quit
	case tui.KeyTab:
		m.toggleFocus()
		return m, nil
	case tui.KeyEnter:
		if m.focus == focusEnv {
			if m.envLocked {
				m.setFocus(focusFile)
				return m, nil
			}
			val, ok := m.envPane.SelectCurrent()
			if !ok {
				return m, nil
			}
			m.selectedEnv = val
			m.setFocus(focusFile)
			return m, nil
		}

		if m.selectedEnv == "" {
			m.setFocus(focusEnv)
			return m, nil
		}

		val, ok := m.filePane.SelectCurrent()
		if !ok {
			return m, nil
		}

		m.selectedFile = val
		return m, tea.Quit
	case tui.KeyCancel:
		return m.handleCancel()
	case tui.KeyUp, tui.KeyCtrlP:
		p := m.focusedPane()
		p.Cursor = tui.MoveCursorUp(p.Cursor)
		p.syncViewport()
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN:
		p := m.focusedPane()
		p.Cursor = tui.MoveCursorDown(p.Cursor, len(p.Filtered)-1)
		p.syncViewport()
		return m, nil
	}

	if m.focus == focusEnv && !m.envLocked {
		cmd := m.envPane.UpdateInput(msg)
		m.envPane.syncViewport()
		return m, cmd
	}

	cmd := m.filePane.UpdateInput(msg)
	m.filePane.syncViewport()
	return m, cmd
}

func (m filePathModel) handleCancel() (tea.Model, tea.Cmd) {
	p := m.focusedPane()

	if m.focus == focusEnv {
		if p.HasFilterValue() && !m.envLocked {
			p.ClearFilter()
			p.syncViewport()
			return m, nil
		}
		m.cancelled = true
		return m, tea.Quit
	}

	if p.HasFilterValue() {
		p.ClearFilter()
		p.syncViewport()
		return m, nil
	}

	if m.envLocked {
		m.cancelled = true
		return m, tea.Quit
	}

	m.setFocus(focusEnv)
	return m, nil
}

func (m *filePathModel) toggleFocus() {
	if m.focus == focusEnv {
		m.setFocus(focusFile)
		return
	}
	if m.envLocked {
		return
	}
	m.setFocus(focusEnv)
}

func (m *filePathModel) setFocus(f helperFocus) {
	m.focus = f
	if m.focus == focusEnv {
		if !m.envLocked {
			m.envPane.TextInput.Model.Focus()
		}
		m.filePane.TextInput.Model.Blur()
		return
	}
	m.envPane.TextInput.Model.Blur()
	m.filePane.TextInput.Model.Focus()
}

func (m filePathModel) View() string {
	envLockedValue := m.selectedEnv
	if envLockedValue == "" {
		envLockedValue = utils.DefaultEnvVal
	}

	envFocus := m.focus == focusEnv && !m.envLocked
	fileFocus := m.focus == focusFile

	envSection := m.envPane.renderSection(envFocus, m.envLocked, envLockedValue)
	fileSection := m.filePane.renderSection(fileFocus, false, "")

	lockedNote := ""
	if m.envLocked {
		lockedNote = tui.HelpStyle.Render(
			"Environment is locked by -env flag. Rerun without -env to change it interactively.",
		)
	}

	helpLine := tui.HelpStyle.Render(
		"enter: select | tab: switch env/file | esc: clear/back/cancel | arrows: navigate",
	)

	parts := []string{}
	if lockedNote != "" {
		parts = append(parts, lockedNote)
	}
	parts = append(parts, envSection, "", fileSection, helpLine)

	return "\n" + strings.Join(parts, "\n") + "\n"
}

func RunFilePath(
	envItems []string,
	fileItems []string,
	initialEnv string,
	envLocked bool,
) (SingleFileSelection, error) {
	model := newFilePathModel(envItems, fileItems, initialEnv, envLocked)
	out, err := tea.NewProgram(model).Run()
	if err != nil {
		return SingleFileSelection{}, err
	}

	result := out.(filePathModel)
	return SingleFileSelection{
		Env:       result.selectedEnv,
		File:      result.selectedFile,
		Cancelled: result.cancelled,
	}, nil
}
