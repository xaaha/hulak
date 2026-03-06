package gqlexplorer

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

type formItemKind int

const (
	formItemToggle formItemKind = iota
	formItemTextInput
	formItemDropdown
)

type formItem struct {
	kind     formItemKind
	name     string
	typeHint string
	required bool
	isField  bool // true for return type fields, false for arguments

	toggle   tui.Toggle
	input    tui.TextInput
	dropdown tui.Dropdown
}

func (f *formItem) Focus() {
	switch f.kind {
	case formItemToggle:
		f.toggle.Focus()
	case formItemTextInput:
		f.input.Model.Focus()
	case formItemDropdown:
		f.dropdown.Focus()
	}
}

func (f *formItem) Blur() {
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
	switch f.kind {
	case formItemToggle:
		return f.toggle.Focused()
	case formItemTextInput:
		return f.input.Model.Focused()
	case formItemDropdown:
		return f.dropdown.Focused()
	}
	return false
}

func (f *formItem) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch f.kind {
	case formItemToggle:
		f.toggle, _ = f.toggle.Update(msg)
	case formItemTextInput:
		f.input.Update(msg)
	case formItemDropdown:
		f.dropdown, _ = f.dropdown.Update(msg)
	}
	return nil
}

func (f *formItem) View() string {
	hint := tui.HelpStyle.Render(f.typeHint)
	border := tui.HelpStyle.Render("│")
	switch f.kind {
	case formItemToggle:
		return f.toggle.View() + tui.KeySpace + hint
	case formItemTextInput:
		label := f.name + tui.KeySpace + hint
		if f.required {
			label += tui.KeySpace + tui.HelpStyle.Render("*")
		}
		return label + tui.KeySpace + border + tui.KeySpace + f.input.Model.View() + tui.KeySpace + border
	case formItemDropdown:
		return f.name + tui.KeySpace + hint + tui.KeySpace + f.dropdown.View()
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

func newArgFormItem(
	arg graphql.Argument,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
) formItem {
	required := strings.HasSuffix(arg.Type, "!")
	base := ExtractBaseType(arg.Type)

	if base == "Boolean" {
		return formItem{
			kind:     formItemToggle,
			name:     arg.Name,
			typeHint: arg.Type,
			required: required,
			toggle:   tui.NewToggle(arg.Name, false),
		}
	}

	if et, ok := resolveEnumType(enumTypes, endpoint, base); ok {
		options := make([]string, len(et.Values))
		for i, v := range et.Values {
			options[i] = v.Name
		}
		return formItem{
			kind:     formItemDropdown,
			name:     arg.Name,
			typeHint: arg.Type,
			required: required,
			dropdown: tui.NewDropdown(arg.Name, options, 0),
		}
	}

	placeholder := fmt.Sprintf("%s value", base)
	ti := tui.NewFilterInput(tui.TextInputOpts{
		Prompt:      "",
		Placeholder: placeholder,
		MinWidth:    max(len(placeholder), 15),
	})
	ti.Model.Blur()
	return formItem{
		kind:     formItemTextInput,
		name:     arg.Name,
		typeHint: arg.Type,
		required: required,
		input:    ti,
	}
}

func newFieldFormItem(field graphql.ObjectField, selected bool) formItem {
	return formItem{
		kind:     formItemToggle,
		name:     field.Name,
		typeHint: field.Type,
		isField:  true,
		toggle:   tui.NewToggle(field.Name, selected),
	}
}

func resolveEnumType(
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	baseType string,
) (graphql.EnumType, bool) {
	if et, ok := enumTypes[ScopedTypeKey(endpoint, baseType)]; ok {
		return et, true
	}
	et, ok := enumTypes[baseType]
	return et, ok
}

// DetailForm holds the interactive form items for the detail panel.
// Items are ordered: arguments first, then return-type field toggles.
type DetailForm struct {
	items    []formItem
	cursor   int
	argCount int // number of leading argument items
}

// buildDetailForm creates a DetailForm for the given operation.
// Arguments come first, then return-type field toggles (all initially selected).
func buildDetailForm(
	op *UnifiedOperation,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
) *DetailForm {
	var items []formItem

	for _, arg := range op.Arguments {
		items = append(items, newArgFormItem(arg, enumTypes, op.Endpoint))
	}
	argCount := len(items)

	if op.ReturnType != "" {
		base := ExtractBaseType(op.ReturnType)
		if ot, ok := resolveObjectType(objectTypes, op.Endpoint, base); ok {
			for _, f := range ot.Fields {
				items = append(items, newFieldFormItem(f, true))
			}
		}
	}

	if len(items) == 0 {
		return nil
	}

	return &DetailForm{
		items:    items,
		cursor:   0,
		argCount: argCount,
	}
}

// Len returns the total number of form items.
func (df *DetailForm) Len() int {
	return len(df.items)
}

// FocusCurrent focuses the item at the current cursor position
// and blurs all others.
func (df *DetailForm) FocusCurrent() {
	for i := range df.items {
		if i == df.cursor {
			df.items[i].Focus()
		} else {
			df.items[i].Blur()
		}
	}
}

// BlurAll removes focus from every item.
func (df *DetailForm) BlurAll() {
	for i := range df.items {
		df.items[i].Blur()
	}
}

// CursorUp moves the inner cursor up, clamping at 0.
func (df *DetailForm) CursorUp() {
	df.cursor = tui.MoveCursorUp(df.cursor)
	df.FocusCurrent()
}

// CursorDown moves the inner cursor down, clamping at the last item.
func (df *DetailForm) CursorDown() {
	df.cursor = tui.MoveCursorDown(df.cursor, len(df.items)-1)
	df.FocusCurrent()
}

// HandleKey routes a key message to the currently focused item.
func (df *DetailForm) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if df.cursor >= 0 && df.cursor < len(df.items) {
		return df.items[df.cursor].HandleKey(msg)
	}
	return nil
}

// ConsumesTextInput returns true if the focused item is a text input
// or expanded dropdown that should capture arrow keys and other chars.
func (df *DetailForm) ConsumesTextInput() bool {
	if df.cursor >= 0 && df.cursor < len(df.items) {
		return df.items[df.cursor].ConsumesTextInput()
	}
	return false
}

// View renders all form items in a single flat list.
func (df *DetailForm) View(op *UnifiedOperation) string {
	var lines []string

	header := tui.SubtitleStyle.Render("›" + op.Name)
	if op.ReturnType != "" {
		header += tui.HelpStyle.Render(": " + op.ReturnType)
	}
	lines = append(lines, header, "")

	for i := range df.items {
		prefix := strings.Repeat(tui.KeySpace, 4)
		if i == df.cursor {
			prefix = strings.Repeat(tui.KeySpace, 2) + "› "
		}
		lines = append(lines, prefix+df.items[i].View())
	}

	return strings.Join(lines, "\n")
}
