package gqlexplorer

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/xaaha/hulak/pkg/features/graphql"
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

func newListArgFormItems(
	arg graphql.Argument,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	group int,
	enabled bool,
	nestedLists map[string]nestedListSpec,
) []formItem {
	itemType := ExtractListItemType(arg.Type)
	if it, ok := resolveType(inputTypes, endpoint, ExtractBaseType(itemType)); ok {
		ctx := inputExpandCtx{
			inputTypes:     inputTypes,
			enumTypes:      enumTypes,
			endpoint:       endpoint,
			argName:        arg.Name,
			parentPath:     []varPathSeg{{index: group, isIndex: true}},
			depth:          0,
			parentEnabled:  false,
			inTopLevelList: true,
			listType:       arg.Type,
			listGroup:      group,
			visited:        map[string]bool{ExtractBaseType(itemType): true},
			nestedLists:    nestedLists,
		}
		items := make([]formItem, 0, len(it.Fields))
		for _, field := range it.Fields {
			items = append(items, expandInputField(field, &ctx)...)
		}
		return items
	}

	fi := newTypedFormItem(arg.Name, itemType, enumTypes, endpoint)
	fi.argName = arg.Name
	fi.path = []varPathSeg{{index: group, isIndex: true}}
	fi.listType = arg.Type
	fi.listItem = true
	fi.listGroup = group
	fi.valueType = itemType
	fi.enabled = enabled
	fi.required = strings.HasSuffix(arg.Type, "!")
	fi.typeHint = arg.Type
	if fi.kind == formItemDropdown {
		fi.dropdown = listDropdown(fi.dropdown)
		fi.enabled = enabled
		fi.required = strings.HasSuffix(arg.Type, "!")
		fi.label = fmt.Sprintf("%s[%d]", arg.Name, group)
		return []formItem{fi}
	}
	if group > 0 {
		fi.continued = true
		fi.required = false
	}
	return []formItem{fi}
}

func newArgFormItem(
	arg graphql.Argument,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
) formItem {
	return newTypedFormItem(arg.Name, arg.Type, enumTypes, endpoint)
}

// maxInputFormDepth caps how deep the form expands nested input objects. The
// visited set already terminates cyclic input types; this is a safety net for
// very deep but acyclic schemas so the form stays usable.
const maxInputFormDepth = 8

// inputExpandCtx carries the state threaded through recursive expansion of an
// input-object argument (or list element) into individual form items.
type inputExpandCtx struct {
	inputTypes map[string]graphql.InputType
	enumTypes  map[string]graphql.EnumType
	endpoint   string
	argName    string
	// parentPath is the path from the argument root to the object that owns the
	// field currently being expanded.
	parentPath []varPathSeg
	depth      int
	// parentEnabled reports whether the owning object contributes by default,
	// so an optional parent disables its children (mirroring top-level args).
	parentEnabled bool
	// inTopLevelList marks that this item belongs to a top-level list argument,
	// which drives dynamic continuation rows and whole-list enable/disable.
	inTopLevelList bool
	listType       string
	listGroup      int
	visited        map[string]bool

	// nestedLists accumulates the specs needed to regenerate additional
	// elements of list-of-input-object fields nested inside this argument,
	// keyed by pathKey. It is the same underlying map for the whole
	// expansion of one top-level argument, so a spec registered deep in the
	// recursion is visible to the DetailForm afterward.
	nestedLists map[string]nestedListSpec
	// nestedListBoundary/nestedListGroup identify which nested list element
	// the field currently being expanded belongs to, if any. They are
	// inherited unchanged through intermediate non-list objects so a deeply
	// nested leaf still resolves to the list element that contains it.
	nestedListBoundary string
	nestedListGroup    int
}

// nestedListSpec captures the context present when a list-of-input-object
// field nested inside another argument is first encountered, so additional
// elements can be regenerated on demand as the user fills the previous one
// in (mirroring newListArgFormItems for top-level list arguments).
//
// inTopLevelList/listType/listGroup carry the ambient top-level list context
// (if any) that was active at that point, so a list field nested inside a
// top-level list argument's element keeps contributing to the outer list's
// own group accounting (see syncListArgRows) in addition to this field's own
// nested boundary. Dropping them here would make every regenerated element's
// leaves report listGroup 0 regardless of which outer element they actually
// belong to, corrupting the outer list's grow/shrink scan.
type nestedListSpec struct {
	argName        string
	fieldPath      []varPathSeg
	elementBase    string
	depth          int
	visited        map[string]bool
	inTopLevelList bool
	listType       string
	listGroup      int
}

// expandNestedListGroup builds the form items for one element of a
// list-of-input-object field nested inside another argument, resolving the
// element's input type first. Used by syncNestedListBoundary, which only has
// the spec (not an already-resolved graphql.InputType) on hand.
func expandNestedListGroup(
	boundary string,
	spec *nestedListSpec,
	group int,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	nestedLists map[string]nestedListSpec,
) []formItem {
	it, ok := resolveType(inputTypes, endpoint, spec.elementBase)
	if !ok || len(it.Fields) == 0 {
		return nil
	}
	return expandNestedListElement(it, boundary, spec, group, inputTypes, enumTypes, endpoint, nestedLists)
}

// expandNestedListElement builds the form items for one element of a
// list-of-input-object field, given the element's already-resolved input
// type.
func expandNestedListElement(
	it graphql.InputType,
	boundary string,
	spec *nestedListSpec,
	group int,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	nestedLists map[string]nestedListSpec,
) []formItem {
	child := inputExpandCtx{
		inputTypes:         inputTypes,
		enumTypes:          enumTypes,
		endpoint:           endpoint,
		argName:            spec.argName,
		parentPath:         appendPathSeg(spec.fieldPath, varPathSeg{index: group, isIndex: true}),
		depth:              spec.depth,
		parentEnabled:      false,
		inTopLevelList:     spec.inTopLevelList,
		listType:           spec.listType,
		listGroup:          spec.listGroup,
		visited:            spec.visited,
		nestedLists:        nestedLists,
		nestedListBoundary: boundary,
		nestedListGroup:    group,
	}
	items := make([]formItem, 0, len(it.Fields))
	for _, sub := range it.Fields {
		items = append(items, expandInputField(sub, &child)...)
	}
	return items
}

// expandInputField turns one input field into form items. Scalar, enum and
// Boolean fields become a single leaf item. Fields whose type is an input
// object (or a list of input objects) recurse into their own fields, to
// arbitrary depth, guarded by a visited set (cycles) and a depth cap.
func expandInputField(field graphql.InputField, ctx *inputExpandCtx) []formItem {
	fieldPath := appendPathSeg(ctx.parentPath, varPathSeg{key: field.Name})
	fieldRequired := strings.HasSuffix(field.Type, "!")
	base := ExtractBaseType(field.Type)

	it, isInput := resolveType(ctx.inputTypes, ctx.endpoint, base)
	if isInput && !ctx.visited[base] && ctx.depth < maxInputFormDepth && len(it.Fields) > 0 {
		if IsListType(field.Type) {
			boundary := pathKey(ctx.argName, fieldPath)
			spec := nestedListSpec{
				argName:        ctx.argName,
				fieldPath:      fieldPath,
				elementBase:    base,
				depth:          ctx.depth + 1,
				visited:        cloneVisited(ctx.visited, base),
				inTopLevelList: ctx.inTopLevelList,
				listType:       ctx.listType,
				listGroup:      ctx.listGroup,
			}
			if ctx.nestedLists != nil {
				if _, exists := ctx.nestedLists[boundary]; !exists {
					ctx.nestedLists[boundary] = spec
				}
			}
			return expandNestedListElement(
				it, boundary, &spec, 0, ctx.inputTypes, ctx.enumTypes, ctx.endpoint, ctx.nestedLists,
			)
		}

		child := inputExpandCtx{
			inputTypes:         ctx.inputTypes,
			enumTypes:          ctx.enumTypes,
			endpoint:           ctx.endpoint,
			argName:            ctx.argName,
			parentPath:         fieldPath,
			depth:              ctx.depth + 1,
			parentEnabled:      ctx.parentEnabled && fieldRequired,
			inTopLevelList:     ctx.inTopLevelList,
			listType:           ctx.listType,
			listGroup:          ctx.listGroup,
			visited:            cloneVisited(ctx.visited, base),
			nestedLists:        ctx.nestedLists,
			nestedListBoundary: ctx.nestedListBoundary,
			nestedListGroup:    ctx.nestedListGroup,
		}
		items := make([]formItem, 0, len(it.Fields))
		for _, sub := range it.Fields {
			items = append(items, expandInputField(sub, &child)...)
		}
		return items
	}

	return []formItem{newLeafFormItem(field, fieldPath, fieldRequired, ctx)}
}

func newLeafFormItem(
	field graphql.InputField,
	fieldPath []varPathSeg,
	fieldRequired bool,
	ctx *inputExpandCtx,
) formItem {
	fi := newTypedFormItem(field.Name, field.Type, ctx.enumTypes, ctx.endpoint)
	fi.argName = ctx.argName
	fi.path = fieldPath
	fi.depth = ctx.depth
	fi.required = fieldRequired
	fi.label = labelForPath(ctx.argName, fieldPath)
	if ctx.inTopLevelList {
		fi.listType = ctx.listType
		fi.listItem = true
		fi.listGroup = ctx.listGroup
		fi.enabled = false
	} else {
		fi.enabled = ctx.parentEnabled && fieldRequired
	}
	fi.listBoundary = ctx.nestedListBoundary
	fi.boundaryGroup = ctx.nestedListGroup
	return fi
}

// labelForPath builds a display label for a nested field. Direct fields of a
// top-level input-object argument keep an empty label so only the field name is
// shown (matching the original single-level behavior); deeper or list-nested
// fields get a dotted/bracketed path like arg[0].field.subfield.
func labelForPath(argName string, path []varPathSeg) string {
	if len(path) == 1 && !path[0].isIndex {
		return ""
	}
	return pathKey(argName, path)
}

// pathKey renders an argument name and path as a flat string, used both for
// display labels (via labelForPath) and as the map key identifying a nested
// list boundary.
func pathKey(argName string, path []varPathSeg) string {
	var b strings.Builder
	b.WriteString(argName)
	for _, seg := range path {
		if seg.isIndex {
			b.WriteString("[")
			b.WriteString(strconv.Itoa(seg.index))
			b.WriteString("]")
		} else {
			b.WriteString(".")
			b.WriteString(seg.key)
		}
	}
	return b.String()
}

func appendPathSeg(path []varPathSeg, seg varPathSeg) []varPathSeg {
	out := make([]varPathSeg, len(path)+1)
	copy(out, path)
	out[len(path)] = seg
	return out
}

func cloneVisited(visited map[string]bool, add string) map[string]bool {
	out := make(map[string]bool, len(visited)+1)
	for k := range visited {
		out[k] = true
	}
	out[add] = true
	return out
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

func newFragmentFormItem(typeName string) formItem {
	label := fragmentPrefix + typeName
	return formItem{
		kind:       formItemToggle,
		name:       label,
		typeHint:   typeName,
		isField:    true,
		expandable: true,
		toggle:     tui.NewToggle(label, false),
	}
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
			kind:      formItemToggle,
			name:      name,
			typeHint:  typeStr,
			valueType: typeStr,
			required:  required,
			enabled:   required,
			toggle:    tui.NewToggle(name, false),
		}
	}

	if et, ok := resolveType(enumTypes, endpoint, base); ok {
		options := make([]string, len(et.Values))
		for i, v := range et.Values {
			options[i] = v.Name
		}
		return formItem{
			kind:      formItemDropdown,
			name:      name,
			typeHint:  typeStr,
			valueType: typeStr,
			required:  required,
			enabled:   required,
			dropdown:  tui.NewDropdown(name, options, 0),
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
		kind:      formItemTextInput,
		name:      name,
		typeHint:  typeStr,
		valueType: typeStr,
		required:  required,
		enabled:   required,
		input:     ti,
	}
}

// DetailForm holds the interactive form items for the detail panel.
// Items are ordered: arguments first, then return-type field toggles.
type DetailForm struct {
	items       []formItem
	cursor      int
	argCount    int // number of leading argument items
	inputTypes  map[string]graphql.InputType
	enumTypes   map[string]graphql.EnumType
	objectTypes map[string]graphql.ObjectType
	endpoint    string
	// nestedLists holds the specs needed to regenerate additional elements
	// of list-of-input-object fields nested inside an argument, keyed by
	// pathKey.
	nestedLists map[string]nestedListSpec

	// Search state (vim-style / search)
	search          tui.PanelSearch
	preSearchCursor int
}

func buildDetailForm(
	op *UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
) *DetailForm {
	var items []formItem
	nestedLists := make(map[string]nestedListSpec)

	for _, arg := range op.Arguments {
		base := ExtractBaseType(arg.Type)
		if IsListType(arg.Type) {
			items = append(items, newListArgFormItems(
				arg, inputTypes, enumTypes, op.Endpoint, 0, strings.HasSuffix(arg.Type, "!"), nestedLists,
			)...)
		} else if it, ok := resolveType(inputTypes, op.Endpoint, base); ok {
			// if parent is not required, its children should be optional as well
			argRequired := strings.HasSuffix(arg.Type, "!")
			ctx := inputExpandCtx{
				inputTypes:    inputTypes,
				enumTypes:     enumTypes,
				endpoint:      op.Endpoint,
				argName:       arg.Name,
				depth:         0,
				parentEnabled: argRequired,
				visited:       map[string]bool{base: true},
				nestedLists:   nestedLists,
			}
			for _, field := range it.Fields {
				items = append(items, expandInputField(field, &ctx)...)
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
		if ut, ok := resolveType(unionTypes, op.Endpoint, base); ok {
			for _, pt := range ut.PossibleTypes {
				items = append(items, newFragmentFormItem(pt))
			}
		} else if it, ok := resolveType(interfaceTypes, op.Endpoint, base); ok {
			for _, f := range it.Fields {
				childBase := ExtractBaseType(f.Type)
				_, isObj := resolveType(objectTypes, op.Endpoint, childBase)
				fi := newFieldFormItem(f, !isObj)
				fi.expandable = isObj
				items = append(items, fi)
			}
			for _, pt := range it.PossibleTypes {
				items = append(items, newFragmentFormItem(pt))
			}
		} else if ot, ok := resolveType(objectTypes, op.Endpoint, base); ok {
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
		inputTypes:  inputTypes,
		enumTypes:   enumTypes,
		objectTypes: objectTypes,
		endpoint:    op.Endpoint,
		nestedLists: nestedLists,
		search:      tui.NewPanelSearch(),
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

func (df *DetailForm) argRange(argName string) (int, int, bool) {
	start := -1
	end := -1
	for i := 0; i < df.argCount; i++ {
		if df.items[i].argName != argName {
			if start != -1 {
				break
			}
			continue
		}
		if start == -1 {
			start = i
		}
		end = i + 1
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}

func (df *DetailForm) argDefinition(argName string) (graphql.Argument, bool) {
	start, _, ok := df.argRange(argName)
	if !ok {
		return graphql.Argument{}, false
	}
	item := df.items[start]
	if !item.listItem {
		return graphql.Argument{Name: argName, Type: item.valueType}, true
	}
	return graphql.Argument{Name: argName, Type: item.listType}, true
}

func (df *DetailForm) argItems(argName string) []*formItem {
	var items []*formItem
	for i := 0; i < df.argCount; i++ {
		if df.items[i].argName == argName {
			items = append(items, &df.items[i])
		}
	}
	return items
}

func (df *DetailForm) setArgEnabled(argName string, enabled bool) {
	for i := 0; i < df.argCount; i++ {
		if df.items[i].argName == argName {
			df.items[i].enabled = enabled
		}
	}
}

func (df *DetailForm) syncListArgRows(argName string) {
	start, end, ok := df.argRange(argName)
	if !ok || !df.items[start].listItem {
		return
	}
	enabled := df.items[start].enabled
	arg, ok := df.argDefinition(argName)
	if !ok {
		return
	}

	df.syncGroupRange(start, end,
		func(item *formItem) int { return item.listGroup },
		func(group int) []formItem {
			return newListArgFormItems(arg, df.inputTypes, df.enumTypes, df.endpoint, group, enabled, df.nestedLists)
		},
	)
}

// boundaryGroupOf reports which element of a nested list boundary an item
// belongs to, derived from the item's own path rather than its (innermost
// only) listBoundary/boundaryGroup fields. This matches not just the
// boundary's own direct leaves but anything nested further inside one of
// its elements too (e.g. a list nested inside a nested list), which is what
// keeps a group's accounting correct even when that group has a field after
// a deeper nested list.
func boundaryGroupOf(item *formItem, fieldPath []varPathSeg) (int, bool) {
	if len(item.path) <= len(fieldPath) {
		return 0, false
	}
	for i, seg := range fieldPath {
		if item.path[i] != seg {
			return 0, false
		}
	}
	idxSeg := item.path[len(fieldPath)]
	if !idxSeg.isIndex {
		return 0, false
	}
	return idxSeg.index, true
}

// boundaryExtent finds the [start, end) span in df.items covering every item
// that belongs to a nested list boundary at any depth: its own direct
// leaves, and anything nested further inside one of its elements. Matching
// on the boundary's fieldPath (via boundaryGroupOf) rather than stopping at
// the first item whose innermost listBoundary differs means a group's own
// field that comes after a deeper nested list is still found, instead of
// being silently dropped once the scan reaches that inner list's items.
func (df *DetailForm) boundaryExtent(fieldPath []varPathSeg) (int, int, bool) {
	start := -1
	end := -1
	for i := 0; i < df.argCount; i++ {
		if _, ok := boundaryGroupOf(&df.items[i], fieldPath); !ok {
			continue
		}
		if start == -1 {
			start = i
		}
		end = i + 1
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}

// syncNestedListBoundary grows or shrinks the elements of a list-of-input-
// object field nested inside another argument, using the same grow/shrink
// rule as syncListArgRows but scoped to the field's own boundary rather than
// the whole argument.
func (df *DetailForm) syncNestedListBoundary(boundary string) {
	spec, ok := df.nestedLists[boundary]
	if !ok {
		return
	}
	start, end, ok := df.boundaryExtent(spec.fieldPath)
	if !ok {
		return
	}

	df.syncGroupRange(start, end,
		func(item *formItem) int {
			group, _ := boundaryGroupOf(item, spec.fieldPath)
			return group
		},
		func(group int) []formItem {
			return expandNestedListGroup(boundary, &spec, group, df.inputTypes, df.enumTypes, df.endpoint, df.nestedLists)
		},
	)
}

// syncGroupRange grows or shrinks a contiguous [start, end) run of argument
// items so it holds exactly one trailing empty group after the last group
// with a meaningful value (or a single empty group if none has one),
// regenerating newly added groups via generate. groupOf reports which group
// an item belongs to; shared by syncListArgRows and syncNestedListBoundary,
// which differ only in how items are grouped and regenerated.
func (df *DetailForm) syncGroupRange(
	start, end int,
	groupOf func(*formItem) int,
	generate func(group int) []formItem,
) {
	lastNonEmptyGroup := -1
	currentGroups := 0
	for i := start; i < end; {
		group := groupOf(&df.items[i])
		groupNonEmpty := false
		for i < end && groupOf(&df.items[i]) == group {
			if hasMeaningfulListValue(&df.items[i]) {
				groupNonEmpty = true
			}
			i++
		}
		if groupNonEmpty {
			lastNonEmptyGroup = group
		}
		currentGroups++
	}

	desiredGroups := 1
	if lastNonEmptyGroup >= 0 {
		desiredGroups = lastNonEmptyGroup + 2
	}

	if desiredGroups < currentGroups {
		cut := start
		for cut < end && groupOf(&df.items[cut]) < desiredGroups {
			cut++
		}
		removed := end - cut
		df.items = append(df.items[:cut], df.items[end:]...)
		df.argCount -= removed
		if df.cursor >= end {
			df.cursor -= removed
		} else if df.cursor >= cut {
			df.cursor = max(cut-1, 0)
		}
		end = cut
		currentGroups = desiredGroups
	}

	if desiredGroups > currentGroups {
		insertAt := end
		for group := currentGroups; group < desiredGroups; group++ {
			next := generate(group)
			df.items = append(df.items[:insertAt], append(next, df.items[insertAt:]...)...)
			df.argCount += len(next)
			insertAt += len(next)
		}
	}

	df.FocusCurrent()
}

func hasMeaningfulListValue(item *formItem) bool {
	if item == nil {
		return false
	}
	switch item.kind {
	case formItemTextInput:
		return strings.TrimSpace(item.Value()) != ""
	case formItemDropdown:
		return strings.TrimSpace(item.Value()) != ""
	case formItemToggle:
		return item.enabled
	default:
		return false
	}
}

func listDropdown(d tui.Dropdown) tui.Dropdown {
	options := make([]string, 0, len(d.Options)+1)
	options = append(options, "")
	options = append(options, d.Options...)
	return tui.NewDropdown(d.Label, options, 0)
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

// Search

func (df *DetailForm) IsSearching() bool { return df.search.Active() }

func (df *DetailForm) StartSearch() {
	df.preSearchCursor = df.cursor
	df.search.Start()
}

func (df *DetailForm) StopSearch(confirm bool) {
	df.search.Stop()
	if !confirm {
		df.cursor = df.preSearchCursor
		df.FocusCurrent()
	}
}

func (df *DetailForm) updateSearchMatches() {
	query := strings.ToLower(df.search.Query())
	var indices []int
	if query != "" {
		for i := range df.items {
			if strings.Contains(strings.ToLower(df.items[i].name), query) {
				indices = append(indices, i)
			}
		}
	}
	df.search.SetMatches(indices)
	df.syncSearchCursor()
}

func (df *DetailForm) syncSearchCursor() {
	if m := df.search.CurrentMatch(); m >= 0 {
		df.cursor = m
		df.FocusCurrent()
	}
}

func (df *DetailForm) SearchFooter() string {
	return df.search.Footer()
}

func (df *DetailForm) HandleSearchKey(msg tea.KeyMsg) tea.Cmd {
	stopped, confirmed, cmd := df.search.HandleKey(msg)
	if stopped {
		if !confirmed {
			df.cursor = df.preSearchCursor
			df.FocusCurrent()
		}
		return cmd
	}
	switch msg.String() {
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyDown, tui.KeyCtrlN:
		df.syncSearchCursor()
	default:
		df.updateSearchMatches()
	}
	return cmd
}

// HandleKey routes a key message to the currently focused item.
func (df *DetailForm) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if df.cursor < 0 || df.cursor >= len(df.items) {
		return nil
	}
	item := &df.items[df.cursor]
	key := msg.String()
	if item.kind == formItemToggle && key == tui.KeyEnter {
		msg = tea.KeyMsg{Type: tea.KeySpace}
		key = tui.KeySpace
	}

	// ── Argument items: Space toggles the enabled checkbox ──
	if !item.isField && key == tui.KeySpace && !item.ConsumesTextInput() {
		var cmd tea.Cmd
		newValue := !item.enabled
		if item.kind == formItemToggle {
			cmd = item.HandleKey(msg)
			newValue = item.toggle.Value
		}
		if item.listItem || item.name == item.argName {
			df.setArgEnabled(item.argName, newValue)
		} else {
			item.enabled = newValue
		}
		// A toggle's enabled state drives hasMeaningfulListValue directly, so
		// list rows must resync here too, or a boolean list/nested list can
		// never grow past its first element via keyboard.
		if item.listItem {
			df.syncListArgRows(item.argName)
		}
		if item.listBoundary != "" {
			df.syncNestedListBoundary(item.listBoundary)
		}
		return cmd
	}

	// Argument text inputs: Enter activates/deactivates editing
	if !item.isField && item.kind == formItemTextInput && key == tui.KeyEnter {
		if item.input.Model.Focused() {
			item.input.Model.Blur()
		} else {
			item.input.Model.Focus()
		}
		return nil
	}

	// Argument text inputs: Esc exits editing
	if !item.isField && item.kind == formItemTextInput &&
		key == tui.KeyCancel && item.input.Model.Focused() {
		item.input.Model.Blur()
		return nil
	}

	// Pass through to widget
	cmd := item.HandleKey(msg)

	// Both checks can apply to the same item (a list nested inside a
	// top-level list argument's element belongs to both), so run
	// independently rather than else-if.
	if !item.isField && item.listItem {
		df.syncListArgRows(item.argName)
	}
	if !item.isField && item.listBoundary != "" {
		df.syncNestedListBoundary(item.listBoundary)
	}

	// Field toggles: expand/collapse children on Space
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

func (df *DetailForm) itemZoneID(prefix string, index int) string {
	return prefix + ":item:" + strconv.Itoa(index)
}

func (df *DetailForm) HandleMouse(prefix string, msg tea.MouseMsg) bool {
	for i := range df.items {
		id := df.itemZoneID(prefix, i)
		if !tui.Hit(id, msg) {
			continue
		}

		df.cursor = i
		df.FocusCurrent()
		item := &df.items[i]

		switch item.kind {
		case formItemToggle:
			_ = item.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
			if !item.isField {
				if item.listItem || item.name == item.argName {
					df.setArgEnabled(item.argName, item.toggle.Value)
				} else {
					item.enabled = item.toggle.Value
				}
				if item.listItem {
					df.syncListArgRows(item.argName)
				}
				if item.listBoundary != "" {
					df.syncNestedListBoundary(item.listBoundary)
				}
			}
			if item.expandable {
				df.toggleExpand(i)
			}
		case formItemTextInput:
			if !item.isField {
				item.enabled = true
			}
			item.input.Model.Focus()
			if item.listItem {
				df.syncListArgRows(item.argName)
			}
			if item.listBoundary != "" {
				df.syncNestedListBoundary(item.listBoundary)
			}
		case formItemDropdown:
			if !item.dropdown.Expanded() {
				if !item.isField {
					item.enabled = true
				}
				item.dropdown.Focus()
				item.dropdown.Expand()
				return true
			}

			_, relY := tui.ZonePos(id, msg)
			optionIndex := relY
			if optionIndex < 0 || optionIndex >= len(item.dropdown.Options) {
				return true
			}
			item.dropdown.Select(optionIndex)
			item.dropdown.Focus()
			if !item.isField {
				item.enabled = true
			}
			if item.listItem {
				df.syncListArgRows(item.argName)
			}
			if item.listBoundary != "" {
				df.syncNestedListBoundary(item.listBoundary)
			}
		}
		return true
	}
	return false
}

// View renders all form items in a single flat list.
// Returns the rendered string and the line number of the focused item.
func (df *DetailForm) View(op *UnifiedOperation) (string, int) {
	return df.viewMarked(op, "", func(_ string, s string) string { return s })
}

func (df *DetailForm) ViewMarked(
	op *UnifiedOperation,
	zonePrefix string,
	mark func(id, view string) string,
) (string, int) {
	return df.viewMarked(op, zonePrefix, mark)
}

func (df *DetailForm) viewMarked(
	op *UnifiedOperation,
	zonePrefix string,
	mark func(id, view string) string,
) (string, int) {
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

		linePrefix := itemPad
		if i == df.cursor {
			if focused {
				linePrefix = strings.Repeat(tui.KeySpace, pad-2) + utils.ChevronRight
			}
			cursorLine = len(lines)
		}
		view := df.items[i].View()
		if zonePrefix != "" {
			view = mark(df.itemZoneID(zonePrefix, i), view)
		}
		for j, line := range strings.Split(view, "\n") {
			if j == 0 {
				lines = append(lines, linePrefix+line)
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
