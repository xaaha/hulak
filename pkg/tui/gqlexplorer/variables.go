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

		if IsListType(arg.Type) {
			values := buildListVariableValues(argItems)
			if len(values) == 0 {
				continue
			}
			entries = append(entries, variableEntry{
				key:   arg.Name,
				value: "[" + strings.Join(values, ", ") + "]",
			})
			continue
		}

		if len(argItems) == 1 && argItems[0].name == arg.Name {
			if value, ok := formatVariableValue(argItems[0]); ok {
				entries = append(entries, variableEntry{key: arg.Name, value: value})
			}
			continue
		}

		var objectEntries []variableEntry
		for _, item := range argItems {
			if value, ok := formatVariableValue(item); ok {
				objectEntries = append(objectEntries, variableEntry{
					key:   item.name,
					value: value,
				})
			}
		}
		if len(objectEntries) == 0 {
			continue
		}
		entries = append(entries, variableEntry{
			key:   arg.Name,
			value: renderVariablesObject(objectEntries, 1),
		})
	}

	if len(entries) == 0 {
		return ""
	}
	return renderVariablesObject(entries, 0)
}

func buildListVariableValues(items []*formItem) []string {
	if len(items) == 0 {
		return nil
	}

	var values []string
	for i := 0; i < len(items); {
		group := items[i].listGroup
		groupItems := items[i : i+1]
		j := i + 1
		for j < len(items) && items[j].listGroup == group {
			groupItems = items[i : j+1]
			j++
		}
		if len(groupItems) == 1 {
			if value, ok := formatVariableValue(groupItems[0]); ok {
				values = append(values, value)
			}
			i = j
			continue
		}

		var objectEntries []variableEntry
		for _, item := range groupItems {
			if value, ok := formatVariableValue(item); ok {
				objectEntries = append(objectEntries, variableEntry{
					key:   item.name,
					value: value,
				})
			}
		}
		if len(objectEntries) > 0 {
			values = append(values, renderVariablesObject(objectEntries, 1))
		}
		i = j
	}
	return values
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

		if IsListType(arg.Type) {
			values := listGoValues(argItems)
			if len(values) == 0 {
				continue
			}
			result[arg.Name] = values
			continue
		}

		if len(argItems) == 1 && argItems[0].name == arg.Name {
			if value, ok := goValue(argItems[0]); ok {
				result[arg.Name] = value
			}
			continue
		}

		obj := make(map[string]any)
		for _, item := range argItems {
			if value, ok := goValue(item); ok {
				obj[item.name] = value
			}
		}
		if len(obj) > 0 {
			result[arg.Name] = obj
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func listGoValues(items []*formItem) []any {
	if len(items) == 0 {
		return nil
	}

	var values []any
	for i := 0; i < len(items); {
		group := items[i].listGroup
		groupItems := items[i : i+1]
		j := i + 1
		for j < len(items) && items[j].listGroup == group {
			groupItems = items[i : j+1]
			j++
		}
		if len(groupItems) == 1 {
			if value, ok := goValue(groupItems[0]); ok {
				values = append(values, value)
			}
			i = j
			continue
		}

		obj := make(map[string]any)
		for _, item := range groupItems {
			if value, ok := goValue(item); ok {
				obj[item.name] = value
			}
		}
		if len(obj) > 0 {
			values = append(values, obj)
		}
		i = j
	}
	return values
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
