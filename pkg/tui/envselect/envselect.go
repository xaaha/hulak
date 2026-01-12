package envselect

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

type item string

func (i item) FilterValue() string { return string(i) }

type delegate struct{}

func (d delegate) Height() int                             { return 1 }
func (d delegate) Spacing() int                            { return 0 }
func (d delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d delegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	s := li.FilterValue()
	if index == m.Index() {
		_, _ = fmt.Fprint(w, tui.SubtitleStyle.Render("> "+s))
	} else {
		_, _ = fmt.Fprint(w, "  "+s)
	}
}

type Model struct {
	list      list.Model
	Selected  string
	Cancelled bool
}

func NewModel() Model {
	items := []list.Item{item("global")}
	if files, err := utils.GetEnvFiles(); err == nil {
		for _, f := range files {
			name := strings.TrimSuffix(f, ".env")
			if name != "global" {
				items = append(items, item(name))
			}
		}
	}

	l := list.New(items, delegate{}, 30, min(len(items)+2, 10))
	l.Title = "Select Environment"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.SetShowPagination(len(items) > 8)
	l.SetShowHelp(false)
	l.Styles.Title = tui.TitleStyle
	l.Styles.FilterPrompt = tui.FilterStyle
	l.Styles.FilterCursor = tui.FilterCursor

	return Model{list: l}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if quit, cancelled := tui.HandleQuitCancelWithFilter(msg, m.list.SettingFilter()); quit {
			m.Cancelled = cancelled
			return m, tea.Quit
		}
		if tui.IsConfirmKey(msg) && !m.list.SettingFilter() {
			m.Selected = string(m.list.SelectedItem().(item))
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	help := tui.HelpStyle.
		Render("enter: select • esc: cancel • ctrl+c: quit • /: filter")
	return "\n" + m.list.View() + "\n" + help
}

func RunEnvSelector() (string, error) {
	final, err := tea.NewProgram(NewModel()).Run()
	if err != nil {
		return "", err
	}
	m := final.(Model)
	if m.Cancelled {
		return "", nil
	}
	return m.Selected, nil
}
