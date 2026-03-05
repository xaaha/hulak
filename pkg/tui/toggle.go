package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Toggle is a reusable checkbox/toggle component for Bubble Tea TUIs.
// It renders as [x] label (on) or [ ] label (off) and responds to
// Space/Enter to flip the value. Only responds to keys when focused.
//
// Not a full tea.Model — embed in a parent model and call Update/View
// explicitly, similar to Panel and TextInput.
type Toggle struct {
	Label   string
	Value   bool
	focused bool
}

// NewToggle creates a Toggle with the given label and initial value.
func NewToggle(label string, initial bool) Toggle {
	return Toggle{Label: label, Value: initial}
}

// Init returns nil; Toggle has no startup commands.
func (t Toggle) Init() tea.Cmd {
	return nil
}

// Update handles key messages. Space and Enter toggle the value when
// focused. All other messages are ignored. Returns the updated Toggle
// and any command (always nil for Toggle).
func (t Toggle) Update(msg tea.Msg) (Toggle, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case KeySpace, KeyEnter:
			t.Value = !t.Value
		}
	}

	return t, nil
}

// View renders the toggle as [x] label or [ ] label with focus-aware
// coloring. The checkbox uses ColorPrimary when focused and on,
// ColorMuted otherwise.
func (t Toggle) View() string {
	check := " "
	checkColor := ColorMuted
	if t.Value {
		check = "x"
		if t.focused {
			checkColor = ColorPrimary
		}
	}

	bracket := lipgloss.NewStyle().Foreground(checkColor)
	label := t.Label
	if t.focused {
		label = lipgloss.NewStyle().Foreground(ColorPrimary).Render(label)
	}

	return bracket.Render("["+check+"]") + " " + label
}

// Focus marks the toggle as focused.
func (t *Toggle) Focus() {
	t.focused = true
}

// Blur removes focus from the toggle.
func (t *Toggle) Blur() {
	t.focused = false
}

// Focused reports whether the toggle currently has focus.
func (t Toggle) Focused() bool {
	return t.focused
}
