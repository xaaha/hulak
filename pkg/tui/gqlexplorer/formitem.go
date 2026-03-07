package gqlexplorer

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	kind       formItemKind
	name       string
	typeHint   string
	required   bool
	isField    bool // true for return type fields, false for arguments
	depth      int
	expandable bool

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
	var cmd tea.Cmd
	switch f.kind {
	case formItemToggle:
		f.toggle, cmd = f.toggle.Update(msg)
	case formItemTextInput:
		_, cmd = f.input.Update(msg)
	case formItemDropdown:
		f.dropdown, cmd = f.dropdown.Update(msg)
	}
	return cmd
}

func (f *formItem) View() string {
	hint := tui.HelpStyle.Render(f.typeHint)
	switch f.kind {
	case formItemToggle:
		return f.toggle.View() + tui.KeySpace + hint
	case formItemTextInput:
		label := f.name + tui.KeySpace + hint
		if f.required {
			label += tui.KeySpace + tui.HelpStyle.Render("*")
		}
		borderColor := tui.ColorMuted
		if f.input.Model.Focused() {
			borderColor = tui.ColorPrimary
		}
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)
		inputBox := boxStyle.Render(f.input.Model.View())

		connector := tui.HelpStyle.Render("└─")
		continuePad := tui.KeySpace + tui.KeySpace
		var b strings.Builder
		b.WriteString(label)
		for i, line := range strings.Split(inputBox, "\n") {
			b.WriteString("\n")
			if i == 0 {
				b.WriteString(connector + line)
			} else {
				b.WriteString(continuePad + line)
			}
		}
		return b.String()
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
	return newTypedFormItem(arg.Name, arg.Type, enumTypes, endpoint)
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

func newInputFieldFormItem(
	field graphql.InputField,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
) formItem {
	return newTypedFormItem(field.Name, field.Type, enumTypes, endpoint)
}

func newTypedFormItem(
	name, typeStr string,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
) formItem {
	required := strings.HasSuffix(typeStr, "!")
	base := ExtractBaseType(typeStr)

	if base == "Boolean" {
		return formItem{
			kind:     formItemToggle,
			name:     name,
			typeHint: typeStr,
			required: required,
			toggle:   tui.NewToggle(name, false),
		}
	}

	if et, ok := resolveType(enumTypes, endpoint, base); ok {
		options := make([]string, len(et.Values))
		for i, v := range et.Values {
			options[i] = v.Name
		}
		return formItem{
			kind:     formItemDropdown,
			name:     name,
			typeHint: typeStr,
			required: required,
			dropdown: tui.NewDropdown(name, options, 0),
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
		name:     name,
		typeHint: typeStr,
		required: required,
		input:    ti,
	}
}

// DetailForm holds the interactive form items for the detail panel.
// Items are ordered: arguments first, then return-type field toggles.
type DetailForm struct {
	items       []formItem
	cursor      int
	argCount    int // number of leading argument items
	objectTypes map[string]graphql.ObjectType
	endpoint    string
}

func buildDetailForm(
	op *UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
) *DetailForm {
	var items []formItem

	for _, arg := range op.Arguments {
		base := ExtractBaseType(arg.Type)
		if it, ok := resolveType(inputTypes, op.Endpoint, base); ok {
			for _, field := range it.Fields {
				items = append(items, newInputFieldFormItem(field, enumTypes, op.Endpoint))
			}
		} else {
			items = append(items, newArgFormItem(arg, enumTypes, op.Endpoint))
		}
	}
	argCount := len(items)

	if op.ReturnType != "" {
		base := ExtractBaseType(op.ReturnType)
		if ot, ok := resolveType(objectTypes, op.Endpoint, base); ok {
			for _, f := range ot.Fields {
				childBase := ExtractBaseType(f.Type)
				_, isObj := resolveType(objectTypes, op.Endpoint, childBase)
				fi := newFieldFormItem(f, !isObj)
				fi.expandable = isObj
				items = append(items, fi)
			}
		}
	}

	if len(items) == 0 {
		return nil
	}

	return &DetailForm{
		items:       items,
		cursor:      0,
		argCount:    argCount,
		objectTypes: objectTypes,
		endpoint:    op.Endpoint,
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
		item := &df.items[df.cursor]
		cmd := item.HandleKey(msg)
		if msg.String() == tui.KeySpace && item.expandable && item.kind == formItemToggle {
			df.toggleExpand(df.cursor)
		}
		return cmd
	}
	return nil
}

// ConsumesTextInput returns true if the focused item is a text input
// or expanded dropdown that should capture typed characters.
func (df *DetailForm) ConsumesTextInput() bool {
	if df.cursor >= 0 && df.cursor < len(df.items) {
		return df.items[df.cursor].ConsumesTextInput()
	}
	return false
}

func (df *DetailForm) toggleExpand(idx int) {
	item := &df.items[idx]
	if !item.expandable || item.kind != formItemToggle {
		return
	}

	if item.toggle.Value {
		base := ExtractBaseType(item.typeHint)
		ot, ok := resolveType(df.objectTypes, df.endpoint, base)
		if !ok {
			return
		}
		childDepth := item.depth + 1
		children := make([]formItem, 0, len(ot.Fields))
		for _, f := range ot.Fields {
			child := newFieldFormItem(f, false)
			child.depth = childDepth
			childBase := ExtractBaseType(f.Type)
			if childDepth < maxObjectTypeDepth {
				if _, ok := resolveType(df.objectTypes, df.endpoint, childBase); ok {
					child.expandable = true
				}
			}
			children = append(children, child)
		}
		tail := make([]formItem, len(df.items[idx+1:]))
		copy(tail, df.items[idx+1:])
		df.items = append(df.items[:idx+1], children...)
		df.items = append(df.items, tail...)
	} else {
		start := idx + 1
		end := start
		for end < len(df.items) && df.items[end].depth > item.depth {
			end++
		}
		if end > start {
			df.items = append(df.items[:start], df.items[end:]...)
			if df.cursor >= end {
				df.cursor -= (end - start)
			} else if df.cursor > idx {
				df.cursor = idx
			}
		}
	}
}

func (df *DetailForm) hasExpandedDropdown() bool {
	if df.cursor >= 0 && df.cursor < len(df.items) {
		item := &df.items[df.cursor]
		return item.kind == formItemDropdown && item.dropdown.Expanded()
	}
	return false
}

// View renders all form items in a single flat list.
// Returns the rendered string and the line number of the focused item.
func (df *DetailForm) View(op *UnifiedOperation) (string, int) {
	var lines []string

	header := tui.SubtitleStyle.Render("›" + op.Name)
	if op.ReturnType != "" {
		header += tui.HelpStyle.Render(": " + op.ReturnType)
	}
	lines = append(lines, header, "")

	const basePad = 4
	const depthIndent = 2

	cursorLine := 0
	for i := range df.items {
		depth := df.items[i].depth
		pad := basePad + depth*depthIndent
		itemPad := strings.Repeat(tui.KeySpace, pad)
		cursorPad := strings.Repeat(tui.KeySpace, pad-2) + "› "

		prefix := itemPad
		if i == df.cursor {
			prefix = cursorPad
			cursorLine = len(lines)
		}
		view := df.items[i].View()
		for j, line := range strings.Split(view, "\n") {
			if j == 0 {
				lines = append(lines, prefix+line)
			} else {
				lines = append(lines, itemPad+line)
			}
		}
		if i == df.cursor && df.items[i].kind == formItemDropdown && df.items[i].dropdown.Expanded() {
			cursorLine += df.items[i].dropdown.Cursor()
		}
	}

	return strings.Join(lines, "\n"), cursorLine
}
