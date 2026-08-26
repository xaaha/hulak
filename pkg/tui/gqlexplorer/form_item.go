package gqlexplorer

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

const fragmentPrefix = utils.Ellipsis + " on "

type formItemKind int

const (
	formItemToggle formItemKind = iota
	formItemTextInput
	formItemDropdown
)

var (
	cbOnFocused  = renderCheckbox(true, true)
	cbOnBlurred  = renderCheckbox(true, false)
	cbOffFocused = renderCheckbox(false, true)
	cbOffBlurred = renderCheckbox(false, false)
)

func renderCheckbox(enabled, focused bool) string {
	t := tui.NewToggle("", enabled)
	if focused {
		t.Focus()
	}
	return t.View()
}

type formItem struct {
	kind       formItemKind
	name       string
	label      string
	typeHint   string
	valueType  string
	required   bool
	isField    bool // true for return type fields, false for arguments
	depth      int
	expandable bool
	listType   string
	listItem   bool
	listGroup  int
	continued  bool

	// listBoundary identifies the nested list-of-input-object element this
	// item belongs to (an argName plus the path prefix up to that element's
	// index), for fields nested inside a list-typed field of an object
	// argument. boundaryGroup is this item's element index within that
	// boundary. Distinct from listItem/listGroup, which only cover
	// top-level list arguments.
	listBoundary  string
	boundaryGroup int

	// enabled controls whether this argument is included in the generated
	// query string. Only meaningful for argument items (isField == false).
	// Required args default to true; optional args default to false.
	enabled bool
	// argName is the top-level operation argument name this item belongs to.
	// For simple args it equals name. For InputType-expanded fields it is
	// the parent argument name, allowing the query builder to map multiple
	// form items back to a single argument declaration.
	argName string
	// path locates this item's value within its top-level argument. It is
	// empty for a plain scalar argument (the value is the argument itself),
	// and otherwise a sequence of object-field keys and list indices that the
	// variables builder walks to place the value inside nested objects/lists.
	path []varPathSeg

	selected bool // cursor is on this item (set by Focus/Blur)

	toggle   tui.Toggle
	input    tui.TextInput
	dropdown tui.Dropdown
}

func (f *formItem) Focus() {
	f.selected = true
	switch f.kind {
	case formItemToggle:
		f.toggle.Focus()
	case formItemTextInput:
		if f.isField {
			f.input.Model.Focus()
		}
	case formItemDropdown:
		f.dropdown.Focus()
	}
}

func (f *formItem) Blur() {
	f.selected = false
	switch f.kind {
	case formItemToggle:
		f.toggle.Blur()
	case formItemTextInput:
		f.input.Model.Blur()
	case formItemDropdown:
		f.dropdown.Blur()
	}
}

func (f *formItem) Focused() bool {
	return f.selected
}

func (f *formItem) checkboxPrefix() string {
	switch {
	case f.enabled && f.selected:
		return cbOnFocused
	case f.enabled:
		return cbOnBlurred
	case f.selected:
		return cbOffFocused
	default:
		return cbOffBlurred
	}
}

func (f *formItem) View() string {
	hint := tui.HelpStyle.Render(f.typeHint)
	labelText := f.name
	if f.label != "" {
		labelText = f.label
	}
	switch f.kind {
	case formItemToggle:
		return f.toggle.View() + tui.KeySpace + hint
	case formItemTextInput:
		editing := f.input.Model.Focused()
		highlighted := f.selected || editing
		boxStyle := tui.InputStyle
		if editing {
			boxStyle = tui.FocusedInputStyle
		}
		inputBox := boxStyle.Render(f.input.Model.View())

		connectorStyle := tui.HelpStyle
		if highlighted {
			connectorStyle = lipgloss.NewStyle().Foreground(tui.ColorPrimary)
		}
		connector := connectorStyle.Render(utils.Connector)
		continuePad := tui.KeySpace + tui.KeySpace
		if f.continued {
			var b strings.Builder
			for i, line := range strings.Split(inputBox, "\n") {
				b.WriteString("\n")
				if i == 0 {
					b.WriteString(connector)
				} else {
					b.WriteString(continuePad)
				}
				b.WriteString(line)
			}
			return strings.TrimPrefix(b.String(), "\n")
		}
		name := labelText
		if highlighted {
			name = lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render(labelText)
		}
		label := name + tui.KeySpace + hint
		if f.required {
			label += tui.KeySpace + tui.HelpStyle.Render(utils.Asterisk)
		}
		if !f.isField {
			label = f.checkboxPrefix() + label
		}
		var b strings.Builder
		b.WriteString(label)
		for i, line := range strings.Split(inputBox, "\n") {
			b.WriteString("\n")
			if i == 0 {
				b.WriteString(connector)
			} else {
				b.WriteString(continuePad)
			}
			b.WriteString(line)
		}
		return b.String()
	case formItemDropdown:
		prefix := ""
		if !f.isField {
			prefix = f.checkboxPrefix()
		}
		return prefix + labelText + tui.KeySpace + hint + tui.KeySpace + f.dropdown.View()
	}
	return ""
}

func (f *formItem) Value() string {
	switch f.kind {
	case formItemToggle:
		if f.toggle.Value {
			return "true"
		}
		return "false"
	case formItemTextInput:
		return f.input.Model.Value()
	case formItemDropdown:
		return f.dropdown.Value()
	}
	return ""
}

func (f *formItem) ConsumesTextInput() bool {
	switch f.kind {
	case formItemTextInput:
		return f.input.Model.Focused()
	case formItemDropdown:
		return f.dropdown.Expanded()
	}
	return false
}
