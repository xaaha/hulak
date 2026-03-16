package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSI colors (0-15) are defined by the terminal's own color scheme.
// When the user switches between dark/light themes, the terminal
// remaps these automatically — no background detection needed.
var (
	jsonStringColor = lipgloss.Color("2") // green
	jsonNumberColor = lipgloss.Color("4") // blue
	jsonBoolColor   = lipgloss.Color("3") // yellow
	jsonNullColor   = lipgloss.Color("5") // magenta
	jsonKeyColor    = lipgloss.Color("6") // cyan
)

// ColorProvider defines how to apply colors to JSON tokens.
type ColorProvider interface {
	ColorString(s string) string
	ColorNumber(s string) string
	ColorBool(s string) string
	ColorNull(s string) string
	ColorKey(s string) string
}

// LipglossColorProvider implements ColorProvider using lipgloss ANSI colors.
type LipglossColorProvider struct{}

// JSONColors is the shared color provider for JSON token coloring.
// Use this instead of constructing LipglossColorProvider{} inline.
var JSONColors ColorProvider = LipglossColorProvider{}

func (l LipglossColorProvider) ColorString(s string) string {
	return lipgloss.NewStyle().Foreground(jsonStringColor).Render(s)
}

func (l LipglossColorProvider) ColorNumber(s string) string {
	return lipgloss.NewStyle().Foreground(jsonNumberColor).Render(s)
}

func (l LipglossColorProvider) ColorBool(s string) string {
	return lipgloss.NewStyle().Foreground(jsonBoolColor).Bold(true).Render(s)
}

func (l LipglossColorProvider) ColorNull(s string) string {
	return lipgloss.NewStyle().Foreground(jsonNullColor).Italic(true).Render(s)
}

func (l LipglossColorProvider) ColorKey(s string) string {
	return lipgloss.NewStyle().Foreground(jsonKeyColor).Bold(true).Render(s)
}

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

// PrintJSONColored pretty-prints JSON to stdout with colored tokens.
func PrintJSONColored(data []byte) error {
	formatted, err := FormatJSONColored(data, JSONColors)
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

		keyJSON, _ := json.Marshal(k)
		keyStr := strings.TrimSuffix(strings.TrimPrefix(string(keyJSON), `"`), `"`)

		buf.WriteString(`"`)
		buf.WriteString(provider.ColorKey(keyStr))
		buf.WriteString(`": `)

		marshalValue(jsonMap[k], buf, depth+1, provider)
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(strings.Repeat("  ", depth))
	buf.WriteString("}")
}

func marshalArray(jsonArray []any, buf *bytes.Buffer, depth int, provider ColorProvider) {
	if len(jsonArray) == 0 {
		buf.WriteString("[]")
		return
	}

	buf.WriteString("[\n")
	indent := strings.Repeat("  ", depth+1)

	for i, val := range jsonArray {
		buf.WriteString(indent)
		marshalValue(val, buf, depth+1, provider)
		if i < len(jsonArray)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(strings.Repeat("  ", depth))
	buf.WriteString("]")
}
