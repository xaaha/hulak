package gqlexplorer

import (
	"sort"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func waitForMouseZone(t *testing.T, id string) (int, int) {
	return waitForMouseZoneMinHeight(t, id, 0)
}

func waitForMouseZoneMinHeight(t *testing.T, id string, minHeight int) (int, int) {
	t.Helper()
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		startX, startY, _, endY, ok := tui.ZoneBounds(id)
		if ok && endY-startY >= minHeight {
			return startX, startY
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("zone %q was not registered", id)
	return 0, 0
}

func sampleOps() []UnifiedOperation {
	return []UnifiedOperation{
		{Name: "getUser", Type: TypeQuery, Description: "fetch user", Endpoint: "http://api/gql"},
		{Name: "listUsers", Type: TypeQuery, Endpoint: "http://api/gql"},
		{Name: "createUser", Type: TypeMutation, Endpoint: "http://api/gql"},
		{Name: "deleteUser", Type: TypeMutation, Endpoint: "http://api/gql"},
		{
			Name:        "onMessage",
			Type:        TypeSubscription,
			Description: "new messages",
			Endpoint:    "http://api/gql",
		},
	}
}

func multiEndpointOps() []UnifiedOperation {
	return []UnifiedOperation{
		{Name: "getUser", Type: TypeQuery, Endpoint: "https://api.spacex.com/graphql"},
		{Name: "listRockets", Type: TypeQuery, Endpoint: "https://api.spacex.com/graphql"},
		{
			Name:     "getCountry",
			Type:     TypeQuery,
			Endpoint: "https://countries.trevorblades.com/graphql",
		},
		{Name: "createPost", Type: TypeMutation, Endpoint: "https://api.spacex.com/graphql"},
		{
			Name:     "updateCountry",
			Type:     TypeMutation,
			Endpoint: "https://countries.trevorblades.com/graphql",
		},
	}
}

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

func TestFilterByName(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	for _, r := range "get" {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *result.(*Model)
	}

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match for 'get', got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "getUser" {
		t.Errorf("expected 'getUser', got %q", m.filtered[0].Name)
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	for _, r := range "GETUSER" {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = *result.(*Model)
	}

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match for 'GETUSER', got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "getUser" {
		t.Errorf("expected 'getUser', got %q", m.filtered[0].Name)
	}
}

func TestFilterNoMatches(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("zzzzz")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.filtered))
	}
}

func TestFilterEmptyRestoresAll(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("get")
	m.applyFilter()

	m.search.Model.SetValue("")
	m.applyFilter()

	if len(m.filtered) != len(m.operations) {
		t.Errorf("expected all %d operations, got %d",
			len(m.operations), len(m.filtered))
	}
}

func TestFilterCursorClampedWhenListShrinks(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.cursor = 4

	m.search.Model.SetValue("getUser")
	m.applyFilter()

	if m.cursor >= len(m.filtered) {
		t.Errorf("cursor %d should be < filtered length %d",
			m.cursor, len(m.filtered))
	}
}

func TestFilterByTypeQueryPrefix(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("q:")
	m.applyFilter()

	for _, op := range m.filtered {
		if op.Type != TypeQuery {
			t.Errorf("expected only queries, got %q (%s)", op.Name, op.Type)
		}
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 queries, got %d", len(m.filtered))
	}
}

func TestFilterByTypeMutationPrefix(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("m:")
	m.applyFilter()

	for _, op := range m.filtered {
		if op.Type != TypeMutation {
			t.Errorf("expected only mutations, got %q (%s)", op.Name, op.Type)
		}
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(m.filtered))
	}
}

func TestFilterByTypeSubscriptionPrefix(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("s:")
	m.applyFilter()

	for _, op := range m.filtered {
		if op.Type != TypeSubscription {
			t.Errorf("expected only subscriptions, got %q (%s)", op.Name, op.Type)
		}
	}
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(m.filtered))
	}
}

func TestFilterByTypePrefixUpperCase(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("Q:")
	m.applyFilter()

	for _, op := range m.filtered {
		if op.Type != TypeQuery {
			t.Errorf("expected only queries with 'Q:', got %q (%s)", op.Name, op.Type)
		}
	}
}

func TestFilterByTypePrefixWithNameSearch(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("q:get")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 match for 'q:get', got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "getUser" {
		t.Errorf("expected 'getUser', got %q", m.filtered[0].Name)
	}
}

func TestFilterByTypePrefixNoNameMatch(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("q:zzz")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches for 'q:zzz', got %d", len(m.filtered))
	}
}

func TestFilterUnknownPrefixTreatedAsPlainSearch(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("x:foo")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches for 'x:foo', got %d", len(m.filtered))
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := result.(*Model)

	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
	if !model.ready {
		t.Error("viewport should be initialized after WindowSizeMsg")
	}
}

func TestWindowSizeMsgHidesHeaderExtrasBelowThreshold(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 40})
	model := result.(*Model)

	if model.search.Model.Placeholder != "" {
		t.Errorf(
			"expected empty placeholder below threshold, got %q",
			model.search.Model.Placeholder,
		)
	}
	if model.badgeCache != "" {
		t.Errorf("expected empty badge cache below threshold, got %q", model.badgeCache)
	}
}

func TestWindowSizeMsgShowsHeaderExtrasAtThreshold(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 111, Height: 40})
	model := result.(*Model)

	if model.search.Model.Placeholder == "" {
		t.Error("expected placeholder at threshold width")
	}
	if model.badgeCache == "" {
		t.Error("expected non-empty badge cache at threshold width")
	}
}

func TestViewContainsSearchPrompt(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "Search") {
		t.Error("view should contain search prompt")
	}
}

func TestViewContainsFilterHint(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "q: queries") {
		t.Error("view should contain filter hint for queries")
	}
	if !strings.Contains(view, "m: mutations") {
		t.Error("view should contain filter hint for mutations")
	}
	if !strings.Contains(view, "s: subscriptions") {
		t.Error("view should contain filter hint for subscriptions")
	}
}

func TestFocusColor(t *testing.T) {
	if got := focusColor(true, badgeColor[TypeQuery]); got != badgeColor[TypeQuery] {
		t.Fatal("expected active color while focused")
	}

	if got := focusColor(false, badgeColor[TypeQuery]); got != tui.ColorMuted {
		t.Fatal("expected muted color while unfocused")
	}
}

func TestFilterHelpText(t *testing.T) {
	tests := []struct {
		name    string
		ops     []UnifiedOperation
		want    []string
		wantNot []string
	}{
		{
			name:    "single type returns empty",
			ops:     []UnifiedOperation{{Name: "getUser", Type: TypeQuery}},
			wantNot: []string{"q: queries", "m: mutations", "s: subscriptions"},
		},
		{
			name:    "empty operations returns empty",
			ops:     nil,
			wantNot: []string{"q: queries", "m: mutations", "s: subscriptions"},
		},
		{
			name: "two types shows both",
			ops: []UnifiedOperation{
				{Name: "getUser", Type: TypeQuery},
				{Name: "createUser", Type: TypeMutation},
			},
			want:    []string{"q: queries", "m: mutations"},
			wantNot: []string{"s: subscriptions"},
		},
		{
			name: "all three types shows all",
			ops: []UnifiedOperation{
				{Name: "getUser", Type: TypeQuery},
				{Name: "createUser", Type: TypeMutation},
				{Name: "onMsg", Type: TypeSubscription},
			},
			want: []string{"q: queries", "m: mutations", "s: subscriptions"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(tc.ops, nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
			hint := m.filterHint
			for _, s := range tc.want {
				if !strings.Contains(hint, s) {
					t.Errorf("expected %q in hint %q", s, hint)
				}
			}
			for _, s := range tc.wantNot {
				if strings.Contains(hint, s) {
					t.Errorf("unexpected %q in hint %q", s, hint)
				}
			}
		})
	}
}

func TestViewContainsOperationCount(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "1/5 operations") {
		t.Errorf("view should contain '1/5 operations', got:\n%s", view)
	}
}

func TestViewContainsOperationNames(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	for _, name := range []string{"getUser", "listUsers", "createUser", "deleteUser", "onMessage"} {
		if !strings.Contains(view, name) {
			t.Errorf("view should contain operation %q", name)
		}
	}
}

func TestViewContainsHelpText(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, helpLeftPanel) {
		t.Error("view should contain left panel help text")
	}
}

func TestViewShowsNoMatchesWhenFilteredEmpty(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	m.search.Model.SetValue("zzzzz")
	m.applyFilter()
	view := m.View()

	if !strings.Contains(view, "(no matches)") {
		t.Error("view should show '(no matches)' when filtered list is empty")
	}
}

func TestViewShowsSelectedCursor(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, utils.ChevronRight) {
		t.Error("view should contain chevron cursor marker for selected item")
	}
}

func TestViewShowsDescriptionForSelectedItem(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "fetch user") {
		t.Error("view should show description for the selected item")
	}
}

func TestViewShowsEndpointForSelectedItem(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "http://api/gql") {
		t.Error("view should show endpoint for the selected item")
	}
}

func TestViewHasBorder(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("view should have rounded border characters")
	}
}

func TestViewFilteredCountUpdates(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	m.search.Model.SetValue("q:")
	m.applyFilter()
	view := m.View()

	if !strings.Contains(view, "1/2 operations") {
		t.Errorf("view should contain '1/2 operations' after filtering, got:\n%s", view)
	}
}

func TestViewOperationCountTracksCursorPosition(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.width = 160
	m.height = 40
	m.cursor = 2

	view := m.View()

	if !strings.Contains(view, "3/5 operations") {
		t.Errorf("view should contain '3/5 operations' for the third selected item, got:\n%s", view)
	}
}

func TestTypeRank(t *testing.T) {
	tests := []struct {
		name     string
		opType   OperationType
		expected int
	}{
		{"query", TypeQuery, 0},
		{"mutation", TypeMutation, 1},
		{"subscription", TypeSubscription, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := typeRank[tc.opType]; got != tc.expected {
				t.Errorf("typeRank[%q] = %d, want %d", tc.opType, got, tc.expected)
			}
		})
	}

	t.Run("unknown type defaults to zero value", func(t *testing.T) {
		if got := typeRank[OperationType("unknown")]; got != 0 {
			t.Errorf("typeRank[unknown] = %d, want 0", got)
		}
	})
}

func TestBadgeColorMapping(t *testing.T) {
	for _, opType := range []OperationType{TypeQuery, TypeMutation, TypeSubscription} {
		t.Run(string(opType), func(t *testing.T) {
			if _, ok := badgeColor[opType]; !ok {
				t.Errorf("badgeColor missing entry for %q", opType)
			}
		})
	}
}

func TestCollectEndpoints(t *testing.T) {
	t.Run("single endpoint", func(t *testing.T) {
		// collectEndpoints expects EndpointShort to be pre-populated (done by NewModel).
		ops := []UnifiedOperation{
			{Name: "a", EndpointShort: "api"},
			{Name: "b", EndpointShort: "api"},
		}
		eps := collectEndpoints(ops)
		if len(eps) != 1 {
			t.Errorf("expected 1 endpoint, got %d", len(eps))
		}
	})

	t.Run("multiple endpoints sorted", func(t *testing.T) {
		ops := []UnifiedOperation{
			{Name: "a", EndpointShort: "beta.example.com"},
			{Name: "b", EndpointShort: "alpha.example.com"},
		}
		eps := collectEndpoints(ops)
		if len(eps) != 2 {
			t.Fatalf("expected 2 endpoints, got %d", len(eps))
		}
		if !sort.StringsAreSorted(eps) {
			t.Errorf("endpoints should be sorted, got %v", eps)
		}
	})

	t.Run("empty operations", func(t *testing.T) {
		eps := collectEndpoints(nil)
		if len(eps) != 0 {
			t.Errorf("expected 0 endpoints, got %d", len(eps))
		}
	})
}

func TestFilterHintEndpoints(t *testing.T) {
	t.Run("single endpoint hides e: endpoints", func(t *testing.T) {
		m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		if strings.Contains(m.filterHint, "e: endpoints") {
			t.Error("should not show 'e: endpoints' with single endpoint")
		}
	})

	t.Run("multiple endpoints shows e: endpoints", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		if !strings.Contains(m.filterHint, "e: endpoints") {
			t.Errorf("should show 'e: endpoints' with multiple endpoints, got %q", m.filterHint)
		}
	})
}

func TestEndpointFilterCombinesWithTypeFilter(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.activeEndpoints = map[string]bool{
		"api.spacex.com": true,
	}
	m.search.Model.SetValue("q:")
	m.applyFilter()

	for _, op := range m.filtered {
		if op.Type != TypeQuery {
			t.Errorf("expected only queries, got %q (%s)", op.Name, op.Type)
		}
		if op.Endpoint != "https://api.spacex.com/graphql" {
			t.Errorf("expected spacex endpoint, got %q", op.Endpoint)
		}
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 results (getUser, listRockets), got %d", len(m.filtered))
	}
}

func TestEndpointFilterAlone(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.activeEndpoints = map[string]bool{
		"countries.trevorblades.com": true,
	}
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Fatalf("expected 2 results, got %d", len(m.filtered))
	}
	for _, op := range m.filtered {
		if op.Endpoint != "https://countries.trevorblades.com/graphql" {
			t.Errorf("expected countries endpoint, got %q", op.Endpoint)
		}
	}
}

func TestEndpointFilterMultipleSelected(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.activeEndpoints = map[string]bool{
		"api.spacex.com":             true,
		"countries.trevorblades.com": true,
	}
	m.applyFilter()

	if len(m.filtered) != len(m.operations) {
		t.Errorf("with all endpoints selected, expected %d, got %d",
			len(m.operations), len(m.filtered))
	}
}

func TestEndpointFilterEmptyRestoresAll(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.activeEndpoints = map[string]bool{}
	m.applyFilter()

	if len(m.filtered) != len(m.operations) {
		t.Errorf("empty endpoint filter should show all, expected %d, got %d",
			len(m.operations), len(m.filtered))
	}
}

func TestIsEndpointMode(t *testing.T) {
	t.Run("active on e:", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		m.search.Model.SetValue("e:")
		if !m.isEndpointMode() {
			t.Error("should be in endpoint mode with 'e:' prefix")
		}
	})

	t.Run("active on E:", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		m.search.Model.SetValue("E:")
		if !m.isEndpointMode() {
			t.Error("should be in endpoint mode with 'E:' prefix")
		}
	})

	t.Run("active after type prefix q:e:", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		m.search.Model.SetValue("q:e:")
		if !m.isEndpointMode() {
			t.Error("should be in endpoint mode with 'q:e:' prefix")
		}
	})

	t.Run("inactive with single endpoint", func(t *testing.T) {
		m := NewModel(sampleOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		m.search.Model.SetValue("e:")
		if m.isEndpointMode() {
			t.Error("should not be in endpoint mode with single endpoint")
		}
	})

	t.Run("inactive on plain text", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
		m.search.Model.SetValue("get")
		if m.isEndpointMode() {
			t.Error("should not be in endpoint mode on plain text")
		}
	})
}

func TestEndpointSearchTerm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"just e:", "e:", ""},
		{"e: with term", "e:space", "space"},
		{"e: with uppercase term", "e:SPACE", "space"},
		{"q:e: with term", "q:e:country", "country"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
			m.search.Model.SetValue(tc.input)
			got := m.endpointSearchTerm()
			if got != tc.expected {
				t.Errorf("endpointSearchTerm() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestFilteredEndpoints(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))

	t.Run("no filter returns all", func(t *testing.T) {
		m.search.Model.SetValue("e:")
		eps := m.filteredEndpoints()
		if len(eps) != len(m.endpoints) {
			t.Errorf("expected %d endpoints, got %d", len(m.endpoints), len(eps))
		}
	})

	t.Run("filter narrows list", func(t *testing.T) {
		m.search.Model.SetValue("e:space")
		eps := m.filteredEndpoints()
		if len(eps) != 1 {
			t.Fatalf("expected 1 endpoint matching 'space', got %d", len(eps))
		}
		if !strings.Contains(eps[0], "spacex") {
			t.Errorf("expected spacex endpoint, got %q", eps[0])
		}
	})
}

func TestEndpointToggle(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:")
	ep := m.filteredEndpoints()[0]

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	if !m.activeEndpoints[ep] {
		t.Fatal("precondition: all endpoints start active")
	}

	result, _ := m.Update(enterKey)
	model := result.(*Model)
	if model.activeEndpoints[ep] {
		t.Errorf("expected endpoint %q to be toggled off", ep)
	}

	result, _ = model.Update(enterKey)
	model = result.(*Model)
	if !model.activeEndpoints[ep] {
		t.Errorf("expected endpoint %q to be toggled back on", ep)
	}
}

func TestEndpointEnterToggle(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:")
	ep := m.filteredEndpoints()[0]

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	result, _ := m.Update(enterKey)
	model := result.(*Model)
	if model.activeEndpoints[ep] {
		t.Errorf("enter should toggle endpoint %q off", ep)
	}
}

func TestEndpointNavigation(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := result.(*Model)
	if model.endpointCursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.endpointCursor)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = result.(*Model)
	if model.endpointCursor != 0 {
		t.Errorf("expected cursor 0, got %d", model.endpointCursor)
	}
}

func TestEndpointCtrlNavigation(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	model := result.(*Model)
	if model.endpointCursor != 1 {
		t.Errorf("ctrl+n should move down, expected cursor 1, got %d", model.endpointCursor)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model = result.(*Model)
	if model.endpointCursor != 0 {
		t.Errorf("ctrl+p should move up, expected cursor 0, got %d", model.endpointCursor)
	}
}

func TestShortenEndpoint(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://api.spacex.com/graphql", "api.spacex.com"},
		{"http://localhost:4000/graphql", "localhost:4000"},
		{"https://countries.trevorblades.com/gql", "countries.trevorblades.com"},
		{"https://example.com/api/v2", "example.com/api/v2"},
		{"http://api/gql", "api"},
		{"https://api.spacex.com/graphql?token=123", "api.spacex.com"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := shortenEndpoint(tc.input)
			if got != tc.expected {
				t.Errorf("shortenEndpoint(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRenderEndpointPicker(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:")

	content, _ := m.renderEndpointPicker()

	for _, ep := range m.endpoints {
		if !strings.Contains(content, ep) {
			t.Errorf("picker should contain endpoint %q", ep)
		}
	}
	if !strings.Contains(content, utils.ChevronDownCircled) {
		t.Error("picker should show chevron for cursor endpoint")
	}
	if !strings.Contains(content, utils.CrossMark) {
		t.Error("picker should show toggle mark for active endpoints")
	}
}

func TestEndpointCursorResetsOnSearchChange(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:")
	m.endpointCursor = 1

	m.search.Model.SetValue("e:space")
	m.endpointCursor = 0
	m.applyFilterAndReset()

	eps := m.filteredEndpoints()
	if m.endpointCursor >= len(eps) && len(eps) > 0 {
		t.Error("cursor should be clamped after filtering narrows the list")
	}
}

func TestNegatedEndpointSearch(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:!space")

	if !m.isNegatedEndpointSearch() {
		t.Error("should detect negated search")
	}

	eps := m.filteredEndpoints()
	if len(eps) != 1 {
		t.Fatalf("expected 1 endpoint matching 'space', got %d", len(eps))
	}
	if !strings.Contains(eps[0], "spacex") {
		t.Errorf("expected spacex endpoint, got %q", eps[0])
	}
}

func TestNegatedEndpointEnterKeepsOnlyMatches(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:!space")

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(enterKey)
	model := result.(*Model)

	if len(model.activeEndpoints) != 1 {
		t.Fatalf("expected 1 active endpoint, got %d", len(model.activeEndpoints))
	}
	for ep := range model.activeEndpoints {
		if !strings.Contains(ep, "spacex") {
			t.Errorf("expected only spacex to remain active, got %q", ep)
		}
	}
}

func TestNonNegatedSearchIgnoresBang(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil, nil, nil, nil, make(map[string]yamlparser.APIInfo))
	m.search.Model.SetValue("e:space")

	if m.isNegatedEndpointSearch() {
		t.Error("should not detect negation without ! prefix")
	}
}

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
	if !strings.Contains(view, "Send") || !strings.Contains(view, "ctrl+g") {
		t.Fatalf("expected send action in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Save Query") || !strings.Contains(view, "ctrl+q") {
		t.Fatalf("expected save query action in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Save Request") || !strings.Contains(view, "ctrl+h") {
		t.Fatalf("expected save request action in view, got:\n%s", view)
	}
}

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
