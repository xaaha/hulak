package gqlexplorer

import (
	"fmt"
	"strings"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

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
