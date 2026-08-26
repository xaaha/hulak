package gqlexplorer

import (
	"encoding/json"
	"strconv"
	"strings"
)

type variableEntry struct {
	key   string
	value string
}

// varPathSeg is one step in the location of a form item's value within its
// top-level argument. A key segment descends into an input-object field; an
// index segment descends into a list element. The full path lets the variables
// builder reconstruct arbitrarily nested objects and lists from the flat list
// of form items.
type varPathSeg struct {
	key     string
	index   int
	isIndex bool
}

// varTree is an ordered value tree assembled from form-item paths. Object key
// order and list element order follow insertion order, which mirrors the schema
// field order and list group order of the form items.
type varTree struct {
	kind   varTreeKind
	leaf   string // formatted GraphQL literal for the string builder
	goLeaf any    // native value for the map builder
	keys   []string
	objs   map[string]*varTree
	list   []*varTree
	listAt map[int]int // list group index -> position in list
}

type varTreeKind int

const (
	varTreeLeaf varTreeKind = iota
	varTreeObject
	varTreeList
)

func (t *varTree) insert(path []varPathSeg, leaf string, goLeaf any) {
	if len(path) == 0 {
		t.kind = varTreeLeaf
		t.leaf = leaf
		t.goLeaf = goLeaf
		return
	}
	seg := path[0]
	if seg.isIndex {
		if t.kind != varTreeList {
			t.kind = varTreeList
			t.listAt = make(map[int]int)
		}
		pos, ok := t.listAt[seg.index]
		if !ok {
			child := &varTree{}
			t.list = append(t.list, child)
			pos = len(t.list) - 1
			t.listAt[seg.index] = pos
		}
		t.list[pos].insert(path[1:], leaf, goLeaf)
		return
	}

	if t.kind != varTreeObject {
		t.kind = varTreeObject
		t.objs = make(map[string]*varTree)
	}
	child, ok := t.objs[seg.key]
	if !ok {
		child = &varTree{}
		t.objs[seg.key] = child
		t.keys = append(t.keys, seg.key)
	}
	child.insert(path[1:], leaf, goLeaf)
}

// argValueTree assembles the value tree for a single argument from its form
// items. It returns false when no enabled item contributes a value.
func argValueTree(items []*formItem) (*varTree, bool) {
	root := &varTree{}
	filled := false
	for _, item := range items {
		strVal, ok := formatVariableValue(item)
		if !ok {
			continue
		}
		goVal, _ := goValue(item)
		root.insert(item.path, strVal, goVal)
		filled = true
	}
	return root, filled
}

// BuildVariablesString renders the GraphQL variables payload implied by the
// current detail form state. Only enabled arguments with concrete values are
// included, so empty text inputs are omitted until the user provides a value.
func BuildVariablesString(op *UnifiedOperation, df *DetailForm) string {
	if op == nil || df == nil || df.argCount == 0 {
		return ""
	}

	var entries []variableEntry
	for _, arg := range op.Arguments {
		argItems := df.argItems(arg.Name)
		if len(argItems) == 0 {
			continue
		}
		tree, ok := argValueTree(argItems)
		if !ok {
			continue
		}
		entries = append(entries, variableEntry{
			key:   arg.Name,
			value: renderVarTreeString(tree, 1),
		})
	}

	if len(entries) == 0 {
		return ""
	}
	return renderVariablesObject(entries, 0)
}

func renderVarTreeString(t *varTree, level int) string {
	switch t.kind {
	case varTreeObject:
		entries := make([]variableEntry, 0, len(t.keys))
		for _, k := range t.keys {
			entries = append(entries, variableEntry{
				key:   k,
				value: renderVarTreeString(t.objs[k], level+1),
			})
		}
		return renderVariablesObject(entries, level)
	case varTreeList:
		parts := make([]string, 0, len(t.list))
		for _, child := range t.list {
			parts = append(parts, renderVarTreeString(child, level))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return t.leaf
	}
}

func formatVariableValue(item *formItem) (string, bool) {
	if item == nil {
		return "", false
	}
	if !item.enabled {
		return "", false
	}

	switch item.kind {
	case formItemToggle:
		return item.Value(), true
	case formItemDropdown:
		if strings.TrimSpace(item.Value()) == "" {
			return "", false
		}
		return marshalJSONString(item.Value()), true
	case formItemTextInput:
		return formatTypedVariableText(item.Value(), item.valueType)
	default:
		return "", false
	}
}

func formatTypedVariableText(raw, typeHint string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	if strings.EqualFold(trimmed, "null") {
		return "null", true
	}

	switch ExtractBaseType(typeHint) {
	case "String", "ID":
		return marshalJSONString(trimmed), true
	case "Int":
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed, true
		}
		return trimmed, true
	case "Float":
		if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return trimmed, true
		}
		return trimmed, true
	case "Boolean":
		lower := strings.ToLower(trimmed)
		if lower == "true" || lower == "false" {
			return lower, true
		}
		return trimmed, true
	default:
		if json.Valid([]byte(trimmed)) {
			return trimmed, true
		}
		return marshalJSONString(trimmed), true
	}
}

func renderVariablesObject(entries []variableEntry, level int) string {
	if len(entries) == 0 {
		return "{}"
	}

	indent := strings.Repeat(queryIndent, level)
	itemIndent := strings.Repeat(queryIndent, level+1)

	var sb strings.Builder
	sb.WriteString("{\n")
	for i, entry := range entries {
		sb.WriteString(itemIndent)
		sb.WriteString(marshalJSONString(entry.key))
		sb.WriteString(": ")
		sb.WriteString(entry.value)
		if i < len(entries)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(indent)
	sb.WriteString("}")
	return sb.String()
}

func marshalJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func BuildVariablesMap(op *UnifiedOperation, df *DetailForm) map[string]any {
	if op == nil || df == nil || df.argCount == 0 {
		return nil
	}

	result := make(map[string]any)
	for _, arg := range op.Arguments {
		argItems := df.argItems(arg.Name)
		if len(argItems) == 0 {
			continue
		}
		tree, ok := argValueTree(argItems)
		if !ok {
			continue
		}
		result[arg.Name] = renderVarTreeGo(tree)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func renderVarTreeGo(t *varTree) any {
	switch t.kind {
	case varTreeObject:
		obj := make(map[string]any, len(t.keys))
		for _, k := range t.keys {
			obj[k] = renderVarTreeGo(t.objs[k])
		}
		return obj
	case varTreeList:
		values := make([]any, 0, len(t.list))
		for _, child := range t.list {
			values = append(values, renderVarTreeGo(child))
		}
		return values
	default:
		return t.goLeaf
	}
}

func goValue(item *formItem) (any, bool) {
	if item == nil || !item.enabled {
		return nil, false
	}

	switch item.kind {
	case formItemToggle:
		return item.Value() == "true", true
	case formItemDropdown:
		if strings.TrimSpace(item.Value()) == "" {
			return nil, false
		}
		return item.Value(), true
	case formItemTextInput:
		return typedGoValue(item.Value(), item.valueType)
	default:
		return nil, false
	}
}

func typedGoValue(raw, typeHint string) (any, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	if strings.EqualFold(trimmed, "null") {
		return nil, true
	}

	switch ExtractBaseType(typeHint) {
	case "String", "ID":
		return trimmed, true
	case "Int":
		if v, err := strconv.Atoi(trimmed); err == nil {
			return v, true
		}
		return trimmed, true
	case "Float":
		if v, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return v, true
		}
		return trimmed, true
	case "Boolean":
		lower := strings.ToLower(trimmed)
		if lower == "true" {
			return true, true
		}
		if lower == "false" {
			return false, true
		}
		return trimmed, true
	default:
		if json.Valid([]byte(trimmed)) {
			var parsed any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return parsed, true
			}
		}
		return trimmed, true
	}
}
