package gqlexplorer

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func TestNewModelSortsQueriesFirst(t *testing.T) {
	ops := []UnifiedOperation{
		{Name: "onMsg", Type: TypeSubscription},
		{Name: "createUser", Type: TypeMutation},
		{Name: "getUser", Type: TypeQuery},
	}
	m := NewModel(ops, nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	expected := []OperationType{TypeQuery, TypeMutation, TypeSubscription}
	for i, want := range expected {
		if m.filtered[i].Type != want {
			t.Errorf("index %d: expected type %q, got %q", i, want, m.filtered[i].Type)
		}
	}
}

func TestNewModelEmptyOperations(t *testing.T) {
	m := NewModel(nil, nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	if len(m.operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(m.operations))
	}
	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered, got %d", len(m.filtered))
	}
}

func TestNewModelFilteredMatchesOperations(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	if len(m.filtered) != len(m.operations) {
		t.Errorf("expected filtered (%d) to match operations (%d)",
			len(m.filtered), len(m.operations))
	}
}

func TestNewModelCursorStartsAtZero(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestInitReturnsCmd(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a blink command")
	}
}

func TestNavigateDown(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := result.(*Model)

	if model.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.cursor = 2

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := result.(*Model)

	if model.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.cursor)
	}
}

func TestNavigateCtrlN(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	model := result.(*Model)

	if model.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.cursor)
	}
}

func TestNavigateCtrlP(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.cursor = 3

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model := result.(*Model)

	if model.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", model.cursor)
	}
}

func TestMouseClickSelectsOperationAndMovesCursor(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)
	model.cursor = len(model.filtered) - 1
	model.syncViewport()

	_ = model.View()
	x, y := waitForMouseZone(t, model.operationZoneID(1))

	result, _ = model.Update(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	model = result.(*Model)

	if model.cursor != 1 {
		t.Fatalf("expected cursor at clicked operation, got %d", model.cursor)
	}
	if !model.focus.LeftFocused() {
		t.Fatal("expected left panel to be focused after operation click")
	}
	if model.focus.Typing() {
		t.Fatal("expected typing mode off after clicking operation row")
	}
}

func TestMouseClickTogglesEndpointAndMovesEndpointCursor(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)
	model.search.Model.SetValue("e:")
	model.endpointCursor = 0
	model.applyFilterAndReset()

	eps := model.filteredEndpoints()
	if len(eps) < 2 {
		t.Fatal("expected at least two endpoints")
	}
	clicked := eps[1]

	_ = model.View()
	x, y := waitForMouseZone(t, model.endpointZoneID(1))

	result, _ = model.Update(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	model = result.(*Model)

	if model.endpointCursor != 1 {
		t.Fatalf("expected endpoint cursor at clicked row, got %d", model.endpointCursor)
	}
	if model.activeEndpoints[clicked] {
		t.Fatalf("expected clicked endpoint %q to be toggled off", clicked)
	}
	if !model.focus.LeftFocused() {
		t.Fatal("expected left panel to be focused after endpoint click")
	}
	if model.focus.Typing() {
		t.Fatal("expected typing mode off after clicking endpoint row")
	}
}

func TestMouseClickDetailFormItemFocusesDetailPanel(t *testing.T) {
	ep := "ep"
	op := UnifiedOperation{
		Name: "Search", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String"}},
	}
	m := NewModel([]UnifiedOperation{op}, nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	_ = model.View()
	x, y := waitForMouseZone(t, model.detailForm.itemZoneID(model.detailMousePrefix(), 0))

	result, _ = model.Update(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	model = result.(*Model)

	if !model.focus.IsFocused(model.detailPanel) {
		t.Fatal("expected detail panel to be focused after clicking detail item")
	}
	if model.detailForm.cursor != 0 {
		t.Fatalf("expected detail cursor on clicked item, got %d", model.detailForm.cursor)
	}
	if !model.detailForm.items[0].input.Model.Focused() {
		t.Fatal("expected clicked text input to enter editing")
	}
}

func TestMouseClickSearchInputFocusesLeftPanelAndTyping(t *testing.T) {
	ep := "ep"
	op := UnifiedOperation{
		Name: "Search", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String"}},
	}
	m := NewModel([]UnifiedOperation{op}, nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	model.focus.FocusByNumber(model.detailPanel.Number)
	model.syncSearchFocus()

	_ = model.View()
	x, y := waitForMouseZone(t, model.searchZoneID())

	result, _ = model.Update(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	model = result.(*Model)

	if !model.focus.LeftFocused() {
		t.Fatal("expected search click to focus left panel")
	}
	if !model.focus.Typing() {
		t.Fatal("expected search click to enable typing mode")
	}
	if !model.search.Model.Focused() {
		t.Fatal("expected search input to be focused after click")
	}
}

func TestTabTogglesFocus(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := result.(*Model)
	if model.focus.LeftFocused() {
		t.Error("expected detail panel focused after first tab")
	}
	if !model.focus.IsFocused(model.detailPanel) {
		t.Error("expected detail panel focused after first tab")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(*Model)
	if model.focus.LeftFocused() {
		t.Error("expected query panel focused after second tab")
	}
	if !model.focus.IsFocused(model.queryPanel) {
		t.Error("expected query panel focused after second tab")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(*Model)
	if model.focus.LeftFocused() {
		t.Error("expected variable panel focused after third tab")
	}
	if !model.focus.IsFocused(model.variablePanel) {
		t.Error("expected variable panel focused after third tab")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(*Model)
	if !model.focus.IsFocused(model.responsePanel) {
		t.Error("expected response panel focused after fourth tab")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(*Model)
	if !model.focus.LeftFocused() {
		t.Error("expected left panel focused after fifth tab")
	}
}

func TestEnterMovesFocusToDetailOnly(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	if model.focus.LeftFocused() {
		t.Error("expected detail panel focused after enter")
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	if model.focus.LeftFocused() {
		t.Error("expected detail panel to remain focused after second enter")
	}
}

func TestEnterReactivatesTypingWhenBlurred(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})

	m.focus.SetTyping(false)
	m.syncSearchFocus()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := result.(*Model)
	if !model.focus.Typing() {
		t.Error("enter on blurred left panel should reactivate typing")
	}
	if !model.focus.LeftFocused() {
		t.Error("enter on blurred left panel should stay on left, not jump to detail")
	}
}

func TestLeftArrowMovesSearchCursorWithinText(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)

	for _, r := range "ab" {
		result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = result.(*Model)
	}
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = result.(*Model)
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	model = result.(*Model)

	if got := model.search.Model.Value(); got != "aXb" {
		t.Fatalf("left arrow should move search cursor within text, got %q", got)
	}
}

func TestScrollLeftPanelWhenFocused(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := result.(*Model)

	if !model.focus.LeftFocused() {
		t.Fatal("precondition: left panel should be focused by default")
	}
	model.updateFocusedViewport(tea.KeyMsg{Type: tea.KeyDown})
}

func TestNavigateUpAtTopStays(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := result.(*Model)

	if model.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", model.cursor)
	}
}

func TestNavigateDownAtBottomStays(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.cursor = len(m.filtered) - 1

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := result.(*Model)

	if model.cursor != len(m.filtered)-1 {
		t.Errorf("expected cursor %d, got %d", len(m.filtered)-1, model.cursor)
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Error("expected quit command from ctrl+c")
	}
}

func TestEscBlursThenQuits(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := result.(*Model)
	if cmd != nil {
		t.Error("first esc should blur search, not quit")
	}
	if model.focus.Typing() {
		t.Error("expected typing=false after first esc")
	}

	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Error("second esc should quit")
	}
}

func TestEscClearsSearchFirst(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("get")
	m.applyFilter()

	filteredBefore := len(m.filtered)
	if filteredBefore == len(m.operations) {
		t.Fatal("filter should have reduced the list")
	}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := result.(*Model)

	if model.search.Model.Value() != "" {
		t.Errorf("expected search cleared, got %q", model.search.Model.Value())
	}
	if len(model.filtered) != len(model.operations) {
		t.Errorf("expected all operations restored, got %d/%d",
			len(model.filtered), len(model.operations))
	}
	if cmd != nil {
		t.Error("expected no quit command when clearing search")
	}
}
