package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/xaaha/hulak/pkg/features/graphql"
)

func forceColorProfile(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(prev)
	})
}

func TestFormatQueryForPanelFocusedAppliesHighlighting(t *testing.T) {
	forceColorProfile(t)
	raw := "query getUser($id: ID!) {\n  getUser(id: $id) {\n    id\n  }\n}"

	got := formatQueryForPanel(raw, true)
	if got == raw {
		t.Fatal("expected focused query panel to apply highlighting")
	}
	if ansi.Strip(got) != raw {
		t.Fatal("expected highlighted query to preserve plain text content")
	}
	if !strings.Contains(got, gqlOperationStyle.Render("getUser")) {
		t.Fatal("expected operation name to receive its own accent color")
	}
}

func TestFormatQueryForPanelUnfocusedKeepsPlainText(t *testing.T) {
	raw := "query getUser { id }"

	if got := formatQueryForPanel(raw, false); got != raw {
		t.Fatal("expected unfocused query panel to keep plain text")
	}
}

func TestFormatVariablesForPanelFocusedAppliesJSONHighlighting(t *testing.T) {
	forceColorProfile(t)
	raw := "{\n  \"id\": \"123\",\n  \"enabled\": true\n}"

	got := formatVariablesForPanel(raw, true)
	if got == raw {
		t.Fatal("expected focused variables panel to apply highlighting")
	}
	if ansi.Strip(got) != raw {
		t.Fatal("expected highlighted variables to preserve plain text content")
	}
}

func TestFormatVariablesForPanelUnfocusedKeepsPlainText(t *testing.T) {
	raw := "{\n  \"id\": \"123\"\n}"

	if got := formatVariablesForPanel(raw, false); got != raw {
		t.Fatal("expected unfocused variables panel to keep plain text")
	}
}

func TestQueryAndVariablesPanelsStayPlainUntilFocused(t *testing.T) {
	forceColorProfile(t)
	objTypes := map[string]graphql.ObjectType{
		"User": {Name: "User", Fields: []graphql.ObjectField{{Name: "id", Type: "ID!"}}},
	}
	ops := []UnifiedOperation{{
		Name: "getUser", Type: TypeQuery, Endpoint: "http://api/gql", ReturnType: "User!",
		Arguments: []graphql.Argument{{Name: "id", Type: "ID!"}},
	}}
	m := NewModel(ops, nil, nil, objTypes, nil, nil, nil)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	if got := ansi.Strip(model.queryPanel.View(false)); !strings.Contains(got, "query getUser") {
		t.Fatal("expected unfocused query panel to render plain query text")
	}

	model.focus.FocusByNumber(model.queryPanel.Number)
	model.syncViewport()
	if got := model.queryPanel.View(true); ansi.Strip(got) == got {
		t.Fatal("expected focused query panel view to contain ANSI highlighting")
	}

	model.detailForm.items[0].input.Model.SetValue("123")
	model.focus.FocusByNumber(model.variablePanel.Number)
	model.syncViewport()
	if got := model.variablePanel.View(true); ansi.Strip(got) == got {
		t.Fatal("expected focused variable panel view to contain ANSI highlighting")
	}
	if got := ansi.Strip(model.variablePanel.View(true)); !strings.Contains(got, "\"id\": \"123\"") {
		t.Fatal("expected focused variable panel to preserve variable ordering and text")
	}
}
