package gqlexplorer

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func newGoldenExplorer(t *testing.T) *teatest.TestModel {
	t.Helper()
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	return teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(160, 40))
}

func newNestedInputExplorer(t *testing.T) *teatest.TestModel {
	t.Helper()
	ops := []UnifiedOperation{{
		Name:       "createExpense",
		Type:       TypeMutation,
		Endpoint:   "http://api/gql",
		ReturnType: "Expense!",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "CreateExpenseInput!"},
		},
	}}
	inputTypes := map[string]graphql.InputType{
		"CreateExpenseInput": {
			Name: "CreateExpenseInput",
			Fields: []graphql.InputField{
				{Name: "amount", Type: "Int!"},
				{Name: "tuitionDetails", Type: "[ExpenseTuitionDetailInput!]"},
			},
		},
		"ExpenseTuitionDetailInput": {
			Name: "ExpenseTuitionDetailInput",
			Fields: []graphql.InputField{
				{Name: "courseCode", Type: "String!"},
				{Name: "creditHours", Type: "Int"},
			},
		},
	}
	m := NewModel(ops, inputTypes, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	return teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(160, 40))
}

func TestExplorerNestedInputExpansion(t *testing.T) {
	tm := newNestedInputExplorer(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestExplorerInitialRender(t *testing.T) {
	tm := newGoldenExplorer(t)

	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestExplorerAfterNavigation(t *testing.T) {
	tm := newGoldenExplorer(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestExplorerAfterFiltering(t *testing.T) {
	tm := newGoldenExplorer(t)

	tm.Type("get")
	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}

func TestExplorerDetailPanelFocused(t *testing.T) {
	tm := newGoldenExplorer(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.Quit())
	out := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second))
	teatest.RequireEqualOutput(t, []byte(out.View()))
}
