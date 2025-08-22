// Package utils in this file pretty prints json
package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Safe ANSI 8 colors
const (
	reset   = "\033[0m"
	cyan    = "\033[96m" // number
	yellow  = "\033[93m" // booleans
	green   = "\033[92m" // strings
	magenta = "\033[95m" // null
)

// PrintJSONColored Pretty-prints JSON with safe ANSI colors
func PrintJSONColored(data []byte) error {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err // not valid JSON
	}

	var buf bytes.Buffer
	marshalValue(obj, &buf, 0)
	fmt.Println(buf.String())
	return nil
}

func marshalValue(val any, buf *bytes.Buffer, depth int) {
	switch v := val.(type) {
	case map[string]any:
		marshalMap(v, buf, depth)
	case []any:
		marshalArray(v, buf, depth)
	case string:
		s, _ := json.Marshal(v) // adds quotes & escapes
		buf.WriteString(green + string(s) + reset)
	case float64:
		buf.WriteString(cyan + strconv.FormatFloat(v, 'f', -1, 64) + reset)
	case bool:
		buf.WriteString(yellow + strconv.FormatBool(v) + reset)
	case nil:
		buf.WriteString(magenta + "null" + reset)
	default:
		fmt.Fprintf(buf, "%v", v)
	}
}

func marshalMap(m map[string]any, buf *bytes.Buffer, depth int) {
	if len(m) == 0 {
		buf.WriteString("{}")
		return
	}

	buf.WriteString("{\n")
	indent := strings.Repeat("  ", depth+1)

	// sort keys for stable output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		buf.WriteString(indent)
		buf.WriteString(fmt.Sprintf("\"%s\": ", k) + reset)
		marshalValue(m[k], buf, depth+1)
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(strings.Repeat("  ", depth) + "}")
}

func marshalArray(a []any, buf *bytes.Buffer, depth int) {
	if len(a) == 0 {
		buf.WriteString("[]")
		return
	}

	buf.WriteString("[\n")
	indent := strings.Repeat("  ", depth+1)

	for i, v := range a {
		buf.WriteString(indent)
		marshalValue(v, buf, depth+1)
		if i < len(a)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(strings.Repeat("  ", depth) + "]")
}
