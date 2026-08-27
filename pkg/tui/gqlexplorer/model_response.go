package gqlexplorer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

func (m *Model) clearResponse() {
	m.responseSearch.Stop()
	m.responseBody = ""
	m.responseColoredBody = ""
	m.responseStatusCode = 0
	m.responseDuration = ""
	m.responsePanel.SetHeader("")
	m.responsePanel.SetContent("", "")
	m.responsePanel.Footer = ""
}

type cachedResponse struct {
	body        string
	coloredBody string
	statusCode  int
	duration    string
}

func (m *Model) saveResponseToCache() {
	if m.detailFormKey == "" || m.responseBody == "" {
		return
	}
	m.responseCache[m.detailFormKey] = &cachedResponse{
		body:        m.responseBody,
		coloredBody: m.responseColoredBody,
		statusCode:  m.responseStatusCode,
		duration:    m.responseDuration,
	}
}

func (m *Model) restoreResponseFromCache(formKey string) {
	cached, ok := m.responseCache[formKey]
	if !ok {
		m.clearResponse()
		return
	}
	m.responseSearch.Stop()
	m.responseBody = cached.body
	m.responseColoredBody = cached.coloredBody
	m.responseStatusCode = cached.statusCode
	m.responseDuration = cached.duration
	m.setResponseContent()
	m.responsePanel.GotoTop()
	m.responsePanel.Footer = ""
}

func (m *Model) handleQueryExecuted(msg *queryExecutedMsg) {
	var bodyJSON []byte
	if msg.resp.Response != nil && msg.resp.Response.Body != nil {
		bodyJSON, _ = json.MarshalIndent(msg.resp.Response.Body, "", "  ")
	}
	if len(bodyJSON) == 0 {
		bodyJSON, _ = json.MarshalIndent(msg.resp, "", "  ")
	}
	m.responseBody = string(bodyJSON)

	colored, err := utils.FormatJSONColored(bodyJSON, utils.JSONColors)
	if err != nil {
		m.responseColoredBody = m.responseBody
	} else {
		m.responseColoredBody = colored
	}

	if msg.resp.Response != nil {
		m.responseStatusCode = msg.resp.Response.StatusCode
	} else {
		m.responseStatusCode = 0
	}
	m.responseDuration = msg.resp.Duration

	m.setResponseContent()
	m.responsePanel.GotoTop()
	m.responseSearch.Stop()
	m.responsePanel.Footer = ""
	m.saveResponseToCache()
}

func (m *Model) handleResponseSearchKey(msg tea.KeyMsg) tea.Cmd {
	stopped, _, cmd := m.responseSearch.HandleKey(msg)
	if stopped {
		m.syncResponseFooter()
		return cmd
	}
	switch msg.String() {
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyDown, tui.KeyCtrlN:
		m.scrollResponseToMatch()
	default:
		m.updateResponseSearchMatches()
	}
	m.syncResponseFooter()
	return cmd
}

func (m *Model) updateResponseSearchMatches() {
	query := strings.ToLower(m.responseSearch.Query())
	var indices []int
	if query != "" {
		for i, line := range strings.Split(m.responseBody, "\n") {
			if strings.Contains(strings.ToLower(line), query) {
				indices = append(indices, i)
			}
		}
	}
	m.responseSearch.SetMatches(indices)
	m.scrollResponseToMatch()
}

func (m *Model) saveZoneID() string { return m.mouse.ID("resp-save") }

func (m *Model) saveResponse() tea.Cmd {
	if m.responseBody == "" {
		return nil
	}
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	op := &m.filtered[m.cursor]

	dir := m.responseSaveDir(op.Endpoint)
	if dir == "" {
		return m.enqueueNotification(tui.NotificationError, "Cannot determine save directory")
	}

	// if we need to make this customizable, we need to re-think how we save name
	stamp := time.Now().Format("02-01-2006-15:04:05.00")
	fileName := op.Name + "-" + stamp + utils.ResponseBase + ".json"
	fullPath := filepath.Join(dir, fileName)

	if err := os.WriteFile(fullPath, []byte(m.responseBody), utils.SecretPer); err != nil {
		return m.enqueueNotification(tui.NotificationError, "Save failed: "+err.Error())
	}
	return m.enqueueNotification(tui.NotificationInfo, "Saved "+relativePath(fullPath))
}

func (m *Model) responseSaveDir(endpoint string) string {
	if p, ok := m.schemaFilePaths[endpoint]; ok && p != "" {
		return filepath.Dir(p)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

func (m *Model) setResponseContent() {
	header := m.responseHeader()
	if header != "" {
		header = lipgloss.NewStyle().
			Width(m.responsePanel.Width()).
			Align(lipgloss.Center).
			Render(header)
	}
	m.responsePanel.SetHeader(header)
	m.responsePanel.SetContent(m.responseColoredBody, "")
}

func (m *Model) responseHeader() string {
	if m.responseBody == "" {
		return ""
	}
	focused := m.focus.IsFocused(m.responsePanel)
	var parts []string

	if m.responseStatusCode > 0 {
		code := strconv.Itoa(m.responseStatusCode)
		chipColor := focusColor(focused, statusChipColor(m.responseStatusCode))
		parts = append(parts, tui.RenderChip(code, tui.ChipVariantBadge, chipColor))
	}
	if m.responseDuration != "" {
		parts = append(parts, tui.HelpStyle.Render(m.responseDuration))
	}
	saveLabel := "Save"
	if focused {
		saveLabel = "Save Ctrl+S"
	}
	saveColor := focusColor(focused, badgeColor[TypeMutation])
	saveChip := tui.RenderChip(saveLabel, tui.ChipVariantBadge, saveColor)
	parts = append(
		parts,
		tui.HelpStyle.Render(formatResponseSize(len(m.responseBody))),
		m.mouse.Mark(m.saveZoneID(), saveChip),
	)

	return strings.Join(parts, "  ")
}

func statusChipColor(code int) lipgloss.TerminalColor {
	switch {
	case code >= 200 && code < 300:
		return tui.ColorSuccess
	case code >= 400 && code < 500:
		return tui.ColorWarn
	default:
		return tui.ColorError
	}
}

func formatResponseSize(n int) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func (m *Model) syncResponseFooter() {
	m.responsePanel.Footer = m.responseSearch.Footer()
	if !m.responseSearch.Active() {
		m.setResponseContent()
	}
}

var searchHighlightStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("3")). // yellow
	Foreground(lipgloss.Color("0"))  // black

func (m *Model) scrollResponseToMatch() {
	matchLine := m.responseSearch.CurrentMatch()
	if matchLine < 0 {
		return
	}

	query := strings.ToLower(m.responseSearch.Query())
	plainLines := strings.Split(m.responseBody, "\n")
	coloredLines := strings.Split(m.responseColoredBody, "\n")

	colStart := 0
	colEnd := 0
	if matchLine < len(plainLines) && matchLine < len(coloredLines) {
		plain := plainLines[matchLine]
		idx := strings.Index(strings.ToLower(plain), query)
		if idx >= 0 {
			colStart = idx
			colEnd = idx + len(query)
			before := plain[:idx]
			match := plain[idx : idx+len(query)]
			after := plain[idx+len(query):]
			coloredLines[matchLine] = before + searchHighlightStyle.Render(match) + after
		}
	}

	content := strings.Join(coloredLines, "\n")
	m.responsePanel.SyncContent(content, matchLine)
	m.responsePanel.EnsureVisible(matchLine, colStart, colEnd)
}
