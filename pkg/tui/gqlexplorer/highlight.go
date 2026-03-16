package gqlexplorer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

var (
	gqlKeywordStyle     = lipgloss.NewStyle().Foreground(tui.ColorPrimary).Bold(true)
	gqlOperationStyle   = lipgloss.NewStyle().Foreground(tui.ColorSecondary)
	gqlVariableStyle    = lipgloss.NewStyle().Foreground(tui.ColorWarn)
	gqlPunctuationStyle = lipgloss.NewStyle().Foreground(tui.ColorMuted)
)

func formatVariablesForPanel(raw string, focused bool) string {
	if !focused || strings.TrimSpace(raw) == "" {
		return raw
	}
	return colorizeJSONPreservingLayout(raw)
}

func formatQueryForPanel(raw string, focused bool) string {
	if !focused || raw == "" {
		return raw
	}

	var out strings.Builder
	expectOperationName := false
	for i := 0; i < len(raw); {
		switch ch := raw[i]; {
		case ch == '$':
			expectOperationName = false
			j := i + 1
			for j < len(raw) && isGraphQLIdentChar(rune(raw[j])) {
				j++
			}
			out.WriteString(gqlVariableStyle.Render(raw[i:j]))
			i = j
		case isGraphQLPunctuation(ch):
			if expectOperationName {
				expectOperationName = false
			}
			out.WriteString(gqlPunctuationStyle.Render(raw[i : i+1]))
			i++
		case isGraphQLIdentStart(rune(ch)):
			j := i + 1
			for j < len(raw) && isGraphQLIdentChar(rune(raw[j])) {
				j++
			}
			token := raw[i:j]
			switch {
			case isGraphQLKeyword(token):
				out.WriteString(gqlKeywordStyle.Render(token))
				expectOperationName = expectsOperationName(token)
			case expectOperationName:
				out.WriteString(gqlOperationStyle.Render(token))
				expectOperationName = false
			default:
				out.WriteString(token)
			}
			i = j
		default:
			out.WriteByte(raw[i])
			i++
		}
	}

	return out.String()
}

func expectsOperationName(token string) bool {
	switch token {
	case "query", "mutation", "subscription", "fragment":
		return true
	default:
		return false
	}
}

func isGraphQLKeyword(token string) bool {
	switch token {
	case "query", "mutation", "subscription", "fragment", "on":
		return true
	default:
		return false
	}
}

func isGraphQLPunctuation(ch byte) bool {
	switch ch {
	case '{', '}', '(', ')', '[', ']', ':', ',', '!':
		return true
	default:
		return false
	}
}

func isGraphQLIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isGraphQLIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func colorizeJSONPreservingLayout(raw string) string {
	var out strings.Builder

	for i := 0; i < len(raw); {
		switch ch := raw[i]; {
		case ch == '"':
			token, next, ok := scanJSONString(raw, i)
			if !ok {
				out.WriteByte(raw[i])
				i++
				continue
			}
			if isJSONKey(raw, next) {
				out.WriteString(token)
			} else {
				out.WriteString(utils.JSONColors.ColorString(token))
			}
			i = next
		case isJSONNumberStart(raw, i):
			token, next := scanJSONNumber(raw, i)
			out.WriteString(utils.JSONColors.ColorNumber(token))
			i = next
		case strings.HasPrefix(raw[i:], "true"):
			out.WriteString(utils.JSONColors.ColorBool("true"))
			i += len("true")
		case strings.HasPrefix(raw[i:], "false"):
			out.WriteString(utils.JSONColors.ColorBool("false"))
			i += len("false")
		case strings.HasPrefix(raw[i:], "null"):
			out.WriteString(utils.JSONColors.ColorNull("null"))
			i += len("null")
		default:
			out.WriteByte(raw[i])
			i++
		}
	}

	return out.String()
}

func scanJSONString(raw string, start int) (string, int, bool) {
	i := start + 1
	for i < len(raw) {
		switch raw[i] {
		case '\\':
			i += 2
		case '"':
			return raw[start : i+1], i + 1, true
		default:
			_, width := utf8.DecodeRuneInString(raw[i:])
			if width <= 0 {
				return "", start, false
			}
			i += width
		}
	}
	return "", start, false
}

func isJSONKey(raw string, pos int) bool {
	for pos < len(raw) {
		switch raw[pos] {
		case ' ', '\n', '\r', '\t':
			pos++
		case ':':
			return true
		default:
			return false
		}
	}
	return false
}

func isJSONNumberStart(raw string, i int) bool {
	if i >= len(raw) {
		return false
	}
	ch := raw[i]
	if ch != '-' && (ch < '0' || ch > '9') {
		return false
	}
	if i > 0 {
		prev := raw[i-1]
		if (prev >= '0' && prev <= '9') || prev == '"' || prev == '_' {
			return false
		}
	}
	return true
}

func scanJSONNumber(raw string, start int) (string, int) {
	i := start
	if raw[i] == '-' {
		i++
	}
	for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
		i++
	}
	if i < len(raw) && raw[i] == '.' {
		i++
		for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
			i++
		}
	}
	if i < len(raw) && (raw[i] == 'e' || raw[i] == 'E') {
		i++
		if i < len(raw) && (raw[i] == '+' || raw[i] == '-') {
			i++
		}
		for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
			i++
		}
	}
	return raw[start:i], i
}
