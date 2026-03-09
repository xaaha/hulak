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
		return formatTypedVariableText(item.Value(), item.typeHint)
	default:
		return "", false
	}
}

func formatTypedVariableText(raw, typeHint string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(strings.TrimSpace(typeHint), "[") {
		return formatListVariableText(trimmed, typeHint)
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

func formatListVariableText(raw, typeHint string) (string, bool) {
	if json.Valid([]byte(raw)) {
		var arr []any
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			return raw, true
		}
	}

	itemType := extractListItemType(typeHint)
	parts := splitListInput(raw)
	if len(parts) == 0 {
		return "", false
	}

	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value, ok := formatTypedVariableText(part, itemType)
		if !ok {
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return "", false
	}
	return "[" + strings.Join(values, ", ") + "]", true
}

func extractListItemType(typeHint string) string {
	t := strings.TrimSpace(typeHint)
	t = strings.TrimSuffix(t, "!")
	t = strings.TrimPrefix(t, "[")
	t = strings.TrimSuffix(t, "]")
	return strings.TrimSpace(t)
}

func splitListInput(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	if !strings.ContainsAny(raw, ",\n") {
		return strings.Fields(raw)
	}

	var parts []string
	for _, chunk := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n'
	}) {
		trimmed := strings.TrimSpace(chunk)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
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
