package gqlexplorer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
)

// Cached styles — these never change at runtime, so building them once
// at package init avoids repeated allocations per View() frame.
var _containerStyle = tui.BoxStyle.Padding(0, 1)

// hasHeaderContentSpace guards optional header UI that is visually noisy in
// narrow terminals (badge row + placeholder hint).
// When space is limited, returning false keeps the search row stable by hiding those extras.
func (m *Model) hasHeaderContentSpace() bool {
	return m.width >= minHeaderContentWidth
}

func (m *Model) updateSearchPlaceholder() {
	if m.hasHeaderContentSpace() {
		m.search.Model.Placeholder = searchPlaceholderText
		return
	}
	m.search.Model.Placeholder = ""
}

func (m *Model) leftPanelWidth() int {
	contentW := m.contentWidth()
	if !m.hasTwoPanelLayout() {
		return max(contentW, 1)
	}

	leftW := contentW * tui.LeftPanelPct / 100
	leftW = max(leftW, tui.MinLeftPanelWidth)
	maxLeft := max(contentW-tui.MinRightPanelWidth, 1)
	leftW = min(leftW, maxLeft)
	return max(leftW, 1)
}

func (m *Model) rightPanelWidth() int {
	if !m.hasTwoPanelLayout() {
		return 0
	}
	return max(m.contentWidth()-m.leftPanelWidth(), 0)
}

func (m *Model) hasTwoPanelLayout() bool {
	return m.contentWidth() >= tui.MinLeftPanelWidth+tui.MinRightPanelWidth
}

func (m *Model) contentWidth() int {
	return max(m.width-_containerStyle.GetHorizontalFrameSize(), 1)
}

func (m *Model) updateHelpBarHeight() {
	contentW := m.contentWidth()
	m.helpBarH = 1
	for _, h := range []string{
		helpLeftPanel, helpDetailPanel, helpSearchPanel,
		helpQueryPanel, helpVariablePanel, helpResponsePanel, helpEndpointFilter,
	} {
		rendered := tui.HelpBarStyle.Render(
			lipgloss.NewStyle().Width(contentW).Align(lipgloss.Center).Render(h),
		)
		if lines := lipgloss.Height(rendered); lines > m.helpBarH {
			m.helpBarH = lines
		}
	}
}

func (m *Model) contentHeight() int {
	return max(m.height-_containerStyle.GetVerticalFrameSize()-m.helpBarH, 1)
}

func (m *Model) detailTopHeight() int {
	return max(m.contentHeight()*tui.DetailTopPct/100, 1)
}

func (m *Model) detailPanelWidth(rightW int) int {
	return max(rightW/2, 1)
}

// gql variable panel height, 2/3 of the remaining height below the top row
func (m *Model) variablePanelHeight() int {
	remaining := max(m.contentHeight()-m.detailTopHeight(), 1)
	return max((remaining*2)/3, 1)
}

// callAreaHeight returns the remaining height allocated to extras
// below the variable panel. I am calling it callAreaheight
func (m *Model) callAreaHeight() int {
	return max(m.contentHeight()-m.detailTopHeight()-m.variablePanelHeight(), 1)
}

func (m *Model) updateBadgeCache() {
	if !m.hasHeaderContentSpace() {
		m.badgeCache = ""
		return
	}
	m.badgeCache = m.renderBadges()
}

func (m *Model) syncSearchFocus() {
	if m.focus.LeftFocused() && m.focus.Typing() {
		m.search.Model.Focus()
		return
	}
	m.search.Model.Blur()
}

func (m *Model) updateFocusedViewport(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.focus.LeftFocused() {
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}
	if m.focus.IsFocused(m.queryPanel) {
		return m.queryPanel.Update(msg)
	}
	if m.focus.IsFocused(m.variablePanel) {
		return m.variablePanel.Update(msg)
	}
	if m.focus.IsFocused(m.responsePanel) {
		return m.responsePanel.Update(msg)
	}
	return m.detailPanel.Update(msg)
}

func (m *Model) viewportHeight() int {
	panelW := max(m.leftPanelWidth(), 1)
	headerLines := tui.SearchBoxHeight
	// Only count the badge row when it will actually be rendered.
	// updateBadgeCache clears badgeCache in narrow terminals, so counting
	// it unconditionally causes a 1-line viewport height mismatch.
	if len(m.activeEndpoints) > 0 && m.hasHeaderContentSpace() {
		headerLines++
	}
	if m.filterHint != "" {
		headerLines += wrappedLineCount(m.filterHint, panelW)
	}
	footerLines := tui.StatusRowHeight
	h := max(m.contentHeight()-headerLines-footerLines, 1)
	return h
}

// wrappedLineCount returns how many visual lines text occupies at the given
// width. It performs a full lipgloss render internally, which is fine for
// short strings (help text, filter hint) but would be a concern for longer content.
func wrappedLineCount(text string, width int) int {
	if width <= 0 || text == "" {
		return 0
	}
	return lipgloss.Height(lipgloss.NewStyle().Width(width).Render(text))
}
