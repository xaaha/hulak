package gqlexplorer

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

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
	typeHint   string
	required   bool
	isField    bool // true for return type fields, false for arguments
	depth      int
	expandable bool

	// enabled controls whether this argument is included in the generated
	// query string. Only meaningful for argument items (isField == false).
	// Required args default to true; optional args default to false.
	enabled bool
	// argName is the top-level operation argument name this item belongs to.
	// For simple args it equals name. For InputType-expanded fields it is
	// the parent argument name, allowing the query builder to map multiple
	// form items back to a single argument declaration.
	argName string

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
	switch f.kind {
	case formItemToggle:
		return f.toggle.View() + tui.KeySpace + hint
	case formItemTextInput:
		editing := f.input.Model.Focused()
		highlighted := f.selected || editing
		name := f.name
		if highlighted {
			name = lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render(f.name)
		}
		label := name + tui.KeySpace + hint
		if f.required {
			label += tui.KeySpace + tui.HelpStyle.Render(utils.Asterisk)
		}
		if !f.isField {
			label = f.checkboxPrefix() + label
		}
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
		prefix := ""
		if !f.isField {
			prefix = f.checkboxPrefix()
		}
		return prefix + f.name + tui.KeySpace + hint + tui.KeySpace + f.dropdown.View()
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
			enabled:  required,
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
			enabled:  required,
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
		enabled:  required,
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

	// ── Search state (vim-style / search) ──────────────────────
	searching       bool
	searchInput     tui.TextInput
	matchIndices    []int
	matchCursor     int
	preSearchCursor int
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
				fi := newInputFieldFormItem(field, enumTypes, op.Endpoint)
				fi.argName = arg.Name
				items = append(items, fi)
			}
		} else {
			fi := newArgFormItem(arg, enumTypes, op.Endpoint)
			fi.argName = arg.Name
			items = append(items, fi)
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

	si := tui.NewFilterInput(tui.TextInputOpts{
		Prompt:      "",
		Placeholder: "",
		MinWidth:    10,
	})
	si.Model.Blur()

	return &DetailForm{
		items:       items,
		cursor:      0,
		argCount:    argCount,
		objectTypes: objectTypes,
		endpoint:    op.Endpoint,
		searchInput: si,
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

func (df *DetailForm) enabledArgNames() map[string]bool {
	names := make(map[string]bool)
	for i := 0; i < df.argCount; i++ {
		if df.items[i].enabled {
			names[df.items[i].argName] = true
		}
	}
	return names
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

func (df *DetailForm) CursorToTop() {
	df.cursor = 0
	df.FocusCurrent()
}

func (df *DetailForm) CursorToBottom() {
	df.cursor = len(df.items) - 1
	df.FocusCurrent()
}

// ── Search ─────────────────────────────────────────────────

func (df *DetailForm) IsSearching() bool { return df.searching }

func (df *DetailForm) StartSearch() {
	df.searching = true
	df.preSearchCursor = df.cursor
	df.searchInput.Model.SetValue("")
	df.searchInput.Model.Focus()
	df.matchIndices = nil
	df.matchCursor = 0
}

func (df *DetailForm) StopSearch(confirm bool) {
	df.searching = false
	df.searchInput.Model.Blur()
	if !confirm {
		df.cursor = df.preSearchCursor
		df.FocusCurrent()
	}
	df.matchIndices = nil
}

func (df *DetailForm) updateSearchMatches() {
	query := strings.ToLower(df.searchInput.Model.Value())
	df.matchIndices = nil
	if query == "" {
		return
	}
	for i := range df.items {
		if strings.Contains(strings.ToLower(df.items[i].name), query) {
			df.matchIndices = append(df.matchIndices, i)
		}
	}
	if len(df.matchIndices) > 0 {
		df.matchCursor = 0
		df.cursor = df.matchIndices[0]
		df.FocusCurrent()
	}
}

func (df *DetailForm) nextMatch() {
	if len(df.matchIndices) == 0 {
		return
	}
	df.matchCursor = (df.matchCursor + 1) % len(df.matchIndices)
	df.cursor = df.matchIndices[df.matchCursor]
	df.FocusCurrent()
}

func (df *DetailForm) prevMatch() {
	if len(df.matchIndices) == 0 {
		return
	}
	df.matchCursor--
	if df.matchCursor < 0 {
		df.matchCursor = len(df.matchIndices) - 1
	}
	df.cursor = df.matchIndices[df.matchCursor]
	df.FocusCurrent()
}

func (df *DetailForm) searchStatus() string {
	if df.searchInput.Model.Value() == "" {
		return ""
	}
	if len(df.matchIndices) == 0 {
		return "no matches"
	}
	return fmt.Sprintf("%d/%d", df.matchCursor+1, len(df.matchIndices))
}

func (df *DetailForm) SearchFooter() string {
	if !df.searching {
		return ""
	}
	label := lipgloss.NewStyle().Foreground(tui.ColorPrimary).Render("Search(/)")
	input := df.searchInput.Model.View()
	result := label + " " + input
	if status := df.searchStatus(); status != "" {
		result += "  " + tui.HelpStyle.Render(status)
	}
	return result
}

func (df *DetailForm) HandleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case tui.KeyEnter:
		df.StopSearch(true)
		return nil
	case tui.KeyCancel:
		df.StopSearch(false)
		return nil
	case tui.KeyUp, tui.KeyCtrlP:
		df.prevMatch()
		return nil
	case tui.KeyDown, tui.KeyCtrlN:
		df.nextMatch()
		return nil
	}
	_, cmd := df.searchInput.Update(msg)
	df.updateSearchMatches()
	return cmd
}

// HandleKey routes a key message to the currently focused item.
func (df *DetailForm) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if df.cursor < 0 || df.cursor >= len(df.items) {
		return nil
	}
	item := &df.items[df.cursor]
	key := msg.String()

	// ── Argument items: Space toggles the enabled checkbox ──
	if !item.isField && key == tui.KeySpace && !item.ConsumesTextInput() {
		if item.kind == formItemToggle {
			cmd := item.HandleKey(msg)
			item.enabled = item.toggle.Value
			return cmd
		}
		item.enabled = !item.enabled
		return nil
	}

	// ── Argument text inputs: Enter activates/deactivates editing ──
	if !item.isField && item.kind == formItemTextInput && key == tui.KeyEnter {
		if item.input.Model.Focused() {
			item.input.Model.Blur()
		} else {
			item.input.Model.Focus()
		}
		return nil
	}

	// ── Argument toggles: Enter toggles value + enabled ──
	if !item.isField && item.kind == formItemToggle && key == tui.KeyEnter {
		cmd := item.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
		item.enabled = item.toggle.Value
		return cmd
	}

	// ── Argument text inputs: Esc exits editing ──
	if !item.isField && item.kind == formItemTextInput &&
		key == tui.KeyCancel && item.input.Model.Focused() {
		item.input.Model.Blur()
		return nil
	}

	// ── Pass through to widget ──
	cmd := item.HandleKey(msg)

	// ── Field toggles: expand/collapse children on Space ──
	if key == tui.KeySpace && item.expandable && item.kind == formItemToggle {
		df.toggleExpand(df.cursor)
	}

	return cmd
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

	focused := df.items[df.cursor].Focused()
	headerStyle := tui.HelpStyle
	if focused {
		headerStyle = tui.SubtitleStyle
	}
	header := headerStyle.Render(utils.ChevronRight + op.Name)
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

		prefix := itemPad
		if i == df.cursor {
			if focused {
				prefix = strings.Repeat(tui.KeySpace, pad-2) + utils.ChevronRight
			}
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
		if i == df.cursor && df.items[i].kind == formItemDropdown &&
			df.items[i].dropdown.Expanded() {
			cursorLine += df.items[i].dropdown.Cursor()
		}
	}

	return strings.Join(lines, "\n"), cursorLine
}
