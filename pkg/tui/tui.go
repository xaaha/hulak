package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	title       string
	description string
}

func (i item) Title() string       { return string(i.title) }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return string(i.title) }

type model struct {
	list     list.Model
	selected string
}

func initialModel() model {
	items := []list.Item{
		item{title: "Global", description: "Default"},
		item{title: "Dev", description: ""},
		item{title: "Staging", description: ""},
		item{title: "Prod", description: ""},
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 15)
	l.Title = "Select Environment"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.SetShowPagination(true)
	l.SetShowHelp(true)

	return model{list: l}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selected = string(i.title)
			}
			return m, tea.Quit

		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder()).
		Padding(1).
		Render(m.list.View())

	return lipgloss.Place(
		40, 18,
		lipgloss.Left,
		lipgloss.Center,
		box,
	)
}

// func main() {
// 	p := tea.NewProgram(initialModel())
// 	m, err := p.Run()
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
//
// 	if m := m.(model); m.selected != "" {
// 		fmt.Println("Selected:", m.selected)
// 	}
// }
