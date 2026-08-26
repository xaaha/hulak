package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func opsWithFields() []UnifiedOperation {
	return []UnifiedOperation{
		{
			Name:       "getUser",
			Type:       TypeQuery,
			Endpoint:   "http://api/gql",
			ReturnType: "User!",
		},
		{
			Name:       "getPost",
			Type:       TypeQuery,
			Endpoint:   "http://api/gql",
			ReturnType: "Post!",
		},
	}
}

func TestFormCachePreservesState(t *testing.T) {
	objTypes := map[string]graphql.ObjectType{
		"User": {Name: "User", Fields: []graphql.ObjectField{
			{Name: "id", Type: "ID!"},
			{Name: "name", Type: "String"},
		}},
		"Post": {Name: "Post", Fields: []graphql.ObjectField{
			{Name: "title", Type: "String"},
		}},
	}
	m := NewModel(opsWithFields(), nil, nil, objTypes, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	if model.detailForm == nil {
		t.Fatal("expected detail form for getUser")
	}
	if model.detailForm.Len() != 2 {
		t.Fatalf("expected 2 field items, got %d", model.detailForm.Len())
	}
	if !model.detailForm.items[0].toggle.Value {
		t.Fatal("expected first field toggled on by default")
	}

	model.detailForm.items[0].toggle.Value = false

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(*Model)

	if model.filtered[model.cursor].Name != "getPost" {
		t.Fatalf("expected cursor on getPost, got %s", model.filtered[model.cursor].Name)
	}
	if model.detailForm == nil || model.detailForm.Len() != 1 {
		t.Fatal("expected detail form for getPost with 1 field")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = result.(*Model)

	if model.filtered[model.cursor].Name != "getUser" {
		t.Fatalf("expected cursor on getUser, got %s", model.filtered[model.cursor].Name)
	}
	if model.detailForm == nil {
		t.Fatal("expected cached detail form for getUser")
	}
	if model.detailForm.items[0].toggle.Value {
		t.Error("expected first field to remain toggled off after cache restore")
	}
}

func TestFormCacheCleared(t *testing.T) {
	objTypes := map[string]graphql.ObjectType{
		"User": {Name: "User", Fields: []graphql.ObjectField{
			{Name: "id", Type: "ID!"},
		}},
		"Post": {Name: "Post", Fields: []graphql.ObjectField{
			{Name: "title", Type: "String"},
		}},
	}
	m := NewModel(opsWithFields(), nil, nil, objTypes, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = result.(*Model)

	if len(model.formCache) != 1 {
		t.Errorf("expected 1 cached form after switching, got %d", len(model.formCache))
	}
}

func TestQueryPanelShowsQueryString(t *testing.T) {
	objTypes := map[string]graphql.ObjectType{
		"User": {Name: "User", Fields: []graphql.ObjectField{
			{Name: "id", Type: "ID!"},
			{Name: "name", Type: "String"},
		}},
	}
	ops := []UnifiedOperation{{
		Name: "getUser", Type: TypeQuery, Endpoint: "http://api/gql", ReturnType: "User!",
	}}
	m := NewModel(ops, nil, nil, objTypes, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	view := model.View()
	if !strings.Contains(view, "query getUser") {
		t.Error("view should contain query string in panel [3]")
	}
	if !strings.Contains(view, "id") {
		t.Error("query string should include selected field 'id'")
	}
	if !strings.Contains(view, "Query") {
		t.Error("view should contain query panel bottom-left label")
	}
}

func TestVariablePanelShowsBottomLeftLabelWhenEmpty(t *testing.T) {
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

	view := model.View()
	if !strings.Contains(view, "Variables") {
		t.Error("view should contain variable panel bottom-left label")
	}
}

func TestViewShowsRefreshButtonInCallArea(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.SetRefresh(func() (RefreshPayload, error) {
		return RefreshPayload{}, nil
	})

	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	view := model.View()
	if !strings.Contains(view, "Refresh") || !strings.Contains(view, "ctrl+r") {
		t.Fatalf("expected refresh button in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Send") || !strings.Contains(view, "ctrl+o") {
		t.Fatalf("expected send action in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Save Query") || !strings.Contains(view, "ctrl+q") {
		t.Fatalf("expected save query action in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Save Request") || !strings.Contains(view, "ctrl+x") {
		t.Fatalf("expected save request action in view, got:\n%s", view)
	}
}
