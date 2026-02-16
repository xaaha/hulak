package gqlexplorer

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

func (m Model) renderList() (string, int) {
	itemPrefix := strings.Repeat(tui.KeySpace, itemPadding)
	detailPrefix := strings.Repeat(tui.KeySpace, detailPadding)
	selectedPrefix := strings.Repeat(tui.KeySpace, itemPadding-len(utils.ChevronRight)) + utils.ChevronRight

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
				for _, line := range strings.Split(wrapped, "\n") {
					lines = append(lines, tui.HelpStyle.Render(detailPrefix+line))
				}
			}
			if op.Endpoint != "" {
				wrapped := lipgloss.NewStyle().Width(wrapW).Render(op.Endpoint)
				for _, line := range strings.Split(wrapped, "\n") {
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
	selectedPrefix := strings.Repeat(tui.KeySpace, itemPadding-len(utils.ChevronRight)) + utils.ChevronRight

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

func shortenEndpoint(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, "/graphql")
	url = strings.TrimSuffix(url, "/gql")
	url = strings.TrimSuffix(url, "/")
	return url
}
