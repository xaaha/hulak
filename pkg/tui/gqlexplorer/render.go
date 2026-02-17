package gqlexplorer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

func (m Model) renderList() (string, int) {
	itemPrefix := strings.Repeat(tui.KeySpace, itemPadding)
	detailPrefix := strings.Repeat(tui.KeySpace, detailPadding)
	selectedPrefix := strings.Repeat(
		tui.KeySpace,
		itemPadding-len(utils.ChevronRight),
	) + utils.ChevronRight

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
			if op.Description != "" {
				wrapped := lipgloss.NewStyle().Width(wrapW).Render(op.Description)
				for line := range strings.SplitSeq(wrapped, "\n") {
					lines = append(lines, tui.HelpStyle.Render(detailPrefix+line))
				}
			}
			if op.Endpoint != "" {
				wrapped := lipgloss.NewStyle().Width(wrapW).Render(op.Endpoint)
				for line := range strings.SplitSeq(wrapped, "\n") {
					lines = append(lines, tui.HelpStyle.Render(detailPrefix+line))
				}
			}
		} else {
			lines = append(lines, itemPrefix+op.Name)
		}
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m Model) renderEndpointPicker() (string, int) {
	itemPrefix := strings.Repeat(tui.KeySpace, itemPadding)
	selectedPrefix := strings.Repeat(
		tui.KeySpace,
		itemPadding-len(utils.ChevronRight),
	) + utils.ChevronRight

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
		toggle := strings.Repeat(tui.KeySpace, 2)
		if m.pendingEndpoints[ep] {
			toggle = checkMark + tui.KeySpace
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
		shortened = append(shortened, shortenEndpoint(ep))
	}
	sort.Strings(shortened)

	maxW := m.leftPanelWidth()
	var result string
	for i, s := range shortened {
		badge := tui.RenderBadge(s, tui.ColorMuted)
		candidate := result
		if candidate != "" {
			candidate += tui.KeySpace
		}
		candidate += badge
		if lipgloss.Width(candidate) > maxW && i > 0 {
			result += tui.KeySpace + tui.HelpStyle.Render("…")
			break
		}
		result = candidate
	}
	return result
}

func renderDetail(
	op UnifiedOperation,
	width int,
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
			if it, ok := inputTypes[base]; ok {
				lines = appendInputTypeFields(lines, it, argPad+strings.Repeat(tui.KeySpace, 2), inputTypes, 1)
			}
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

const maxInputTypeDepth = 3

func appendInputTypeFields(
	lines []string,
	it graphql.InputType,
	indent string,
	inputTypes map[string]graphql.InputType,
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
			if nested, ok := inputTypes[base]; ok {
				childIndent := indent + tui.KeySpace + tui.KeySpace
				if i < len(it.Fields)-1 {
					childIndent = indent + tui.HelpStyle.Render("│") + tui.KeySpace
				}
				lines = appendInputTypeFields(lines, nested, childIndent, inputTypes, depth+1)
			}
		}
	}
	return lines
}

func (m Model) renderLeftContent() string {
	badges := m.renderBadges()
	search := tui.BorderStyle.
		Padding(0, 1).
		Width(m.leftPanelWidth() - 2).
		Render(m.search.Model.View())

	var statusLine string
	if m.pickingEndpoints {
		statusLine = tui.HelpStyle.Render(tui.KeySpace + endpointPickerTitle)
	} else {
		statusLine = tui.HelpStyle.Render(
			tui.KeySpace + fmt.Sprintf(operationFormat, len(m.filtered), len(m.operations)),
		)
	}

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

func renderDivider(height int) string {
	style := lipgloss.NewStyle().Foreground(tui.ColorMuted)
	line := style.Render(" │ ")
	lines := make([]string, max(height, 1))
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func shortenEndpoint(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, "/graphql")
	url = strings.TrimSuffix(url, "/gql")
	url = strings.TrimSuffix(url, "/")
	return url
}
