package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func TestBuildVariablesStringScalarArgs(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "getUser",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID!"},
			{Name: "active", Type: "Boolean"},
			{Name: "limit", Type: "Int"},
		},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].input.Model.SetValue("user-123")
	df.items[1].enabled = true
	df.items[1].toggle.Value = true
	df.items[2].enabled = true
	df.items[2].input.Model.SetValue("25")

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"id\": \"user-123\",\n" +
		"  \"active\": true,\n" +
		"  \"limit\": 25\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildVariablesStringInputObjectArg(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "searchUsers",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "filter", Type: "UserFilter!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"UserFilter": {
			Name: "UserFilter",
			Fields: []graphql.InputField{
				{Name: "query", Type: "String"},
				{Name: "active", Type: "Boolean"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].enabled = true
	df.items[0].input.Model.SetValue("alice")
	df.items[1].enabled = true
	df.items[1].toggle.Value = true

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"filter\": {\n" +
		"    \"query\": \"alice\",\n" +
		"    \"active\": true\n" +
		"  }\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildVariablesStringListArgFromRepeatedInputs(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "getUsers",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "ids", Type: "[UUID!]!"},
		},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].input.Model.SetValue("11111111-1111-1111-1111-111111111111")
	df.syncListArgRows("ids")
	df.items[1].input.Model.SetValue("22222222-2222-2222-2222-222222222222")
	df.syncListArgRows("ids")

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"ids\": [\"11111111-1111-1111-1111-111111111111\", \"22222222-2222-2222-2222-222222222222\"]\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildVariablesStringListArgAllowsNullItems(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "getUsers",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "ids", Type: "[ID!]!"},
		},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].input.Model.SetValue("a")
	df.syncListArgRows("ids")
	df.items[1].input.Model.SetValue("null")
	df.syncListArgRows("ids")

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"ids\": [\"a\", null]\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildVariablesStringListEnumArg(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "getUsers",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "statuses", Type: "[Status!]!"},
		},
	}
	enums := map[string]graphql.EnumType{
		"Status": {Name: "Status", Values: []graphql.EnumValue{{Name: "OPEN"}, {Name: "CLOSED"}}},
	}
	df := buildDetailForm(op, nil, enums, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].dropdown.Selected = 1
	df.syncListArgRows("statuses")
	df.items[1].dropdown.Selected = 2
	df.syncListArgRows("statuses")

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"statuses\": [\"OPEN\", \"CLOSED\"]\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildVariablesStringListInputObjectArg(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "searchUsers",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "filters", Type: "[UserFilter!]!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"UserFilter": {
			Name: "UserFilter",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String"},
				{Name: "active", Type: "Boolean"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].enabled = true
	df.items[0].input.Model.SetValue("alice")
	df.items[1].enabled = true
	df.items[1].toggle.Value = true
	df.syncListArgRows("filters")
	df.items[2].enabled = true
	df.items[2].input.Model.SetValue("bob")
	df.syncListArgRows("filters")

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"filters\": [{\n" +
		"    \"name\": \"alice\",\n" +
		"    \"active\": true\n" +
		"  }, {\n" +
		"    \"name\": \"bob\"\n" +
		"  }]\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildVariablesStringOmitsEmptyTextInputs(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "getUser",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID!"},
		},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	if got := BuildVariablesString(op, df); got != "" {
		t.Fatalf("expected empty variables string for blank text input, got %q", got)
	}
}

func TestBuildVariablesStringSupportsNull(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "getUser",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID"},
		},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	df.items[0].enabled = true
	df.items[0].input.Model.SetValue("null")

	got := BuildVariablesString(op, df)
	want := "{\n" +
		"  \"id\": null\n" +
		"}"
	if got != want {
		t.Fatalf("BuildVariablesString()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestLeftArrowMovesDetailInputCursorWithinText(t *testing.T) {
	ops := []UnifiedOperation{{
		Name:     "getUser",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID!"},
		},
	}}
	m := NewModel(ops, nil, nil, nil, nil, nil)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	model.focus.FocusByNumber(model.detailPanel.Number)
	model.syncViewport()

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)

	for _, r := range "ab" {
		result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = result.(*Model)
	}
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = result.(*Model)
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	model = result.(*Model)

	if got := model.detailForm.items[0].input.Model.Value(); got != "aXb" {
		t.Fatalf("left arrow should move detail input cursor within text, got %q", got)
	}
}

func TestVariablePanelShowsVariables(t *testing.T) {
	objTypes := map[string]graphql.ObjectType{
		"User": {
			Name: "User",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "ID!"},
			},
		},
	}
	ops := []UnifiedOperation{{
		Name:       "getUser",
		Type:       TypeQuery,
		Endpoint:   "http://api/gql",
		ReturnType: "User!",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID!"},
		},
	}}
	m := NewModel(ops, nil, nil, objTypes, nil, nil)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	if model.detailForm == nil {
		t.Fatal("expected detail form")
	}
	model.detailForm.items[0].input.Model.SetValue("user-123")
	model.syncViewport()

	view := model.View()
	if !strings.Contains(view, "\"id\": \"user-123\"") {
		t.Fatalf("view should contain rendered variables payload, got:\n%s", view)
	}
}

func TestYankTextVariablePanel(t *testing.T) {
	ops := []UnifiedOperation{{
		Name:     "getUser",
		Type:     TypeQuery,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID!"},
		},
	}}
	m := NewModel(ops, nil, nil, nil, nil, nil)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	if model.detailForm == nil {
		t.Fatal("expected detail form")
	}
	model.detailForm.items[0].input.Model.SetValue("user-123")
	model.syncViewport()
	model.focus.FocusByNumber(model.variablePanel.Number)

	if got := model.yankText(); !strings.Contains(got, "\"id\": \"user-123\"") {
		t.Fatalf("variable panel yank should contain variables payload, got %q", got)
	}
}
