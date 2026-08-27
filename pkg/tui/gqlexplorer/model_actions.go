package gqlexplorer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

func (m *Model) handleBottomAction(id string) tea.Cmd {
	switch id {
	case "badge":
		if m.notification.ToggleLast() {
			m.updateActionRow()
		}
		return nil
	case "refresh":
		return m.startRefresh()
	case "send":
		return m.executeQuery()
	case "saveQuery":
		return m.saveQueryAndVariables()
	case "createRequest":
		return m.createHulakRequestFile()
	default:
		return nil
	}
}

func (m *Model) enqueueNotification(severity tui.NotificationSeverity, message string) tea.Cmd {
	cmd := m.notification.Enqueue(severity, message)
	m.updateActionRow()
	return cmd
}

func (m *Model) canSend() bool {
	if m.executing {
		return false
	}
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return false
	}
	op := &m.filtered[m.cursor]
	if op.Type == TypeSubscription {
		return false
	}
	_, ok := m.apiInfos[op.Endpoint]
	return ok
}

func (m *Model) canSaveOrCreate() bool {
	return len(m.filtered) > 0 && m.cursor < len(m.filtered)
}

func (m *Model) updateActionRow() {
	items := []tui.ActionItem{
		{
			ID:      "refresh",
			Label:   "Refresh         ctrl+r",
			Key:     tui.KeyRefresh,
			Enabled: m.refreshFn != nil && !m.refreshing,
		},
		{
			ID:      "send",
			Label:   "Send            ctrl+o",
			Key:     tui.KeySend,
			Enabled: m.canSend(),
		},
		{
			ID:      "saveQuery",
			Label:   "Save Query      ctrl+q",
			Key:     tui.KeySaveQuery,
			Enabled: m.canSaveOrCreate(),
		},
		{
			ID:      "createRequest",
			Label:   "Save Request    ctrl+x",
			Key:     tui.KeyCreateRequest,
			Enabled: m.canSaveOrCreate(),
		},
	}
	m.actionRow.SetItems(items)
	m.actionRow.SetBadge(tui.ActionBadge{
		Label:    "Notification @",
		Key:      tui.KeyAt,
		Severity: m.notification.Severity(),
		Visible:  m.notification.HasLast(),
	})
}
