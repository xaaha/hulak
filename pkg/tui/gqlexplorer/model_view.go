package gqlexplorer

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/xaaha/hulak/pkg/tui"
)

func (m *Model) yankText() string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	op := &m.filtered[m.cursor]
	switch {
	case m.focus.IsFocused(m.queryPanel):
		return BuildQueryString(op, m.detailForm)
	case m.focus.IsFocused(m.variablePanel):
		return BuildVariablesString(op, m.detailForm)
	case m.focus.IsFocused(m.responsePanel):
		return m.responseBody
	case m.focus.IsFocused(m.detailPanel):
		return m.detailPanelPlainText(op)
	case m.focus.LeftFocused():
		return formatOperationSummary(op)
	}
	return ""
}

func (m *Model) detailPanelPlainText(op *UnifiedOperation) string {
	var styled string
	if m.detailForm != nil {
		styled, _ = m.detailForm.View(op)
	} else {
		styled = renderDetail(op, m.inputTypes, m.objectTypes, m.unionTypes, m.interfaceTypes)
	}
	return ansi.Strip(styled)
}

func formatOperationSummary(op *UnifiedOperation) string {
	var b strings.Builder
	b.WriteString(op.Name)
	if op.Description != "" {
		b.WriteString("\n  ")
		b.WriteString(op.Description)
	}
	if op.Endpoint != "" {
		b.WriteString("\n  ")
		b.WriteString(op.Endpoint)
	}
	return b.String()
}

func (m *Model) applyFilterAndReset() {
	m.applyFilter()
	m.viewport.GotoTop()
	m.syncViewport()
}

func (m *Model) syncViewport() {
	var content string
	var cursorLine int
	if m.isEndpointMode() {
		content, cursorLine = m.renderEndpointPicker()
	} else {
		content, cursorLine = m.renderList()
	}
	tui.SyncViewport(&m.viewport, content, cursorLine, tui.DefaultScrollMargin)

	if m.isEndpointMode() {
		return
	}

	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		op := &m.filtered[m.cursor]

		formKey := op.Endpoint + "\x1f" + op.Name
		if m.detailFormKey != formKey {
			if m.detailForm != nil && m.detailFormKey != "" {
				m.formCache[m.detailFormKey] = m.detailForm
			}
			if !m.executing {
				m.saveResponseToCache()
			}
			if cached, ok := m.formCache[formKey]; ok {
				m.detailForm = cached
			} else {
				m.detailForm = buildDetailForm(op, m.inputTypes, m.enumTypes, m.objectTypes, m.unionTypes, m.interfaceTypes)
			}
			m.detailFormKey = formKey
			m.detailPanel.GotoTop()
			if !m.executing {
				m.restoreResponseFromCache(formKey)
			}
		}

		if m.detailForm != nil {
			if m.focus.IsFocused(m.detailPanel) {
				m.detailForm.FocusCurrent()
			} else {
				m.detailForm.BlurAll()
			}
			content, cursorLine := m.detailForm.ViewMarked(op, m.detailMousePrefix(), m.mouse.Mark)
			m.detailPanel.SyncContent(content, cursorLine)
			m.detailPanel.Footer = m.detailForm.SearchFooter()
		} else {
			cacheKey := op.Endpoint + "\x1f" + op.Name + "\x1f" + strconv.Itoa(m.rightPanelWidth())
			if m.detailPanel.SetContent(renderDetail(op, m.inputTypes, m.objectTypes, m.unionTypes, m.interfaceTypes), cacheKey) {
				m.detailPanel.GotoTop()
			}
		}

		query := BuildQueryString(op, m.detailForm)
		variables := BuildVariablesString(op, m.detailForm)
		m.queryPanel.SetContent(formatQueryForPanel(query, m.focus.IsFocused(m.queryPanel)), "")
		m.variablePanel.SetContent(
			formatVariablesForPanel(variables, m.focus.IsFocused(m.variablePanel)),
			"",
		)
		if m.responseBody != "" {
			m.setResponseContent()
		}
	} else {
		m.detailForm = nil
		m.detailFormKey = ""
		m.detailPanel.Footer = ""
		m.detailPanel.SetContent("", "")
		m.queryPanel.SetContent("", "")
		m.variablePanel.SetContent("", "")
		m.responsePanel.SetHeader("")
		m.responsePanel.SetContent("", "")
	}
}

func (m *Model) renderHelpBar(width int) string {
	var raw string
	switch {
	case m.focus.LeftFocused() && m.isEndpointMode():
		raw = helpEndpointFilter
	case m.focus.IsFocused(m.queryPanel):
		raw = helpQueryPanel
	case m.focus.IsFocused(m.variablePanel):
		raw = helpVariablePanel
	case m.focus.IsFocused(m.responsePanel) && m.responseSearch.Active():
		raw = helpSearchPanel
	case m.focus.IsFocused(m.responsePanel):
		raw = helpResponsePanel
	case m.focus.IsFocused(m.detailPanel) && m.detailForm != nil && m.detailForm.IsSearching():
		raw = helpSearchPanel
	case m.focus.IsFocused(m.detailPanel):
		raw = helpDetailPanel
	default:
		raw = helpLeftPanel
	}
	return tui.HelpBarStyle.Render(
		lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(raw),
	)
}

func (m *Model) View() string {
	leftW := m.leftPanelWidth()
	contentW := m.contentWidth()
	contentH := m.contentHeight()

	helpBar := m.renderHelpBar(contentW)

	leftCol := lipgloss.NewStyle().
		Width(leftW).
		Height(contentH).
		Render(m.renderLeftContent())
	if !m.hasTwoPanelLayout() {
		body := lipgloss.JoinVertical(lipgloss.Left, leftCol, helpBar)
		box := _containerStyle.
			Width(max(m.width-_containerStyle.GetHorizontalFrameSize(), 1)).
			Height(max(m.height-_containerStyle.GetVerticalFrameSize(), 1)).
			Render(body)

		return tui.ScanMouseZones(
			lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box),
		)
	}

	rightW := m.rightPanelWidth()

	var detailView, queryView, variableView, responseView string
	if m.detailPanel.CanRender() {
		detailView = m.detailPanel.View(m.focus.IsFocused(m.detailPanel))
	}
	if m.queryPanel.CanRender() {
		queryView = m.queryPanel.View(m.focus.IsFocused(m.queryPanel))
	}
	if m.variablePanel.CanRender() {
		variableView = m.variablePanel.View(m.focus.IsFocused(m.variablePanel))
	}
	if m.responsePanel.CanRender() {
		responseView = m.responsePanel.View(m.focus.IsFocused(m.responsePanel))
	}

	topRight := lipgloss.JoinHorizontal(lipgloss.Top, detailView, queryView)
	detailW := m.detailPanelWidth(rightW)
	queryW := max(rightW-detailW, 1)

	actionsView := lipgloss.NewStyle().
		Width(queryW).
		Height(m.callAreaHeight()).
		Render(m.renderActionsPanel(queryW, m.callAreaHeight()))
	rightBottom := lipgloss.JoinVertical(lipgloss.Left, variableView, actionsView)
	bottomSection := lipgloss.JoinHorizontal(lipgloss.Top, responseView, rightBottom)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRight, bottomSection)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
	body := lipgloss.JoinVertical(lipgloss.Left, combined, helpBar)

	boxH := max(m.height-_containerStyle.GetVerticalFrameSize(), 1)

	box := _containerStyle.
		Height(boxH).
		Render(body)

	if m.notification.Visible() {
		box = tui.OverlayCenter(
			m.notification.RenderModal(max(m.width-8, 1), max(m.height-6, 1)),
			m.width,
			m.height,
		)
	}

	return tui.ScanMouseZones(lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, box))
}
