package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/utils"
)

// SelectorModel is a generic filterable list selector.
// Provide items and a prompt, get back the user's selection.
type SelectorModel struct {
	items     []string
	filtered  []string
	cursor    int
	textInput TextInput
	Selected  string
	Cancelled bool
}

// NewSelector creates a SelectorModel with the given items and prompt.
func NewSelector(items []string, prompt string) SelectorModel {
	var placeholder string
	if len(items) > 0 {
		placeholder = items[0]
	}
	return SelectorModel{
		items:    items,
		filtered: items,
		textInput: NewFilterInput(TextInputOpts{
			Prompt:      prompt,
			Placeholder: placeholder,
			MinWidth:    20,
		}),
	}
}

func (m SelectorModel) Items() int {
	return len(m.items)
}

func (m SelectorModel) Init() tea.Cmd {
	return m.textInput.Init()
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m SelectorModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyQuit:
		m.Cancelled = true
		return m, tea.Quit

	case KeyCancel:
		if m.textInput.Model.Value() != "" {
			m.textInput.Model.Reset()
			m.applyFilter()
			return m, nil
		}
		m.Cancelled = true
		return m, tea.Quit

	case KeyEnter:
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			m.Selected = m.filtered[m.cursor]
		}
		return m, tea.Quit

	case KeyUp, KeyCtrlP:
		m.cursor = MoveCursorUp(m.cursor)
		return m, nil

	case KeyDown, KeyCtrlN:
		m.cursor = MoveCursorDown(m.cursor, len(m.filtered)-1)
		return m, nil
	}

	prevValue := m.textInput.Model.Value()
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if m.textInput.Model.Value() != prevValue {
		m.applyFilter()
	}

	return m, cmd
}

func (m *SelectorModel) applyFilter() {
	userInput := m.textInput.Model.Value()
	if userInput == "" {
		m.filtered = m.items
	} else {
		m.filtered = nil
		lower := strings.ToLower(userInput)
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item), lower) {
				m.filtered = append(m.filtered, item)
			}
		}
	}
	m.cursor = ClampCursor(m.cursor, len(m.filtered)-1)
}

func (m SelectorModel) View() string {
	title := m.textInput.ViewTitle()
	list := m.renderList()
	help := HelpStyle.Render("enter: select | esc: cancel | arrows: navigate")

	content := title + "\n\n" + list + "\n" + help
	return "\n" + BoxStyle.Render(content) + "\n"
}

func (m SelectorModel) renderList() string {
	if len(m.filtered) == 0 {
		return HelpStyle.Render("   (no matches)")
	}

	var lines []string
	for i, item := range m.filtered {
		if i == m.cursor {
			lines = append(lines, SubtitleStyle.Render(utils.ChevronRight+KeySpace+item))
		} else {
			lines = append(lines, strings.Repeat(KeySpace, 3)+item)
		}
	}
	return strings.Join(lines, "\n")
}

/*
RunSelector runs a generic selector TUI with the given items and prompt.
Returns the selected item, or empty string if cancelled.
Returns emptyErr if no items are available.
*/
func RunSelector(items []string, prompt string, emptyErr error) (string, error) {
	if len(items) == 0 {
		return "", emptyErr
	}

	model := NewSelector(items, prompt)
	m, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}

	result := m.(SelectorModel)
	if result.Cancelled {
		return "", nil
	}
	return result.Selected, nil
}
