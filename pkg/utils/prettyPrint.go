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
// Uses basic ANSI palette indices (0–15) so the terminal's own palette handles
// light/dark adaptation — no manual AdaptiveColor mapping needed.
type LipglossColorProvider struct{}

var (
	lgString  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	lgNumber  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	lgBoolean = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	lgNull    = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

func (l LipglossColorProvider) ColorString(s string) string { return lgString.Render(s) }
func (l LipglossColorProvider) ColorNumber(s string) string { return lgNumber.Render(s) }
func (l LipglossColorProvider) ColorBool(s string) string   { return lgBoolean.Render(s) }
func (l LipglossColorProvider) ColorNull(s string) string   { return lgNull.Render(s) }
func (l LipglossColorProvider) ColorKey(s string) string    { return s }

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
