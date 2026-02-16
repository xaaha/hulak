package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// ColorProvider defines how to apply colors to JSON tokens.
// Implement this for different rendering backends (CLI, TUI, etc).
type ColorProvider interface {
	ColorString(s string) string
	ColorNumber(s string) string
	ColorBool(s string) string
	ColorNull(s string) string
	ColorKey(s string) string
}

type fatihColorProvider struct{}

var (
	green   = color.New(color.FgHiGreen)   // for strings
	yellow  = color.New(color.FgHiYellow)  // for booleans
	cyan    = color.New(color.FgHiCyan)    // for numbers
	magenta = color.New(color.FgHiMagenta) // for null
)

func (f fatihColorProvider) ColorString(s string) string { return green.Sprint(s) }
func (f fatihColorProvider) ColorNumber(s string) string { return cyan.Sprint(s) }
func (f fatihColorProvider) ColorBool(s string) string   { return yellow.Sprint(s) }
func (f fatihColorProvider) ColorNull(s string) string   { return magenta.Sprint(s) }
func (f fatihColorProvider) ColorKey(s string) string    { return s }

// FormatJSONColored formats JSON data as an indented, colored string using the given ColorProvider.
func FormatJSONColored(data []byte, provider ColorProvider) (string, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	marshalValue(obj, &buf, 0, provider)
	return buf.String(), nil
}

// PrintJSONColored pretty-prints JSON to stdout using fatih/color.
// Needed as CLI needs std-out
func PrintJSONColored(data []byte) error {
	formatted, err := FormatJSONColored(data, fatihColorProvider{})
	if err != nil {
		return err
	}
	fmt.Println(formatted)
	return nil
}

func marshalValue(val any, buf *bytes.Buffer, depth int, provider ColorProvider) {
	switch v := val.(type) {
	case map[string]any:
		marshalMap(v, buf, depth, provider)
	case []any:
		marshalArray(v, buf, depth, provider)
	case string:
		s, _ := json.Marshal(v)
		buf.WriteString(provider.ColorString(string(s)))
	case float64:
		buf.WriteString(provider.ColorNumber(strconv.FormatFloat(v, 'f', -1, 64)))
	case bool:
		buf.WriteString(provider.ColorBool(strconv.FormatBool(v)))
	case nil:
		buf.WriteString(provider.ColorNull("null"))
	default:
		fmt.Fprintf(buf, "%v", v)
	}
}

func marshalMap(jsonMap map[string]any, buf *bytes.Buffer, depth int, provider ColorProvider) {
	if len(jsonMap) == 0 {
		buf.WriteString("{}")
		return
	}

	buf.WriteString("{\n")
	indent := strings.Repeat("  ", depth+1)

	keys := make([]string, 0, len(jsonMap))
	for k := range jsonMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		buf.WriteString(indent)
		fmt.Fprintf(buf, "\"%s\": ", provider.ColorKey(k))
		marshalValue(jsonMap[k], buf, depth+1, provider)
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(strings.Repeat("  ", depth) + "}")
}

func marshalArray(jsonArray []any, buf *bytes.Buffer, depth int, provider ColorProvider) {
	if len(jsonArray) == 0 {
		buf.WriteString("[]")
		return
	}

	buf.WriteString("[\n")
	indent := strings.Repeat("  ", depth+1)

	for idx, val := range jsonArray {
		buf.WriteString(indent)
		marshalValue(val, buf, depth+1, provider)
		if idx < len(jsonArray)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(strings.Repeat("  ", depth) + "]")
}
