package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerModel wraps a spinner.Model with a message label.
type spinnerModel struct {
	Model   spinner.Model
	Message string
}

// newSpinner creates a spinnerModel with the Dot style and primary color.
func newSpinner(message string) spinnerModel {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorPrimary)),
	)
	return spinnerModel{Model: s, Message: message}
}

// Init returns the tick command to start the spinner animation.
func (s *spinnerModel) Init() tea.Cmd {
	return s.Model.Tick
}

// Update forwards messages to the inner spinner model.
func (s *spinnerModel) Update(msg tea.Msg) (*spinnerModel, tea.Cmd) {
	var cmd tea.Cmd
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}

// View renders the spinner animation followed by the message.
func (s *spinnerModel) View() string {
	return fmt.Sprintf("%s %s", s.Model.View(), s.Message)
}

// taskDoneMsg signals that a background task completed.
type taskDoneMsg struct {
	Result any
	Err    error
}

type spinnerTaskModel struct {
	spinner spinnerModel
	result  any
	err     error
	task    func() (any, error)
}

func (m *spinnerTaskModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), func() tea.Msg {
		result, err := m.task()
		return taskDoneMsg{Result: result, Err: err}
	})
}

func (m *spinnerTaskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case taskDoneMsg:
		m.result = msg.Result
		m.err = msg.Err
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == KeyQuit {
			m.err = fmt.Errorf("interrupted")
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	_, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *spinnerTaskModel) View() string {
	return fmt.Sprintf("\n  %s\n", m.spinner.View())
}

// RunWithSpinner displays a spinner while task executes in the background.
func RunWithSpinner(message string, task func() (any, error)) (any, error) {
	model := spinnerTaskModel{
		spinner: newSpinner(message),
		task:    task,
	}
	p := tea.NewProgram(&model)
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	result := m.(*spinnerTaskModel)
	return result.result, result.err
}
