package gqlexplorer

import (
	"maps"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

const (
	helpEndpointPicker  = " k↑/j↓: navigate | space: toggle | enter: confirm | esc: cancel"
	checkMark           = utils.CheckMark
	endpointPickerTitle = "Filter Endpoints:"
)

func collectEndpoints(operations []UnifiedOperation) []string {
	seen := make(map[string]bool)
	var endpoints []string
	for _, op := range operations {
		shortEndpoint := op.EndpointShort
		if shortEndpoint == "" {
			shortEndpoint = shortenEndpoint(op.Endpoint)
		}
		if shortEndpoint != "" && !seen[shortEndpoint] {
			seen[shortEndpoint] = true
			endpoints = append(endpoints, shortEndpoint)
		}
	}
	sort.Strings(endpoints)
	return endpoints
}

func buildFilterHint(operations []UnifiedOperation, endpoints []string) string {
	hasType := make(map[OperationType]bool)
	for _, op := range operations {
		hasType[op.Type] = true
	}
	var parts []string
	if len(hasType) >= 2 {
		if hasType[TypeQuery] {
			parts = append(parts, "q: queries")
		}
		if hasType[TypeMutation] {
			parts = append(parts, "m: mutations")
		}
		if hasType[TypeSubscription] {
			parts = append(parts, "s: subscriptions")
		}
	}
	if len(endpoints) > 1 {
		parts = append(parts, "e: endpoints")
	}
	if len(parts) == 0 {
		return ""
	}
	return tui.HelpStyle.Render(tui.KeySpace + strings.Join(parts, " | "))
}

func (m *Model) applyFilter() {
	query := m.search.Model.Value()
	hasEndpointFilter := len(m.activeEndpoints) > 0

	if query == "" && !hasEndpointFilter {
		m.filtered = m.operations
		m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
		return
	}

	var typeFilter OperationType
	searchTerm := strings.ToLower(query)

	if len(query) >= 2 && query[1] == ':' {
		switch query[0] {
		case 'q', 'Q':
			typeFilter = TypeQuery
		case 'm', 'M':
			typeFilter = TypeMutation
		case 's', 'S':
			typeFilter = TypeSubscription
		}
		if typeFilter != "" {
			searchTerm = strings.ToLower(strings.TrimSpace(query[2:]))
		}
	}

	m.filtered = nil
	for _, op := range m.operations {
		if typeFilter != "" && op.Type != typeFilter {
			continue
		}
		if hasEndpointFilter && !m.activeEndpoints[op.EndpointShort] {
			continue
		}
		if searchTerm == "" || strings.Contains(op.NameLower, searchTerm) {
			m.filtered = append(m.filtered, op)
		}
	}
	m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
}

func (m *Model) shouldEnterEndpointPicker(value string) bool {
	return len(m.endpoints) > 1 && len(value) >= 2 &&
		(value[len(value)-2] == 'e' || value[len(value)-2] == 'E') &&
		value[len(value)-1] == ':'
}

func (m *Model) enterEndpointPicker() {
	m.pickingEndpoints = true
	m.endpointCursor = 0
	m.pendingEndpoints = make(map[string]bool)
	maps.Copy(m.pendingEndpoints, m.activeEndpoints)
	m.syncViewport()
}

func (m Model) handleEndpointPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case tui.KeyQuit:
		return m, tea.Quit
	case tui.KeyCancel:
		m.pickingEndpoints = false
		m.pendingEndpoints = nil
		m.stripEndpointPrefix()
		m.syncViewport()
		return m, nil
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyK:
		m.endpointCursor = tui.MoveCursorUp(m.endpointCursor)
		m.syncViewport()
		return m, nil
	case tui.KeyDown, tui.KeyCtrlN, tui.KeyJ:
		m.endpointCursor = tui.MoveCursorDown(m.endpointCursor, len(m.endpoints)-1)
		m.syncViewport()
		return m, nil
	case tui.KeySpace:
		ep := m.endpoints[m.endpointCursor]
		m.pendingEndpoints[ep] = !m.pendingEndpoints[ep]
		if !m.pendingEndpoints[ep] {
			delete(m.pendingEndpoints, ep)
		}
		m.syncViewport()
		return m, nil
	case tui.KeyEnter:
		m.activeEndpoints = make(map[string]bool)
		for k, v := range m.pendingEndpoints {
			if v {
				m.activeEndpoints[k] = true
			}
		}
		m.pickingEndpoints = false
		m.pendingEndpoints = nil
		m.stripEndpointPrefix()
		m.applyFilter()
		m.viewport.GotoTop()
		m.syncViewport()
		return m, nil
	}
	return m, nil
}

func (m *Model) stripEndpointPrefix() {
	val := m.search.Model.Value()
	for {
		idx := strings.LastIndex(strings.ToLower(val), "e:")
		if idx < 0 {
			break
		}
		val = strings.TrimSpace(val[:idx])
	}
	m.search.Model.SetValue(val)
}
