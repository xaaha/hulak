package gqlexplorer

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

const (
	helpEndpointFilter = "↑↓ Ctrl+n/p | Enter: toggle | !term: keep only matches | Esc: back"
)

func collectEndpoints(operations []UnifiedOperation) []string {
	seen := make(map[string]bool)
	var endpoints []string
	for i := range operations {
		ep := operations[i].EndpointShort // populated by NewModel before this is called
		if ep != "" && !seen[ep] {
			seen[ep] = true
			endpoints = append(endpoints, ep)
		}
	}
	sort.Strings(endpoints)
	return endpoints
}

func buildFilterHint(operations []UnifiedOperation, endpoints []string) string {
	hasType := make(map[OperationType]bool)
	for i := range operations {
		hasType[operations[i].Type] = true
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
	return tui.HelpStyle.Render(strings.Join(parts, " | "))
}

// isEndpointMode returns true when the search input contains the e: prefix,
// indicating that the left panel should show a toggleable endpoint list
// instead of the normal operation list. Requires multiple endpoints.
func (m *Model) isEndpointMode() bool {
	if len(m.endpoints) <= 1 {
		return false
	}
	val := strings.ToLower(m.search.Model.Value())
	// e: can appear standalone or after another prefix like q:e:
	return strings.Contains(val, "e:")
}

// endpointSearchTerm extracts the text typed after the e: prefix.
// This text is used to narrow the endpoint list. Returns lowercase.
func (m *Model) endpointSearchTerm() string {
	val := strings.ToLower(m.search.Model.Value())
	idx := strings.LastIndex(val, "e:")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(val[idx+2:])
}

func (m *Model) isNegatedEndpointSearch() bool {
	return strings.HasPrefix(m.endpointSearchTerm(), "!")
}

// filteredEndpoints returns the subset of endpoints matching the
// search term typed after e:. A leading ! is stripped before matching.
func (m *Model) filteredEndpoints() []string {
	term := m.endpointSearchTerm()
	term = strings.TrimPrefix(term, "!")
	if term == "" {
		return m.endpoints
	}
	var result []string
	for _, ep := range m.endpoints {
		if strings.Contains(strings.ToLower(ep), term) {
			result = append(result, ep)
		}
	}
	return result
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
	for i := range m.operations {
		op := &m.operations[i]
		if typeFilter != "" && op.Type != typeFilter {
			continue
		}
		if hasEndpointFilter && !m.activeEndpoints[op.EndpointShort] {
			continue
		}
		if searchTerm == "" || strings.Contains(op.NameLower, searchTerm) {
			m.filtered = append(m.filtered, *op)
		}
	}
	m.cursor = tui.ClampCursor(m.cursor, len(m.filtered)-1)
}

// handleEndpointKey processes keys when the left panel is in endpoint
// mode (e: prefix active). Returns true if the key was consumed.
func (m *Model) handleEndpointKey(msg tea.KeyMsg) bool {
	eps := m.filteredEndpoints()
	if len(eps) == 0 {
		return false
	}

	switch msg.String() {
	case tui.KeyUp, tui.KeyCtrlP:
		m.endpointCursor = tui.MoveCursorUp(m.endpointCursor)
		m.syncViewport()
		return true
	case tui.KeyDown, tui.KeyCtrlN:
		m.endpointCursor = tui.MoveCursorDown(m.endpointCursor, len(eps)-1)
		m.syncViewport()
		return true
	case tui.KeyEnter:
		if m.isNegatedEndpointSearch() {
			keep := make(map[string]bool, len(eps))
			for _, ep := range eps {
				keep[ep] = true
			}
			m.activeEndpoints = keep
		} else {
			ep := eps[m.endpointCursor]
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
