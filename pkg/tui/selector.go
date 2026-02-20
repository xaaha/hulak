package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// SelectorModel is a generic filterable list selector.
// Provide items and a prompt, get back the user's selection.
type SelectorModel struct {
	FilterableList
	Selected  string
	Cancelled bool
}

func NewSelector(items []string, prompt string) SelectorModel {
	var placeholder string
	if len(items) > 0 {
		placeholder = items[0]
	}

	return SelectorModel{
		FilterableList: NewFilterableList(items, prompt, placeholder, false),
	}
}

func (m SelectorModel) Items() int {
	return len(m.items)
}

func (m SelectorModel) Init() tea.Cmd {
	return m.TextInput.Init()
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}
	return m, m.UpdateInput(msg)
}

func (m SelectorModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyQuit:
		m.Cancelled = true
		return m, tea.Quit

	case KeyCancel:
		if m.HasFilterValue() {
			m.ClearFilter()
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
		return m, nil

	case KeyDown, KeyCtrlN:
		m.Cursor = MoveCursorDown(m.Cursor, len(m.Filtered)-1)
		return m, nil
	}

	return m, m.UpdateInput(msg)
}

func (m SelectorModel) View() string {
	title := m.TextInput.ViewTitle()
	list := m.RenderItems()
	help := HelpStyle.Render("enter: select | esc: cancel | arrows: navigate")

	content := title + "\n\n" + list + "\n" + help
	return "\n" + BoxStyle.Render(content) + "\n"
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
