package gqlexplorer

import (
	"strings"
)

const queryIndent = "  "

// BuildQueryString generates a formatted GraphQL query string from the
// operation definition and the current detail form state. It includes
// variable declarations for all arguments and a selection set for any
// return-type fields the user has toggled on.
//
// When df is nil the query contains just the operation call with no
// selection set, matching the format:
//
//	query Foo { Foo }
func BuildQueryString(op *UnifiedOperation, df *DetailForm) string {
	if op == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(string(op.Type))
	sb.WriteString(" ")
	sb.WriteString(op.Name)

	if len(op.Arguments) > 0 {
		sb.WriteString("(")
		for i, arg := range op.Arguments {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("$")
			sb.WriteString(arg.Name)
			sb.WriteString(": ")
			sb.WriteString(arg.Type)
		}
		sb.WriteString(")")
	}
	sb.WriteString(" {\n")

	sb.WriteString(queryIndent)
	sb.WriteString(op.Name)

	if len(op.Arguments) > 0 {
		sb.WriteString("(")
		for i, arg := range op.Arguments {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.Name)
			sb.WriteString(": $")
			sb.WriteString(arg.Name)
		}
		sb.WriteString(")")
	}

	if df != nil {
		fields := df.items[df.argCount:]
		if lines, _ := buildSelectionLines(fields, 0, -1, 2); len(lines) > 0 {
			sb.WriteString(" {\n")
			sb.WriteString(strings.Join(lines, "\n"))
			sb.WriteString("\n")
			sb.WriteString(queryIndent)
			sb.WriteString("}")
		}
	}

	sb.WriteString("\n}")
	return sb.String()
}

// buildSelectionLines walks the field form items and returns indented lines
// for every toggled-on field. Expandable (object-type) fields recurse into
// their children, producing nested { } blocks. Items with depth ≤ parentDepth
// signal the end of the current nesting level.
//
// level controls indentation: each level adds one queryIndent unit.
// Top-level fields inside a selection set start at level 2 (operation body
// indent + selection set indent).
func buildSelectionLines(items []formItem, start, parentDepth, level int) ([]string, int) {
	var lines []string
	indent := strings.Repeat(queryIndent, level)
	i := start

	for i < len(items) {
		item := &items[i]
		if item.depth <= parentDepth {
			break
		}

		if item.kind == formItemToggle && item.toggle.Value {
			if item.expandable {
				childLines, consumed := buildSelectionLines(
					items, i+1, item.depth, level+1,
				)
				if len(childLines) > 0 {
					lines = append(lines, indent+item.name+" {")
					lines = append(lines, childLines...)
					lines = append(lines, indent+"}")
				}
				i += 1 + consumed
				continue
			}
			lines = append(lines, indent+item.name)
		}
		i++
	}

	return lines, i - start
}
