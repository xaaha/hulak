package gqlexplorer

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

func (m Model) renderList() (string, int) {
	itemPrefix := strings.Repeat(" ", itemPadding)
	detailPrefix := strings.Repeat(" ", detailPadding)
	selectedPrefix := strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + utils.CursorMarker

	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render(
			strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + noMatchesLabel,
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
			if op.Description != "" {
				lines = append(lines, tui.HelpStyle.Render(detailPrefix+op.Description))
			}
			if op.Endpoint != "" {
				lines = append(lines, tui.HelpStyle.Render(detailPrefix+op.Endpoint))
			}
		} else {
			lines = append(lines, itemPrefix+op.Name)
		}
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m Model) renderEndpointPicker() (string, int) {
	itemPrefix := strings.Repeat(" ", itemPadding)
	selectedPrefix := strings.Repeat(" ", itemPadding-len(utils.CursorMarker)) + utils.CursorMarker

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
		toggle := "  "
		if m.pendingEndpoints[ep] {
			toggle = checkMark + " "
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
	var badges []string
	for ep := range m.activeEndpoints {
		badges = append(badges, tui.RenderBadge(shortenEndpoint(ep), tui.ColorPrimary))
	}
	sort.Strings(badges)
	return strings.Join(badges, " ")
}

func shortenEndpoint(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, "/graphql")
	url = strings.TrimSuffix(url, "/gql")
	url = strings.TrimSuffix(url, "/")
	return url
}
