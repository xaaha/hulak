package gqlexplorer

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m *Model) spinnerContent() string {
	frame := tui.SpinnerFrames[m.spinnerFrame%len(tui.SpinnerFrames)]
	return tui.HelpStyle.Render(string(frame) + " Executing...")
}

func (m *Model) executeQuery() tea.Cmd {
	if m.executing {
		return nil
	}
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	op := &m.filtered[m.cursor]

	if op.Type == TypeSubscription {
		return m.enqueueNotification(
			tui.NotificationWarn,
			"Subscriptions execution is not supported yet",
		)
	}

	info, ok := m.apiInfos[op.Endpoint]
	if !ok {
		return m.enqueueNotification(
			tui.NotificationError,
			"No API configuration found for "+op.Endpoint,
		)
	}

	query := BuildQueryString(op, m.detailForm)
	if query == "" {
		return m.enqueueNotification(tui.NotificationError, "Empty query")
	}

	varsMap := BuildVariablesMap(op, m.detailForm)
	apiInfo := yamlparser.CloneAPIInfo(info)

	body, err := yamlparser.EncodeGraphQlBody(query, varsMap)
	if err != nil {
		return m.enqueueNotification(tui.NotificationError, "Failed to encode query: "+err.Error())
	}
	apiInfo.Body = body

	m.executing = true
	m.spinnerFrame = 0
	m.clearResponse()
	m.responsePanel.SetContent(m.spinnerContent(), "")
	m.updateActionRow()

	apiCall := func() tea.Msg {
		resp, err := apicalls.StandardCall(context.Background(), apiInfo, false)
		if err != nil {
			return queryErrorMsg{err: err}
		}
		return queryExecutedMsg{resp: resp}
	}
	return tea.Batch(apiCall, spinnerTick())
}
