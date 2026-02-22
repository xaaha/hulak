package gqlexplorer

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

const (
	verticalDivider   = "│"
	horizontalDivider = "─"
)

var (
	itemPrefix     = strings.Repeat(tui.KeySpace, itemPadding)
	selectedPrefix = strings.Repeat(
		tui.KeySpace,
		itemPadding-len(utils.ChevronRight),
	) + utils.ChevronRight
	detailPrefix = strings.Repeat(tui.KeySpace, detailPadding)
	toggleOff    = strings.Repeat(tui.KeySpace, 2)
	toggleOn     = checkMark + tui.KeySpace
)

func appendWrappedHelpLines(lines []string, text string, width int, prefix string) []string {
	if text == "" {
		return lines
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	for line := range strings.SplitSeq(wrapped, "\n") {
		lines = append(lines, tui.HelpStyle.Render(prefix+line))
	}
	return lines
}

func (m Model) renderList() (string, int) {
	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render(
			strings.Repeat(tui.KeySpace, itemPadding-len(utils.ChevronRight)) + noMatchesLabel,
		), 0
	}

	var lines []string
	cursorLine := 0
	var currentType OperationType
	for i, op := range m.filtered {
		if op.Type != currentType {
			currentType = op.Type
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, tui.RenderBadge(string(currentType), badgeColor[currentType]))
		}
		if i == m.cursor {
			cursorLine = len(lines)
			lines = append(lines, tui.SubtitleStyle.Render(selectedPrefix+op.Name))
			wrapW := m.leftPanelWidth() - detailPadding
			lines = appendWrappedHelpLines(lines, op.Description, wrapW, detailPrefix)
			// Full URL shown intentionally; badges/filters use shortened form.
			lines = appendWrappedHelpLines(lines, op.Endpoint, wrapW, detailPrefix)
		} else {
			lines = append(lines, itemPrefix+op.Name)
		}
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m Model) renderEndpointPicker() (string, int) {
	if len(m.endpoints) == 0 {
		return tui.HelpStyle.Render(itemPrefix + noMatchesLabel), 0
	}

	var lines []string
	cursorLine := 0
	for i, ep := range m.endpoints {
		prefix := itemPrefix
		if i == m.endpointCursor {
			prefix = selectedPrefix
			cursorLine = len(lines)
		}
		toggle := toggleOff
		if m.pendingEndpoints[ep] {
			toggle = toggleOn
		}
		style := lipgloss.NewStyle()
		if i == m.endpointCursor {
			style = tui.SubtitleStyle
		}
		lines = append(lines, style.Render(prefix+toggle+ep))
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m Model) renderBadges() string {
	var shortened []string
	for ep := range m.activeEndpoints {
		shortened = append(shortened, ep)
	}
	sort.Strings(shortened)

	maxW := m.leftPanelWidth()
	ellipsis := tui.HelpStyle.Render(utils.Ellipsis)
	ellipsisW := lipgloss.Width(tui.KeySpace + ellipsis)
	var result string
	for i, s := range shortened {
		badge := tui.RenderBadge(s, tui.ColorMuted)
		candidate := result
		if candidate != "" {
			candidate += tui.KeySpace
		}
		candidate += badge
		if i > 0 && lipgloss.Width(candidate)+ellipsisW > maxW {
			result += tui.KeySpace + ellipsis
			break
		}
		if i == 0 && lipgloss.Width(candidate) > maxW {
			return ellipsis
		}
		result = candidate
	}
	return result
}

func renderDetail(
	op UnifiedOperation,
	inputTypes map[string]graphql.InputType,
) string {
	pad := strings.Repeat(tui.KeySpace, 2)
	argPad := strings.Repeat(tui.KeySpace, 4)

	var lines []string

	lines = append(lines, tui.SubtitleStyle.Render(utils.ChevronRight+op.Name))
	lines = append(lines, "")

	if op.ReturnType != "" {
		lines = append(lines, pad+tui.HelpStyle.Render("Returns: ")+op.ReturnType)
		lines = append(lines, "")
	}

	if len(op.Arguments) > 0 {
		lines = append(lines, pad+tui.HelpStyle.Render("Arguments:"))

		nameW, typeW := 0, 0
		for _, arg := range op.Arguments {
			if len(arg.Name) > nameW {
				nameW = len(arg.Name)
			}
			if len(arg.Type) > typeW {
				typeW = len(arg.Type)
			}
		}

		for _, arg := range op.Arguments {
			required := ""
			if strings.HasSuffix(arg.Type, "!") {
				required = tui.KeySpace + tui.HelpStyle.Render("(required)")
			}
			name := fmt.Sprintf("%-*s", nameW, arg.Name)
			typStr := fmt.Sprintf("%-*s", typeW, arg.Type)
			lines = append(lines, argPad+name+tui.KeySpace+tui.KeySpace+typStr+required)

			base := ExtractBaseType(arg.Type)
			if it, ok := resolveInputType(inputTypes, op.Endpoint, base); ok {
				lines = appendInputTypeFields(
					lines,
					it,
					argPad+strings.Repeat(tui.KeySpace, 2),
					inputTypes,
					op.Endpoint,
					1,
				)
			}
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func resolveInputType(
	inputTypes map[string]graphql.InputType,
	endpoint string,
	baseType string,
) (graphql.InputType, bool) {
	if it, ok := inputTypes[ScopedTypeKey(endpoint, baseType)]; ok {
		return it, true
	}
	it, ok := inputTypes[baseType]
	return it, ok
}

const maxInputTypeDepth = 3

func appendInputTypeFields(
	lines []string,
	it graphql.InputType,
	indent string,
	inputTypes map[string]graphql.InputType,
	endpoint string,
	depth int,
) []string {
	for i, f := range it.Fields {
		connector := "├─"
		if i == len(it.Fields)-1 {
			connector = "└─"
		}
		line := indent + tui.HelpStyle.Render(connector) +
			tui.KeySpace + f.Name +
			tui.KeySpace + tui.KeySpace + tui.HelpStyle.Render(f.Type)
		lines = append(lines, line)

		if depth < maxInputTypeDepth {
			base := ExtractBaseType(f.Type)
			if nested, ok := resolveInputType(inputTypes, endpoint, base); ok {
				childIndent := indent + tui.KeySpace + tui.KeySpace
				if i < len(it.Fields)-1 {
					childIndent = indent + tui.HelpStyle.Render(verticalDivider) + tui.KeySpace
				}
				lines = appendInputTypeFields(
					lines,
					nested,
					childIndent,
					inputTypes,
					endpoint,
					depth+1,
				)
			}
		}
	}
	return lines
}

func (m Model) renderLeftContent() string {
	badges := m.badgeCache
	search := tui.BorderStyle.
		Padding(0, 1).
		Width(m.leftPanelWidth() - 2).
		Render(m.search.Model.View())

	content := "\n" + tui.KeySpace
	if m.pickingEndpoints {
		content += endpointPickerTitle
	} else {
		content += fmt.Sprintf(operationFormat, len(m.filtered), len(m.operations))
	}
	statusLine := tui.HelpStyle.Render(content)

	var list string
	if m.ready {
		list = m.viewport.View()
	} else {
		content, _ := m.renderList()
		list = content
	}

	var helpText string
	if m.pickingEndpoints {
		helpText = tui.HelpStyle.Render(helpEndpointPicker)
	} else {
		helpText = tui.HelpStyle.Render(helpNavigation)
	}

	scrollPct := tui.HelpStyle.Render(
		fmt.Sprintf(" %3.f%%", m.viewport.ScrollPercent()*100),
	)

	var header string
	if badges != "" {
		header += badges + "\n"
	}
	header += search
	if m.filterHint != "" {
		header += "\n" + m.filterHint
	}
	return fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s  %s",
		header, statusLine, list, helpText, scrollPct,
	)
}

func renderHorizontalDivider(width int) string {
	style := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	return style.Render(strings.Repeat(horizontalDivider, max(width, 1)))
}

func renderDivider(height int) string {
	style := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	line := style.Render(" " + verticalDivider + " ")
	lines := make([]string, max(height, 1))
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func shortenEndpoint(rawURL string) string {
	if strings.Contains(rawURL, "://") {
		if parsed, err := url.Parse(rawURL); err == nil && parsed.Host != "" {
			path := parsed.Path
			path = strings.TrimSuffix(path, "/graphql")
			path = strings.TrimSuffix(path, "/gql")
			path = strings.TrimSuffix(path, "/")
			if path == "" {
				return parsed.Host
			}
			if strings.HasPrefix(path, "/") {
				return parsed.Host + path
			}
			return parsed.Host + "/" + path
		}
	}
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if q := strings.IndexByte(rawURL, '?'); q >= 0 {
		rawURL = rawURL[:q]
	}
	if h := strings.IndexByte(rawURL, '#'); h >= 0 {
		rawURL = rawURL[:h]
	}
	rawURL = strings.TrimSuffix(rawURL, "/graphql")
	rawURL = strings.TrimSuffix(rawURL, "/gql")
	rawURL = strings.TrimSuffix(rawURL, "/")
	return rawURL
}
