package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/utils"
)

// Dropdown is a reusable select component for Bubble Tea TUIs.
// Collapsed it renders as utils.ChevronRightCircled VALUE;
// expanded it shows a cursor-navigable option list with utils.ChevronDownCircled
// Only responds to keys when focused.

// Not a full tea.Model — embed in a parent model and call Update/View
// explicitly, similar to Panel and Toggle.
type Dropdown struct {
	Label    string
	Options  []string
	Selected int
	focused  bool
	expanded bool
	cursor   int
}

// NewDropdown creates a Dropdown with the given label, options, and
// initially selected index. Out-of-range initial values clamp to 0.
func NewDropdown(label string, options []string, initial int) Dropdown {
	if len(options) == 0 {
		initial = 0
	} else if initial < 0 || initial >= len(options) {
		initial = 0
	}
	return Dropdown{
		Label:    label,
		Options:  options,
		Selected: initial,
		cursor:   initial,
	}
}

// Init returns nil; Dropdown has no startup commands.
func (d Dropdown) Init() tea.Cmd {
	return nil
}

// Update processes key messages based on the current expand state.
// Collapsed: Enter/Space expand the list. Expanded: arrow keys move
// the cursor, Enter/Space select, Esc collapses without changing.
func (d Dropdown) Update(msg tea.Msg) (Dropdown, tea.Cmd) {
	if !d.focused {
		return d, nil
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return d, nil
	}
	if d.expanded {
		return d.updateExpanded(keyMsg)
	}
	return d.updateCollapsed(keyMsg)
}

func (d Dropdown) updateCollapsed(msg tea.KeyMsg) (Dropdown, tea.Cmd) {
	switch msg.String() {
	case KeyEnter, KeySpace:
		if len(d.Options) > 0 {
			d.expanded = true
			d.cursor = d.Selected
		}
	}
	return d, nil
}

func (d Dropdown) updateExpanded(msg tea.KeyMsg) (Dropdown, tea.Cmd) {
	switch msg.String() {
	case KeyEnter, KeySpace:
		d.Selected = d.cursor
		d.expanded = false
	case KeyCancel:
		d.expanded = false
	case KeyUp, KeyCtrlP:
		d.cursor = MoveCursorUp(d.cursor)
	case KeyDown, KeyCtrlN:
		d.cursor = MoveCursorDown(d.cursor, len(d.Options)-1)
	}
	return d, nil
}

// View renders the dropdown. Collapsed: indicator + selected value.
// Expanded: full option list with cursor highlight.
func (d Dropdown) View() string {
	if len(d.Options) == 0 {
		return d.styledIndicator() + " (none)"
	}
	if !d.expanded {
		return d.styledIndicator() + " " + d.Options[d.Selected]
	}

	lines := make([]string, 0, len(d.Options))
	for i, opt := range d.Options {
		if i == d.cursor {
			lines = append(lines, SubtitleStyle.Render(utils.ChevronDownCircled+KeySpace+opt))
		} else {
			lines = append(lines, listPadding+opt)
		}
	}
	return strings.Join(lines, "\n")
}

func (d Dropdown) styledIndicator() string {
	color := ColorMuted
	if d.focused {
		color = ColorPrimary
	}
	return lipgloss.NewStyle().Foreground(color).Render(utils.ChevronRightCircled)
}

// Focus marks the dropdown as focused.
func (d *Dropdown) Focus() {
	d.focused = true
}

// Blur removes focus and collapses the dropdown if expanded.
func (d *Dropdown) Blur() {
	d.focused = false
	d.expanded = false
}

// Focused reports whether the dropdown currently has focus.
func (d Dropdown) Focused() bool {
	return d.focused
}

// Expanded reports whether the option list is currently visible.
func (d Dropdown) Expanded() bool {
	return d.expanded
}

// Cursor returns the current cursor index inside the expanded list.
func (d Dropdown) Cursor() int {
	return d.cursor
}

// Value returns the currently selected option string, or empty if
// there are no options.
func (d Dropdown) Value() string {
	if len(d.Options) == 0 {
		return ""
	}
	return d.Options[d.Selected]
}
