package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func TestCtrlRRefreshesExplorerData(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.SetRefresh(func() (RefreshPayload, error) {
		return RefreshPayload{
			Data: ExplorerData{
				Operations: []UnifiedOperation{
					{Name: "refreshedUser", Type: TypeQuery, Endpoint: "http://api/gql"},
				},
			},
		}, nil
	})

	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	model = result.(*Model)
	if cmd == nil {
		t.Fatal("expected refresh command")
	}

	result, _ = model.Update(cmd())
	model = result.(*Model)

	if len(model.operations) != 1 || model.operations[0].Name != "refreshedUser" {
		t.Fatalf("expected refreshed operations, got %#v", model.operations)
	}
}

func TestRefreshWarningsShowNotificationBadge(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.SetRefresh(func() (RefreshPayload, error) {
		return RefreshPayload{
			Data: ExplorerData{Operations: sampleOps()},
			Warnings: []string{
				"http://bad/graphql: introspection request returned status 500",
				"/tmp/other.yaml: error in headers",
			},
		}, nil
	})

	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	model = result.(*Model)
	result, notifyCmd := model.Update(cmd())
	model = result.(*Model)
	if notifyCmd == nil {
		t.Fatal("expected notification expiry command")
	}

	view := model.View()
	if !strings.Contains(view, "Warning") ||
		!strings.Contains(view, "http://bad/graphql: introspection request") {
		t.Fatalf("expected warning notification content in view, got:\n%s", view)
	}
	copied := model.notification.CopyText()
	if !strings.Contains(copied, "2 schema warnings:") ||
		!strings.Contains(copied, "1. http://bad/graphql: introspection request returned status 500") ||
		!strings.Contains(copied, "2. /tmp/other.yaml: error in headers") {
		t.Fatalf("expected full multi-warning text in copy buffer, got:\n%s", copied)
	}
}

func TestEscDismissesVisibleNotificationModal(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)
	_ = model.enqueueNotification(tui.NotificationWarn, "schema warning")

	if !model.notification.Visible() {
		t.Fatal("expected visible notification")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(*Model)

	if model.notification.Visible() {
		t.Fatal("expected esc to dismiss notification modal")
	}
}

func TestShiftTabCyclesBackward(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model := result.(*Model)
	if !model.focus.IsFocused(model.responsePanel) {
		t.Error("shift+tab from left panel should wrap to response panel")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = result.(*Model)
	if !model.focus.IsFocused(model.variablePanel) {
		t.Error("shift+tab from response panel should go to variable panel")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = result.(*Model)
	if !model.focus.IsFocused(model.queryPanel) {
		t.Error("shift+tab from variable panel should go to query panel")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = result.(*Model)
	if !model.focus.IsFocused(model.detailPanel) {
		t.Error("shift+tab from query panel should go to detail panel")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = result.(*Model)
	if !model.focus.LeftFocused() {
		t.Error("shift+tab from detail panel should go to left panel")
	}
}

func TestEscFromVariableGoesToQuery(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	model.focus.FocusByNumber(model.variablePanel.Number)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(*Model)
	if !model.focus.IsFocused(model.queryPanel) {
		t.Error("esc from variable panel should navigate to query panel")
	}
}

func TestEscFromQueryGoesToDetail(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	model.focus.FocusByNumber(model.queryPanel.Number)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(*Model)
	if !model.focus.IsFocused(model.detailPanel) {
		t.Error("esc from query panel should navigate to detail panel, not search")
	}
}

func TestEscChainQueryToDetailToSearch(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	model.focus.FocusByNumber(model.variablePanel.Number)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(*Model)
	if !model.focus.IsFocused(model.queryPanel) {
		t.Fatal("first esc should go to query panel")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(*Model)
	if !model.focus.IsFocused(model.detailPanel) {
		t.Fatal("second esc should go to detail panel")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = result.(*Model)
	if !model.focus.LeftFocused() {
		t.Error("third esc should go to left (search) panel")
	}
}

func TestFormatOperationSummary(t *testing.T) {
	tests := []struct {
		name string
		op   UnifiedOperation
		want string
	}{
		{
			name: "all fields",
			op: UnifiedOperation{
				Name:        "getUser",
				Description: "fetch user",
				Endpoint:    "http://api/gql",
			},
			want: "getUser\n  fetch user\n  http://api/gql",
		},
		{
			name: "no description",
			op:   UnifiedOperation{Name: "listUsers", Endpoint: "http://api/gql"},
			want: "listUsers\n  http://api/gql",
		},
		{
			name: "no endpoint",
			op:   UnifiedOperation{Name: "getUser", Description: "fetch user"},
			want: "getUser\n  fetch user",
		},
		{
			name: "name only",
			op:   UnifiedOperation{Name: "getUser"},
			want: "getUser",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatOperationSummary(&tc.op)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestYankTextQueryPanel(t *testing.T) {
	objTypes := map[string]graphql.ObjectType{
		"User": {Name: "User", Fields: []graphql.ObjectField{
			{Name: "id", Type: "ID!"},
		}},
	}
	ops := []UnifiedOperation{{
		Name: "getUser", Type: TypeQuery, Endpoint: "http://api/gql", ReturnType: "User!",
	}}
	m := NewModel(ops, nil, nil, objTypes, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	model.focus.FocusByNumber(model.queryPanel.Number)
	text := model.yankText()
	if !strings.Contains(text, "query getUser") {
		t.Errorf("query panel yank should contain query string, got %q", text)
	}
}

func TestYankTextLeftPanel(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	text := model.yankText()
	if !strings.Contains(text, "getUser") {
		t.Errorf("left panel yank should contain operation name, got %q", text)
	}
	if !strings.Contains(text, "fetch user") {
		t.Errorf("left panel yank should contain description, got %q", text)
	}
	if !strings.Contains(text, "http://api/gql") {
		t.Errorf("left panel yank should contain endpoint, got %q", text)
	}
}

func TestSlashOpensSearchInDetailPanel(t *testing.T) {
	m := NewModel(opsWithArgs(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	model.focus.FocusByNumber(model.detailPanel.Number)
	model.syncSearchFocus()
	model.syncViewport()

	if model.detailForm == nil {
		t.Fatal("detailForm should exist after Enter on operation with args")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = result.(*Model)

	if !model.detailForm.IsSearching() {
		t.Fatal("/ should activate search in detail panel")
	}
}

func TestSlashDoesNotOpenSearchOnLeftPanel(t *testing.T) {
	m := NewModel(opsWithArgs(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = result.(*Model)

	if model.detailForm != nil && model.detailForm.IsSearching() {
		t.Fatal("/ on left panel should not activate detail search")
	}
}

func TestSearchHelpShownDuringSearch(t *testing.T) {
	const w = 240
	m := NewModel(opsWithArgs(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 40})
	model := result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	model.focus.FocusByNumber(model.detailPanel.Number)
	model.syncSearchFocus()
	model.syncViewport()

	normalHelp := model.renderHelpBar(w)
	if !strings.Contains(normalHelp, helpDetailPanel) {
		t.Error("should show detail help before search")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = result.(*Model)

	searchHelp := model.renderHelpBar(w)
	if !strings.Contains(searchHelp, helpSearchPanel) {
		t.Error("should show search help during search")
	}
	if strings.Contains(searchHelp, helpDetailPanel) {
		t.Error("should NOT show detail help during search")
	}
}

func TestEscClosesSearchAndRevertsCursor(t *testing.T) {
	m := NewModel(opsWithArgs(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	model.focus.FocusByNumber(model.detailPanel.Number)
	model.syncSearchFocus()
	model.syncViewport()

	original := model.detailForm.cursor

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = result.(*Model)

	if model.detailForm.IsSearching() {
		t.Fatal("Esc should close search")
	}
	if model.detailForm.cursor != original {
		t.Errorf("cursor should revert to %d, got %d", original, model.detailForm.cursor)
	}
}
