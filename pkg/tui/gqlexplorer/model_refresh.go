package gqlexplorer

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

func (m *Model) SetRefresh(fn RefreshFunc) {
	m.refreshFn = fn
	m.updateActionRow()
}

func (m *Model) SetInitialWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	m.initCmd = m.enqueueNotification(tui.NotificationWarn, joinWarnings(warnings))
}

func (m *Model) startRefresh() tea.Cmd {
	if m.refreshFn == nil || m.refreshing {
		return nil
	}
	m.refreshing = true
	m.updateActionRow()
	refreshFn := m.refreshFn
	return func() tea.Msg {
		payload, err := refreshFn()
		return refreshLoadedMsg{payload: payload, err: err}
	}
}

func (m *Model) applyRefreshPayload(payload *RefreshPayload) {
	m.operations = payload.Data.Operations // assign new data first
	// normalize the new data
	for i := range m.operations {
		if m.operations[i].NameLower == "" {
			m.operations[i].NameLower = strings.ToLower(m.operations[i].Name)
		}
		if m.operations[i].EndpointShort == "" {
			m.operations[i].EndpointShort = shortenEndpoint(m.operations[i].Endpoint)
		}
	}
	sort.Slice(m.operations, func(i, j int) bool { // sort the new data
		return typeRank[m.operations[i].Type] < typeRank[m.operations[j].Type]
	})
	m.inputTypes = payload.Data.InputTypes
	m.enumTypes = payload.Data.EnumTypes
	m.objectTypes = payload.Data.ObjectTypes
	m.unionTypes = payload.Data.UnionTypes
	m.interfaceTypes = payload.Data.InterfaceTypes
	m.apiInfos = payload.Data.APIInfos
	m.schemaFilePaths = payload.Data.SchemaFilePaths
	m.endpoints = collectEndpoints(m.operations)
	m.activeEndpoints = make(map[string]bool, len(m.endpoints))
	for _, ep := range m.endpoints {
		m.activeEndpoints[ep] = true
	}
	m.filterHint = buildFilterHint(m.operations, m.endpoints)
	m.cursor = 0
	m.endpointCursor = 0
	m.formCache = make(map[string]*DetailForm)
	m.responseCache = make(map[string]*cachedResponse)
	m.detailForm = nil
	m.detailFormKey = ""
	m.updateBadgeCache()
	m.applyFilterAndReset()
}

func joinWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	if len(warnings) == 1 {
		return warnings[0]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d schema warnings:\n", len(warnings))
	for i, warning := range warnings {
		fmt.Fprintf(&b, "%d. %s", i+1, warning)
		if i < len(warnings)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
