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

// ColorProvider defines how to apply colors to JSON tokens.
type ColorProvider interface {
	ColorString(s string) string
	ColorNumber(s string) string
	ColorBool(s string) string
	ColorNull(s string) string
	ColorKey(s string) string
}

// LipglossColorProvider implements ColorProvider for both CLI and TUI output.
// NOTE: We intentionally use one fixed JSON palette here.
// Terminal/system light-dark inference was too unreliable across terminals,
// and several token colors washed out badly in light mode. Keep this simple
// for now with balanced fixed colors; a proper user-configurable theme can
// revisit light/dark palettes later.
type LipglossColorProvider struct{}

var (
	jsonStringColor  = lipgloss.AdaptiveColor{Light: "#3E8F53", Dark: "#3E8F53"}
	jsonNumberColor  = lipgloss.AdaptiveColor{Light: "#2F6FA3", Dark: "#2F6FA3"}
	jsonBooleanColor = lipgloss.AdaptiveColor{Light: "#8C5A00", Dark: "#8C5A00"}
	jsonNullColor    = lipgloss.AdaptiveColor{Light: "#7653A6", Dark: "#7653A6"}
)

func (l LipglossColorProvider) ColorString(s string) string {
	return lipgloss.NewStyle().Foreground(jsonStringColor).Render(s)
}

func (l LipglossColorProvider) ColorNumber(s string) string {
	return lipgloss.NewStyle().Foreground(jsonNumberColor).Render(s)
}

func (l LipglossColorProvider) ColorBool(s string) string {
	return lipgloss.NewStyle().Foreground(jsonBooleanColor).Render(s)
}

func (l LipglossColorProvider) ColorNull(s string) string {
	return lipgloss.NewStyle().Foreground(jsonNullColor).Render(s)
}
func (l LipglossColorProvider) ColorKey(s string) string { return s }

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
	formatted, err := FormatJSONColored(data, LipglossColorProvider{})
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
