package envselect

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

type item struct {
	title       string
	description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// Model represents the environment selector TUI state
type Model struct {
	list      list.Model
	Selected  string
	Cancelled bool
}

// getEnvItems fetches env files dynamically from the env/ directory
func getEnvItems() []list.Item {
	items := []list.Item{
		item{title: "global", description: "Default environment"},
	}

	files, err := utils.GetEnvFiles()
	if err != nil {
		return items
	}

	for _, file := range files {
		name := strings.TrimSuffix(file, ".env")
		if name == "global" {
			continue // Already added as default
		}
		items = append(items, item{title: name, description: ""})
	}

	return items
}

// NewModel creates a new environment selector model
func NewModel() Model {
	items := getEnvItems()

	l := list.New(items, list.NewDefaultDelegate(), 40, 15)
	l.Title = "Select Environment"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.SetShowPagination(true)
	l.SetShowHelp(true)

	return Model{list: l}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle quit/cancel using shared helper
		if quit, cancelled := tui.HandleQuitCancelWithFilter(msg, m.list.SettingFilter()); quit {
			m.Cancelled = cancelled
			return m, tea.Quit
		}

		// Handle selection
		if tui.IsConfirmKey(msg) && !m.list.SettingFilter() {
			if itm, ok := m.list.SelectedItem().(item); ok {
				m.Selected = itm.title
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder()).
		Padding(1).
		Render(m.list.View() + "\n" + tui.StandardHelpText())

	return lipgloss.Place(
		40, 20,
		lipgloss.Left,
		lipgloss.Center,
		box,
	)
}

// RunEnvSelector launches the TUI and returns the selected environment name.
// Returns empty string if cancelled or error occurred.
func RunEnvSelector() (string, error) {
	program := tea.NewProgram(NewModel())
	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("env selector error: %w", err)
	}

	model, ok := finalModel.(Model)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}

	if model.Cancelled {
		return "", nil
	}

	return model.Selected, nil
}
