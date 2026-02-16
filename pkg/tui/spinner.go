package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner wraps a spinner.Model with a message label.
// Embed it in other Bubble Tea models for loading states.
// Access the inner Model directly for custom spinner styles.
type Spinner struct {
	Model   spinner.Model
	Message string
}

// NewSpinner creates a Spinner with the Dot style and primary color.
func NewSpinner(message string) Spinner {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorPrimary)),
	)
	return Spinner{Model: s, Message: message}
}

// Init returns the tick command to start the spinner animation.
func (s Spinner) Init() tea.Cmd {
	return s.Model.Tick
}

// Update forwards messages to the inner spinner model.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}

// View renders the spinner animation followed by the message.
func (s Spinner) View() string {
	return fmt.Sprintf("%s %s", s.Model.View(), s.Message)
}

// TaskDoneMsg signals that a background task completed.
type TaskDoneMsg struct {
	Result any
	Err    error
}

type spinnerTaskModel struct {
	spinner Spinner
	result  any
	err     error
	task    func() (any, error)
}

func (m spinnerTaskModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), func() tea.Msg {
		result, err := m.task()
		return TaskDoneMsg{Result: result, Err: err}
	})
}

func (m spinnerTaskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TaskDoneMsg:
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
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerTaskModel) View() string {
	return fmt.Sprintf("\n  %s\n", m.spinner.View())
}

// RunWithSpinner displays a spinner while task executes in the background.
func RunWithSpinner(message string, task func() (any, error)) (any, error) {
	model := spinnerTaskModel{
		spinner: NewSpinner(message),
		task:    task,
	}
	p := tea.NewProgram(model)
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	result := m.(spinnerTaskModel)
	return result.result, result.err
}
