package gqlexplorer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

var vimToArrowMap = map[string]tea.KeyType{
	tui.KeyJ: tea.KeyDown,
	tui.KeyK: tea.KeyUp,
	tui.KeyH: tea.KeyLeft,
	tui.KeyL: tea.KeyRight,
}

func vimToArrow(msg tea.KeyMsg) tea.KeyMsg {
	if arrow, ok := vimToArrowMap[msg.String()]; ok {
		return tea.KeyMsg{Type: arrow}
	}
	return msg
}

func (m *Model) forwardKeyToForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd := m.detailForm.HandleKey(msg)
	m.syncViewport()
	return m, cmd
}

func (m *Model) handleDetailFormNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.detailForm.hasExpandedDropdown() {
		return m.forwardKeyToForm(msg)
	}
	switch msg.String() {
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyK:
		m.detailForm.CursorUp()
	case tui.KeyDown, tui.KeyCtrlN, tui.KeyJ:
		m.detailForm.CursorDown()
	case tui.KeyLeft, tui.KeyRight:
		cmd := m.detailPanel.Update(msg)
		return m, cmd
	}
	m.syncViewport()
	return m, nil
}

func (m *Model) jumpToEdge(top bool) {
	switch {
	case m.focus.IsFocused(m.queryPanel):
		if top {
			m.queryPanel.GotoTop()
		} else {
			m.queryPanel.GotoBottom()
		}
	case m.focus.IsFocused(m.variablePanel):
		if top {
			m.variablePanel.GotoTop()
		} else {
			m.variablePanel.GotoBottom()
		}
	case m.focus.IsFocused(m.responsePanel):
		if top {
			m.responsePanel.GotoTop()
		} else {
			m.responsePanel.GotoBottom()
		}
	case m.focus.IsFocused(m.detailPanel) && m.detailForm != nil:
		if top {
			m.detailForm.CursorToTop()
		} else {
			m.detailForm.CursorToBottom()
		}
		m.syncViewport()
	case m.focus.LeftFocused():
		if top {
			m.cursor = 0
		} else {
			m.cursor = max(len(m.filtered)-1, 0)
		}
		m.syncViewport()
	}
}

func (m *Model) switchPanel(key string) {
	if key == tui.KeyShiftTab {
		m.focus.Prev()
	} else {
		m.focus.Next()
	}
	if m.focus.LeftFocused() {
		m.focus.SetTyping(true)
	}
	m.syncSearchFocus()
	m.syncViewport()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.notification.Visible() {
		switch msg.String() {
		case tui.KeyAt, tui.KeyCancel, "q":
			if _, handled := m.actionRow.HandleKey(msg.String()); handled {
				cmd := m.handleBottomAction("badge")
				return m, cmd
			}
			if m.notification.ToggleLast() {
				m.updateActionRow()
			}
			return m, nil
		case tui.KeyYank:
			if text := m.notification.CopyText(); text != "" {
				return m, tui.CopyToClipboard(text)
			}
			return m, nil
		default:
			return m, nil
		}
	}
	if msg.String() == tui.KeyRefresh {
		cmd := m.startRefresh()
		return m, cmd
	}
	if msg.String() == tui.KeySend {
		cmd := m.executeQuery()
		return m, cmd
	}
	if msg.String() == tui.KeySave && m.responseBody != "" && m.focus.IsFocused(m.responsePanel) {
		cmd := m.saveResponse()
		return m, cmd
	}
	if msg.String() == tui.KeySaveQuery {
		cmd := m.saveQueryAndVariables()
		return m, cmd
	}
	if msg.String() == tui.KeyCreateRequest {
		cmd := m.createHulakRequestFile()
		return m, cmd
	}
	if msg.String() == tui.KeyAt && !m.search.Model.Focused() &&
		(m.detailForm == nil || !m.detailForm.ConsumesTextInput()) {
		if _, handled := m.actionRow.HandleKey(msg.String()); handled {
			cmd := m.handleBottomAction("badge")
			return m, cmd
		}
	}

	if m.pendingG {
		m.pendingG = false
		if msg.String() == tui.KeyG {
			m.jumpToEdge(true)
			return m, nil
		}
	}

	if m.focus.LeftFocused() && m.isEndpointMode() {
		if m.handleEndpointKey(msg) {
			return m, nil
		}
	}

	if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil && m.detailForm.IsSearching() {
		if msg.String() == tui.KeyQuit {
			return m, tea.Quit
		}
		cmd := m.detailForm.HandleSearchKey(msg)
		m.syncViewport()
		return m, cmd
	}

	if m.focus.IsFocused(m.responsePanel) && m.responseSearch.Active() {
		if msg.String() == tui.KeyQuit {
			return m, tea.Quit
		}
		cmd := m.handleResponseSearchKey(msg)
		return m, cmd
	}

	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit

		// Esc: step backward one panel at a time
		// variable panel → query panel → detail panel → search (left) → quit
	case tui.KeyCancel:
		// Detail panel: close dropdown first, exit text editing, then step back.
		if m.focus.IsFocused(m.detailPanel) {
			if m.detailForm != nil && m.detailForm.hasExpandedDropdown() {
				m.detailForm.HandleKey(msg)
				m.syncViewport()
				return m, nil
			}
			if m.detailForm != nil && m.detailForm.ConsumesTextInput() {
				m.detailForm.HandleKey(msg)
				m.syncViewport()
				return m, nil
			}
			if m.detailForm != nil {
				m.detailForm.BlurAll()
			}
			m.focus.FocusByNumber(1)
			m.focus.SetTyping(true)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Response panel: step back to variable panel.
		if m.focus.IsFocused(m.responsePanel) {
			m.focus.FocusByNumber(m.variablePanel.Number)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Variable panel: step back to query panel.
		if m.focus.IsFocused(m.variablePanel) {
			m.focus.FocusByNumber(m.queryPanel.Number)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Query panel: step back to detail panel.
		if m.focus.IsFocused(m.queryPanel) {
			m.focus.FocusByNumber(m.detailPanel.Number)
			m.syncSearchFocus()
			m.syncViewport()
			return m, nil
		}
		// Left panel (search): clear text → blur → quit.
		if m.focus.Typing() {
			if m.search.Model.Value() != "" {
				m.search.Model.Reset()
				m.applyFilterAndReset()
				return m, nil
			}
			m.focus.SetTyping(false)
			m.syncSearchFocus()
			return m, nil
		}
		return m, tea.Quit

	// ── Tab / Shift+Tab: cycle panels ───────────────────────────
	case tui.KeyTab, tui.KeyShiftTab:
		m.switchPanel(msg.String())
		return m, nil
	// ── Enter: detail panel form input / left panel → detail ────
	case tui.KeyEnter:
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil {
			return m.forwardKeyToForm(msg)
		}
		if m.focus.LeftFocused() {
			if !m.focus.Typing() {
				m.focus.SetTyping(true)
				m.syncSearchFocus()
				return m, nil
			}
			if m.hasTwoPanelLayout() {
				m.focus.FocusByNumber(m.detailPanel.Number)
				m.syncSearchFocus()
				m.syncViewport()
			}
		}
		return m, nil

	// ── Arrow / vim keys: per-panel navigation ──────────────────
	// Query panel: scroll viewport (j/k vertical, h/l horizontal).
	// Detail panel: navigate form items or scroll.
	// Left panel: move operation cursor.
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyDown, tui.KeyCtrlN, tui.KeyLeft, tui.KeyRight,
		tui.KeyK, tui.KeyJ, tui.KeyH, tui.KeyL, tui.KeyG, tui.KeyShiftG:
		if (msg.String() == tui.KeyLeft || msg.String() == tui.KeyRight) &&
			m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
			m.detailForm.ConsumesTextInput() {
			return m.forwardKeyToForm(msg)
		}
		if (msg.String() == tui.KeyLeft || msg.String() == tui.KeyRight) &&
			m.focus.LeftFocused() && m.focus.Typing() {
			var cmd tea.Cmd
			_, cmd = m.search.Update(msg)
			return m, cmd
		}
		if msg.String() == tui.KeyJ || msg.String() == tui.KeyK ||
			msg.String() == tui.KeyH || msg.String() == tui.KeyL ||
			msg.String() == tui.KeyG || msg.String() == tui.KeyShiftG {
			if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
				m.detailForm.ConsumesTextInput() {
				return m.forwardKeyToForm(msg)
			}
			if m.focus.LeftFocused() && m.focus.Typing() {
				break
			}
		}
		if msg.String() == tui.KeyShiftG {
			m.jumpToEdge(false)
			return m, nil
		}
		if msg.String() == tui.KeyG {
			m.pendingG = true
			return m, nil
		}
		// Query panel: scroll viewport. Vim keys are mapped to arrows
		// because the bubbles viewport only understands arrow key types.
		if m.focus.IsFocused(m.queryPanel) {
			cmd := m.queryPanel.Update(vimToArrow(msg))
			return m, cmd
		}
		// Variable panel: scroll viewport. Vim keys are mapped to arrows
		// because the bubbles viewport only understands arrow key types.
		if m.focus.IsFocused(m.variablePanel) {
			cmd := m.variablePanel.Update(vimToArrow(msg))
			return m, cmd
		}
		if m.focus.IsFocused(m.responsePanel) {
			cmd := m.responsePanel.Update(vimToArrow(msg))
			return m, cmd
		}
		// Detail panel: navigate form or scroll.
		if !m.focus.LeftFocused() {
			if m.detailForm != nil {
				return m.handleDetailFormNavigation(msg)
			}
			cmd := m.detailPanel.Update(msg)
			return m, cmd
		}
		// Left panel: move operation list cursor.
		switch msg.String() {
		case tui.KeyUp, tui.KeyCtrlP, tui.KeyK:
			m.cursor = tui.MoveCursorUp(m.cursor)
			m.syncViewport()
		case tui.KeyDown, tui.KeyCtrlN, tui.KeyJ:
			m.cursor = tui.MoveCursorDown(m.cursor, len(m.filtered)-1)
			m.syncViewport()
		}
		return m, nil

	// Space: detail panel field toggle
	case tui.KeySpace:
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil {
			return m.forwardKeyToForm(msg)
		}

	// Slash: vim-style search in detail form
	case tui.KeySlash:
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
			!m.detailForm.ConsumesTextInput() {
			m.detailForm.StartSearch()
			m.syncViewport()
			return m, nil
		}
		if m.focus.IsFocused(m.responsePanel) && m.responseBody != "" {
			m.responseSearch.Start()
			m.syncResponseFooter()
			return m, nil
		}

	// Yank: copy focused panel content to system clipboard
	case tui.KeyYank:
		if text := m.notification.CopyText(); text != "" {
			return m, tui.CopyToClipboard(text)
		}
		if text := m.yankText(); text != "" {
			return m, tui.CopyToClipboard(text)
		}
		return m, nil
	}

	if !m.focus.Typing() && (m.detailForm == nil || !m.detailForm.ConsumesTextInput()) {
		if key := msg.String(); len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			num := int(key[0] - '0')
			if m.focus.FocusByNumber(num) {
				if m.focus.LeftFocused() {
					m.focus.SetTyping(true)
				}
				m.syncSearchFocus()
				m.syncViewport()
			}
			return m, nil
		}
	}

	if !m.focus.LeftFocused() {
		if m.focus.IsFocused(m.detailPanel) && m.detailForm != nil &&
			m.detailForm.ConsumesTextInput() {
			return m.forwardKeyToForm(msg)
		}
		return m, nil
	}

	prevValue := m.search.Model.Value()
	var cmd tea.Cmd
	_, cmd = m.search.Update(msg)
	newValue := m.search.Model.Value()
	if newValue != prevValue {
		if m.isEndpointMode() {
			m.endpointCursor = 0
		}
		m.applyFilterAndReset()
	}
	return m, cmd
}
