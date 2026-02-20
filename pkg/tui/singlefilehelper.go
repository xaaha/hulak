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
	title        string
	prompt       string
	items        []string
	lowerItems   []string
	filtered     []string
	cursor       int
	textInput    TextInput
	selected     string
	requireInput bool
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

	lowerItems := make([]string, len(items))
	for i, item := range items {
		lowerItems[i] = strings.ToLower(item)
	}

	filtered := items
	if requireInput {
		filtered = []string{}
	}

	return helperPane{
		title:      title,
		prompt:     prompt,
		items:      items,
		lowerItems: lowerItems,
		filtered:   filtered,
		textInput: NewFilterInput(
			TextInputOpts{Prompt: prompt, Placeholder: placeholder, MinWidth: 20},
		),
		requireInput: requireInput,
	}
}

func (p *helperPane) applyFilter() {
	userInput := p.textInput.Model.Value()
	if userInput == "" {
		if p.requireInput {
			p.filtered = []string{}
		} else {
			p.filtered = p.items
		}
	} else {
		p.filtered = make([]string, 0, len(p.items))
		lower := strings.ToLower(userInput)
		for i, item := range p.lowerItems {
			if strings.Contains(item, lower) {
				p.filtered = append(p.filtered, p.items[i])
			}
		}
	}
	p.cursor = ClampCursor(p.cursor, len(p.filtered)-1)
}

func (p *helperPane) clearFilter() {
	p.textInput.Model.Reset()
	p.applyFilter()
}

func (p *helperPane) hasFilterValue() bool {
	return p.textInput.Model.Value() != ""
}

func (p *helperPane) selectCurrent() bool {
	if len(p.filtered) == 0 || p.cursor >= len(p.filtered) {
		return false
	}
	p.selected = p.filtered[p.cursor]
	return true
}

func (p *helperPane) updateInput(msg tea.Msg) tea.Cmd {
	prev := p.textInput.Model.Value()
	updated, cmd := p.textInput.Update(msg)
	p.textInput = updated
	if p.textInput.Model.Value() != prev {
		p.applyFilter()
	}
	return cmd
}

func (p helperPane) renderSection(isFocused bool, isLocked bool, lockedValue string) string {
	title := TitleStyle.Render(p.title)
	inputLine := p.textInput.ViewTitle()
	if isFocused {
		inputLine = BorderStyle.Padding(0, 1).
			BorderForeground(ColorPrimary).
			Render(p.textInput.Model.View())
	}
	if isLocked {
		inputLine = BorderStyle.Padding(0, 1).Render(p.prompt + lockedValue)
	}

	list := p.renderList(isLocked, lockedValue)
	return title + "\n" + inputLine + "\n" + list
}

func (p helperPane) renderList(isLocked bool, lockedValue string) string {
	if isLocked {
		return SubtitleStyle.Render(utils.ChevronRight + KeySpace + lockedValue)
	}

	if len(p.filtered) == 0 {
		if p.requireInput && p.textInput.Model.Value() == "" {
			return ""
		}
		return HelpStyle.Render("   (no matches)")
	}

	lines := make([]string, 0, len(p.filtered))
	padding := strings.Repeat(KeySpace, 3)
	for i, item := range p.filtered {
		if i == p.cursor {
			lines = append(lines, SubtitleStyle.Render(utils.ChevronRight+KeySpace+item))
		} else {
			lines = append(lines, padding+item)
		}
	}

	return strings.Join(lines, "\n")
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
		envLocked:    envLocked,
		selectedEnv:  initialEnv,
		selectedFile: "",
	}

	if envLocked {
		m.setFocus(focusFile)
	} else {
		m.setFocus(focusEnv)
	}

	return m
}

func (m singleFileHelperModel) Init() tea.Cmd {
	return textBlinkBatch(m.envPane.textInput.Init(), m.filePane.textInput.Init())
}

func textBlinkBatch(cmds ...tea.Cmd) tea.Cmd {
	return tea.Batch(cmds...)
}

func (m singleFileHelperModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}

	if m.focus == focusEnv && !m.envLocked {
		return m, m.envPane.updateInput(msg)
	}

	return m, m.filePane.updateInput(msg)
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
			if !m.envPane.selectCurrent() {
				return m, nil
			}
			m.selectedEnv = m.envPane.selected
			m.setFocus(focusFile)
			return m, nil
		}

		if !m.hasEnvSelection() {
			m.setFocus(focusEnv)
			return m, nil
		}

		if !m.filePane.selectCurrent() {
			return m, nil
		}

		m.selectedFile = m.filePane.selected
		return m, tea.Quit
	case KeyCancel:
		return m.handleCancel()
	case KeyUp, KeyCtrlP:
		m.moveUp()
		return m, nil
	case KeyDown, KeyCtrlN:
		m.moveDown()
		return m, nil
	}

	if m.focus == focusEnv && !m.envLocked {
		return m, m.envPane.updateInput(msg)
	}

	return m, m.filePane.updateInput(msg)
}

func (m singleFileHelperModel) handleCancel() (tea.Model, tea.Cmd) {
	if m.focus == focusEnv {
		if m.envPane.hasFilterValue() && !m.envLocked {
			m.envPane.clearFilter()
			return m, nil
		}
		m.cancelled = true
		return m, tea.Quit
	}

	if m.filePane.hasFilterValue() {
		m.filePane.clearFilter()
		return m, nil
	}

	if m.envLocked {
		m.cancelled = true
		return m, tea.Quit
	}

	m.setFocus(focusEnv)
	return m, nil
}

func (m *singleFileHelperModel) moveUp() {
	if m.focus == focusEnv {
		m.envPane.cursor = MoveCursorUp(m.envPane.cursor)
		return
	}
	m.filePane.cursor = MoveCursorUp(m.filePane.cursor)
}

func (m *singleFileHelperModel) moveDown() {
	if m.focus == focusEnv {
		m.envPane.cursor = MoveCursorDown(m.envPane.cursor, len(m.envPane.filtered)-1)
		return
	}
	m.filePane.cursor = MoveCursorDown(m.filePane.cursor, len(m.filePane.filtered)-1)
}

func (m *singleFileHelperModel) toggleFocus() {
	if m.focus == focusEnv {
		m.setFocus(focusFile)
		return
	}
	if m.envLocked {
		m.setFocus(focusFile)
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

func (m singleFileHelperModel) hasEnvSelection() bool {
	return m.selectedEnv != ""
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
