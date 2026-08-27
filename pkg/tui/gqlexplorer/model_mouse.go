package gqlexplorer

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.notification.Visible() {
		return m, nil
	}
	if tui.IsLeftClick(msg) {
		if cmd, ok := m.handleBottomRowClick(msg); ok {
			return m, cmd
		}
		if m.responseBody != "" && tui.Hit(m.saveZoneID(), msg) {
			cmd := m.saveResponse()
			return m, cmd
		}
		if m.handleLeftPanelClick(msg) {
			return m, nil
		}
		if m.handleDetailFormClick(msg) {
			return m, nil
		}
	}

	var cmds []tea.Cmd
	_, cmd := m.search.Update(msg)
	cmds = append(cmds, cmd)
	cmd = m.updateFocusedViewport(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleBottomRowClick(msg tea.MouseMsg) (tea.Cmd, bool) {
	id, ok := m.actionRow.HandleMouse(msg)
	if !ok {
		return nil, false
	}
	return m.handleBottomAction(id), true
}

func (m *Model) handleLeftPanelClick(msg tea.MouseMsg) bool {
	if !m.ready {
		return false
	}

	if tui.Hit(m.searchZoneID(), msg) {
		m.focus.FocusByNumber(1)
		m.focus.SetTyping(true)
		m.syncSearchFocus()
		m.syncViewport()
		return true
	}

	if m.isEndpointMode() {
		eps := m.filteredEndpoints()
		for i := range eps {
			if !tui.Hit(m.endpointZoneID(i), msg) {
				continue
			}
			m.focus.FocusByNumber(1)
			m.focus.SetTyping(false)
			m.endpointCursor = i
			m.syncSearchFocus()
			if m.isNegatedEndpointSearch() {
				keep := make(map[string]bool, len(eps))
				for _, ep := range eps {
					keep[ep] = true
				}
				m.activeEndpoints = keep
			} else {
				ep := eps[i]
				if m.activeEndpoints[ep] {
					delete(m.activeEndpoints, ep)
				} else {
					m.activeEndpoints[ep] = true
				}
			}
			m.updateBadgeCache()
			m.applyFilter()
			m.syncViewport()
			return true
		}
		return false
	}

	for i := range m.filtered {
		if !tui.Hit(m.operationZoneID(i), msg) {
			continue
		}
		m.focus.FocusByNumber(1)
		m.focus.SetTyping(false)
		m.syncSearchFocus()
		m.cursor = i
		m.syncViewport()
		return true
	}
	return false
}

func (m *Model) handleDetailFormClick(msg tea.MouseMsg) bool {
	if m.detailForm == nil || m.isEndpointMode() {
		return false
	}
	if !m.detailForm.HandleMouse(m.detailMousePrefix(), msg) {
		return false
	}
	m.focus.FocusByNumber(m.detailPanel.Number)
	m.syncSearchFocus()
	m.syncViewport()
	return true
}

func (m *Model) operationZoneID(index int) string {
	return m.mouse.ID("operation", strconv.Itoa(index))
}

func (m *Model) endpointZoneID(index int) string {
	return m.mouse.ID("endpoint", strconv.Itoa(index))
}

func (m *Model) detailMousePrefix() string {
	return m.mouse.ID("detail")
}

func (m *Model) searchZoneID() string {
	return m.mouse.ID("search")
}
