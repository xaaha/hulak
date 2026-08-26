package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func opsWithArgs() []UnifiedOperation {
	return []UnifiedOperation{
		{
			Name:       "getUser",
			Type:       TypeQuery,
			Endpoint:   "http://api/gql",
			ReturnType: "User!",
			Arguments: []graphql.Argument{
				{Name: "id", Type: "ID!"},
				{Name: "name", Type: "String"},
			},
		},
		{
			Name:       "listUsers",
			Type:       TypeQuery,
			Endpoint:   "http://api/gql",
			ReturnType: "[User!]!",
		},
	}
}

func TestRenderDetailShowsOperationName(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil, nil, nil, nil)
	if !strings.Contains(detail, "getUser") {
		t.Error("detail should contain operation name")
	}
}

func TestRenderDetailShowsReturnType(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil, nil, nil, nil)
	if !strings.Contains(detail, "User!") {
		t.Error("detail should contain return type")
	}
}

func TestRenderDetailShowsArguments(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil, nil, nil, nil)
	if !strings.Contains(detail, "Arguments:") {
		t.Error("detail should contain Arguments header")
	}
	if !strings.Contains(detail, "id") || !strings.Contains(detail, "ID!") {
		t.Error("detail should contain argument name and type")
	}
	if !strings.Contains(detail, "(required)") {
		t.Error("detail should mark required arguments")
	}
}

func TestRenderDetailOmitsEndpoint(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil, nil, nil, nil)
	if strings.Contains(detail, "Endpoint:") {
		t.Error("detail should not show Endpoint (already in badges and list)")
	}
}

func TestRenderDetailNoArgsOmitsSection(t *testing.T) {
	op := opsWithArgs()[1]
	detail := renderDetail(&op, nil, nil, nil, nil)
	if strings.Contains(detail, "Arguments:") {
		t.Error("detail should not show Arguments section when empty")
	}
}

func TestRenderDetailOptionalArgHasNoRequiredMarker(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil, nil, nil, nil)
	lines := strings.Split(detail, "\n")
	for _, line := range lines {
		if strings.Contains(line, "name") && strings.Contains(line, "String") {
			if strings.Contains(line, "(required)") {
				t.Error("optional argument 'name' should not have (required) marker")
			}
			return
		}
	}
	t.Error("did not find 'name' argument line in detail")
}

func TestViewShowsDetailPanel(t *testing.T) {
	m := NewModel(opsWithArgs(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = *result.(*Model)
	view := m.View()

	if !strings.Contains(view, "User!") {
		t.Error("view should show detail panel with return type in header")
	}
}

func TestDetailPanelUpdatesOnCursorMove(t *testing.T) {
	m := NewModel(opsWithArgs(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = *result.(*Model)

	view1 := m.View()
	if !strings.Contains(view1, "User!") {
		t.Error("first operation should show User! return type")
	}

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = *result.(*Model)
	view2 := m.View()
	if !strings.Contains(view2, "[User!]!") {
		t.Error("second operation should show [User!]! return type")
	}
}

func TestRenderDetailExpandsInputType(t *testing.T) {
	inputTypes := map[string]graphql.InputType{
		"PersonInput": {
			Name: "PersonInput",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String!"},
				{Name: "age", Type: "Int"},
			},
		},
	}
	op := UnifiedOperation{
		Name:       "createUser",
		Type:       TypeMutation,
		ReturnType: "User!",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "PersonInput!"},
		},
	}
	detail := renderDetail(&op, inputTypes, nil, nil, nil)
	if !strings.Contains(detail, "name") || !strings.Contains(detail, "String!") {
		t.Error("detail should expand PersonInput fields showing name and type")
	}
	if !strings.Contains(detail, "age") || !strings.Contains(detail, "Int") {
		t.Error("detail should expand PersonInput fields showing age")
	}
	if !strings.Contains(detail, "├─") || !strings.Contains(detail, "└─") {
		t.Error("detail should use tree connectors for input type fields")
	}
}

func TestRenderDetailNestedInputType(t *testing.T) {
	inputTypes := map[string]graphql.InputType{
		"CreateUserInput": {
			Name: "CreateUserInput",
			Fields: []graphql.InputField{
				{Name: "person", Type: "PersonInput!"},
				{Name: "role", Type: "String"},
			},
		},
		"PersonInput": {
			Name: "PersonInput",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String!"},
			},
		},
	}
	op := UnifiedOperation{
		Name:       "createUser",
		Type:       TypeMutation,
		ReturnType: "User!",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "CreateUserInput!"},
		},
	}
	detail := renderDetail(&op, inputTypes, nil, nil, nil)
	if !strings.Contains(detail, "person") {
		t.Error("detail should show nested input type field 'person'")
	}
	if !strings.Contains(detail, "name") {
		t.Error("detail should expand nested PersonInput showing 'name'")
	}
}

func TestAppendInputTypeFieldsDepthCap(t *testing.T) {
	selfRef := map[string]graphql.InputType{
		"Recursive": {
			Name: "Recursive",
			Fields: []graphql.InputField{
				{Name: "child", Type: "Recursive"},
			},
		},
	}
	lines := appendInputTypeFields(
		nil, selfRef["Recursive"], "", selfRef, "", 1,
	)
	// depths 1→2→3 each emit one line, then recursion stops at maxInputTypeDepth
	if len(lines) != maxInputTypeDepth {
		t.Errorf("expected %d lines (depth cap), got %d", maxInputTypeDepth, len(lines))
	}
}

func TestRenderDetailNilInputTypes(t *testing.T) {
	op := UnifiedOperation{
		Name:       "getUser",
		Type:       TypeQuery,
		ReturnType: "User!",
		Arguments: []graphql.Argument{
			{Name: "id", Type: "ID!"},
		},
	}
	detail := renderDetail(&op, nil, nil, nil, nil)
	if !strings.Contains(detail, "id") {
		t.Error("detail should still render arguments with nil inputTypes")
	}
}

func TestDetailTopHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		// containerStyle vertical frame = 2, HelpBarHeight = 1.
		// contentH = max(height-3, 1), top = max(contentH*40/100, 1)
		{"typical terminal", 40, 14},
		{"small terminal", 10, 2},
		{"minimum size", 5, 1},
		{"zero height", 0, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{height: tc.height, helpBarH: tui.HelpBarHeight}
			got := m.detailTopHeight()
			if got != tc.want {
				t.Errorf("detailTopHeight() = %d, want %d", got, tc.want)
			}
			if got < 1 {
				t.Errorf("detailTopHeight() = %d, must be >= 1", got)
			}
		})
	}
}

func TestResponseAreaHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
	}{
		{"typical terminal", 40},
		{"small terminal", 10},
		{"zero height", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{height: tc.height, helpBarH: tui.HelpBarHeight}
			got := m.callAreaHeight()
			want := max(m.contentHeight()-m.detailTopHeight()-m.variablePanelHeight(), 1)
			if got != want {
				t.Errorf("responseAreaHeight() = %d, want %d", got, want)
			}
			if got < 1 {
				t.Errorf("responseAreaHeight() = %d, must be >= 1", got)
			}
		})
	}
}

func TestHeightPartitionSumsCorrectly(t *testing.T) {
	for h := 0; h <= 100; h++ {
		m := Model{height: h, helpBarH: tui.HelpBarHeight}
		total := m.contentHeight()
		top := m.detailTopHeight()
		variable := m.variablePanelHeight()
		bottom := m.callAreaHeight()
		sum := top + variable + bottom

		// For very small heights where max() clamps to 1, the sum may exceed
		// total. For normal heights the partition should be exact.
		if total >= 3 && sum != total {
			t.Errorf("height=%d: top(%d) + variable(%d) + bottom(%d) = %d, want %d",
				h, top, variable, bottom, sum, total)
		}
	}
}

func TestRenderLeftContentFitsWithinContentHeight(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 28})
	model := result.(*Model)

	leftHeight := lipgloss.Height(model.renderLeftContent())
	contentHeight := model.contentHeight()
	if leftHeight > contentHeight {
		t.Fatalf(
			"left content exceeds available height: left=%d content=%d (width=%d)",
			leftHeight,
			contentHeight,
			model.width,
		)
	}
}

func TestHelpBarChangesWithFocus(t *testing.T) {
	// Width must be wider than the longest help constant so lipgloss
	// centering does not wrap the text.
	const w = 240
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 40})
	model := result.(*Model)

	leftHelp := model.renderHelpBar(w)
	if !strings.Contains(leftHelp, helpLeftPanel) {
		t.Error("left-focused help bar should contain helpLeftPanel text")
	}

	model.focus.FocusByNumber(model.detailPanel.Number)
	detailHelp := model.renderHelpBar(w)
	if !strings.Contains(detailHelp, helpDetailPanel) {
		t.Error("detail-focused help bar should contain helpDetailPanel text")
	}

	model.focus.FocusByNumber(model.queryPanel.Number)
	queryHelp := model.renderHelpBar(w)
	if !strings.Contains(queryHelp, helpQueryPanel) {
		t.Error("query-focused help bar should contain helpQueryPanel text")
	}

	model.focus.FocusByNumber(model.variablePanel.Number)
	variableHelp := model.renderHelpBar(w)
	if !strings.Contains(variableHelp, helpVariablePanel) {
		t.Error("variable-focused help bar should contain helpVariablePanel text")
	}
}

func TestEnterNoFocusChangeInSinglePanel(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 40})
	model := result.(*Model)
	model.focus.FocusByNumber(1)
	model.syncSearchFocus()

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	if !model.focus.LeftFocused() {
		t.Error("expected left panel to stay focused in single-panel layout after enter")
	}
}
