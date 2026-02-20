package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/utils"
)

type helperFocus int

const (
	focusEnv helperFocus = iota
	focusFile
)

type SingleFileSelection struct {
	Env       string
	File      string
	Cancelled bool
}

type helperPane struct {
	filterableList
	title  string
	prompt string
}

func newHelperPane(
	title, prompt string,
	items []string,
	requireInput bool,
) helperPane {
	placeholder := ""
	if len(items) > 0 {
		placeholder = items[0]
	}

	return helperPane{
		filterableList: newFilterableList(items, prompt, placeholder, requireInput),
		title:          title,
		prompt:         prompt,
	}
}

func (p helperPane) renderSection(isFocused bool, isLocked bool, lockedValue string) string {
	title := MutedTitleStyle.Render(p.title)
	inputLine := p.textInput.ViewTitle()
	if isFocused {
		title = TitleStyle.Render(p.title)
	}
	if isFocused {
		inputLine = FocusedInputStyle.Render(p.textInput.Model.View())
	}
	if isLocked {
		inputLine = InputStyle.Render(p.prompt + lockedValue)
	}

	list := p.renderList(isLocked, lockedValue)
	return title + "\n" + inputLine + "\n" + list
}

func (p helperPane) renderList(isLocked bool, lockedValue string) string {
	if isLocked {
		return SubtitleStyle.Render(utils.ChevronRight + KeySpace + lockedValue)
	}
	return p.renderItems()
}

type singleFileHelperModel struct {
	envPane      helperPane
	filePane     helperPane
	focus        helperFocus
	cancelled    bool
	envLocked    bool
	selectedEnv  string
	selectedFile string
}

func newSingleFileHelperModel(
	envItems []string,
	fileItems []string,
	initialEnv string,
	envLocked bool,
) singleFileHelperModel {
	m := singleFileHelperModel{
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

func (m singleFileHelperModel) Init() tea.Cmd {
	return tea.Batch(m.envPane.textInput.Init(), m.filePane.textInput.Init())
}

func (m singleFileHelperModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}

	return m, m.focusedPane().updateInput(msg)
}

func (m *singleFileHelperModel) focusedPane() *helperPane {
	if m.focus == focusEnv {
		return &m.envPane
	}
	return &m.filePane
}

func (m singleFileHelperModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyQuit:
		m.cancelled = true
		return m, tea.Quit
	case KeyTab:
		m.toggleFocus()
		return m, nil
	case KeyEnter:
		if m.focus == focusEnv {
			if m.envLocked {
				m.setFocus(focusFile)
				return m, nil
			}
			val, ok := m.envPane.selectCurrent()
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

		val, ok := m.filePane.selectCurrent()
		if !ok {
			return m, nil
		}

		m.selectedFile = val
		return m, tea.Quit
	case KeyCancel:
		return m.handleCancel()
	case KeyUp, KeyCtrlP:
		p := m.focusedPane()
		p.cursor = MoveCursorUp(p.cursor)
		return m, nil
	case KeyDown, KeyCtrlN:
		p := m.focusedPane()
		p.cursor = MoveCursorDown(p.cursor, len(p.filtered)-1)
		return m, nil
	}

	if m.focus == focusEnv && !m.envLocked {
		return m, m.envPane.updateInput(msg)
	}

	return m, m.filePane.updateInput(msg)
}

func (m singleFileHelperModel) handleCancel() (tea.Model, tea.Cmd) {
	p := m.focusedPane()

	if m.focus == focusEnv {
		if p.hasFilterValue() && !m.envLocked {
			p.clearFilter()
			return m, nil
		}
		m.cancelled = true
		return m, tea.Quit
	}

	if p.hasFilterValue() {
		p.clearFilter()
		return m, nil
	}

	if m.envLocked {
		m.cancelled = true
		return m, tea.Quit
	}

	m.setFocus(focusEnv)
	return m, nil
}

func (m *singleFileHelperModel) toggleFocus() {
	if m.focus == focusEnv {
		m.setFocus(focusFile)
		return
	}
	if m.envLocked {
		return
	}
	m.setFocus(focusEnv)
}

func (m *singleFileHelperModel) setFocus(f helperFocus) {
	m.focus = f
	if m.focus == focusEnv {
		if !m.envLocked {
			m.envPane.textInput.Model.Focus()
		}
		m.filePane.textInput.Model.Blur()
		return
	}
	m.envPane.textInput.Model.Blur()
	m.filePane.textInput.Model.Focus()
}

func (m singleFileHelperModel) View() string {
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
		lockedNote = HelpStyle.Render(
			"Environment is locked by -env flag. Rerun without -env to change it interactively.",
		)
	}

	helpLine := HelpStyle.Render(
		"enter: select | tab: switch env/file | esc: clear/back/cancel | arrows: navigate",
	)

	parts := []string{}
	if lockedNote != "" {
		parts = append(parts, lockedNote)
	}
	parts = append(parts, envSection, "", fileSection, "", helpLine)

	return "\n" + strings.Join(parts, "\n") + "\n"
}

// RunSingleFileHelper runs the combined interactive selector and returns selections.
func RunSingleFileHelper(
	envItems []string,
	fileItems []string,
	initialEnv string,
	envLocked bool,
) (SingleFileSelection, error) {
	model := newSingleFileHelperModel(envItems, fileItems, initialEnv, envLocked)
	out, err := tea.NewProgram(model).Run()
	if err != nil {
		return SingleFileSelection{}, err
	}

	result := out.(singleFileHelperModel)
	return SingleFileSelection{
		Env:       result.selectedEnv,
		File:      result.selectedFile,
		Cancelled: result.cancelled,
	}, nil
}
