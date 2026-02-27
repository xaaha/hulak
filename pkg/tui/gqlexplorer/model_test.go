package gqlexplorer

import (
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/utils"
)

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
	m := NewModel(ops, nil, nil)

	expected := []OperationType{TypeQuery, TypeMutation, TypeSubscription}
	for i, want := range expected {
		if m.filtered[i].Type != want {
			t.Errorf("index %d: expected type %q, got %q", i, want, m.filtered[i].Type)
		}
	}
}

func TestNewModelEmptyOperations(t *testing.T) {
	m := NewModel(nil, nil, nil)

	if len(m.operations) != 0 {
		t.Errorf("expected 0 operations, got %d", len(m.operations))
	}
	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered, got %d", len(m.filtered))
	}
}

func TestNewModelFilteredMatchesOperations(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	if len(m.filtered) != len(m.operations) {
		t.Errorf("expected filtered (%d) to match operations (%d)",
			len(m.filtered), len(m.operations))
	}
}

func TestNewModelCursorStartsAtZero(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestInitReturnsCmd(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a blink command")
	}
}

func TestNavigateDown(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := result.(*Model)

	if model.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.cursor = 2

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := result.(*Model)

	if model.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.cursor)
	}
}

func TestNavigateCtrlN(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	model := result.(*Model)

	if model.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", model.cursor)
	}
}

func TestNavigateCtrlP(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.cursor = 3

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	model := result.(*Model)

	if model.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", model.cursor)
	}
}

func TestTabTogglesFocus(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := result.(*Model)
	if model.focusedPanel != focusRight {
		t.Errorf("expected focusRight after tab, got %v", model.focusedPanel)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(*Model)
	if model.focusedPanel != focusLeft {
		t.Errorf("expected focusLeft after second tab, got %v", model.focusedPanel)
	}
}

func TestEnterMovesFocusToDetailOnly(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model := result.(*Model)
	model.focusedPanel = focusLeft

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	if model.focusedPanel != focusRight {
		t.Errorf("expected focusRight after enter, got %v", model.focusedPanel)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	if model.focusedPanel != focusRight {
		t.Errorf("expected focusRight to remain after second enter, got %v", model.focusedPanel)
	}
}

func TestActiveScrollPanelForcesLeftInEndpointPicker(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.focusedPanel = focusRight
	m.pickingEndpoints = true

	if got := m.activeScrollPanel(); got != focusLeft {
		t.Errorf("expected active scroll panel focusLeft in endpoint picker, got %v", got)
	}
}

func TestNavigateUpAtTopStays(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := result.(*Model)

	if model.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", model.cursor)
	}
}

func TestNavigateDownAtBottomStays(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.cursor = len(m.filtered) - 1

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := result.(*Model)

	if model.cursor != len(m.filtered)-1 {
		t.Errorf("expected cursor %d, got %d", len(m.filtered)-1, model.cursor)
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Error("expected quit command from ctrl+c")
	}
}

func TestEscQuitsWhenSearchEmpty(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Error("expected quit command from esc with empty search")
	}
}

func TestEscClearsSearchFirst(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)

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
	m := NewModel(sampleOps(), nil, nil)

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
	m := NewModel(sampleOps(), nil, nil)
	m.search.Model.SetValue("zzzzz")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.filtered))
	}
}

func TestFilterEmptyRestoresAll(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
	m.cursor = 4

	m.search.Model.SetValue("getUser")
	m.applyFilter()

	if m.cursor >= len(m.filtered) {
		t.Errorf("cursor %d should be < filtered length %d",
			m.cursor, len(m.filtered))
	}
}

func TestFilterByTypeQueryPrefix(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
	m.search.Model.SetValue("Q:")
	m.applyFilter()

	for _, op := range m.filtered {
		if op.Type != TypeQuery {
			t.Errorf("expected only queries with 'Q:', got %q (%s)", op.Name, op.Type)
		}
	}
}

func TestFilterByTypePrefixWithNameSearch(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
	m.search.Model.SetValue("q:zzz")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches for 'q:zzz', got %d", len(m.filtered))
	}
}

func TestFilterUnknownPrefixTreatedAsPlainSearch(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.search.Model.SetValue("x:foo")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches for 'x:foo', got %d", len(m.filtered))
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)

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
	m := NewModel(multiEndpointOps(), nil, nil)

	result, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 40})
	model := result.(*Model)

	if model.search.Model.Placeholder != "" {
		t.Errorf("expected empty placeholder below threshold, got %q", model.search.Model.Placeholder)
	}
	if model.badgeCache != "" {
		t.Errorf("expected empty badge cache below threshold, got %q", model.badgeCache)
	}
}

func TestWindowSizeMsgShowsHeaderExtrasAtThreshold(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)

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
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "Search") {
		t.Error("view should contain search prompt")
	}
}

func TestViewContainsFilterHint(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
			m := NewModel(tc.ops, nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "5/5 operations") {
		t.Errorf("view should contain '5/5 operations', got:\n%s", view)
	}
}

func TestViewContainsOperationNames(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "esc: quit") {
		t.Error("view should contain help text")
	}
}

func TestViewShowsNoMatchesWhenFilteredEmpty(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
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
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, utils.ChevronRight) {
		t.Error("view should contain chevron cursor marker for selected item")
	}
}

func TestViewShowsDescriptionForSelectedItem(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "fetch user") {
		t.Error("view should show description for the selected item")
	}
}

func TestViewShowsEndpointForSelectedItem(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "http://api/gql") {
		t.Error("view should show endpoint for the selected item")
	}
}

func TestViewHasBorder(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	view := m.View()

	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Error("view should have rounded border characters")
	}
}

func TestViewFilteredCountUpdates(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	m.width = 160
	m.height = 40
	m.search.Model.SetValue("q:")
	m.applyFilter()
	view := m.View()

	if !strings.Contains(view, "2/5 operations") {
		t.Errorf("view should contain '2/5 operations' after filtering, got:\n%s", view)
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
		m := NewModel(sampleOps(), nil, nil)
		if strings.Contains(m.filterHint, "e: endpoints") {
			t.Error("should not show 'e: endpoints' with single endpoint")
		}
	})

	t.Run("multiple endpoints shows e: endpoints", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil)
		if !strings.Contains(m.filterHint, "e: endpoints") {
			t.Errorf("should show 'e: endpoints' with multiple endpoints, got %q", m.filterHint)
		}
	})
}

func TestEndpointFilterCombinesWithTypeFilter(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
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
	m := NewModel(multiEndpointOps(), nil, nil)
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
	m := NewModel(multiEndpointOps(), nil, nil)
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
	m := NewModel(multiEndpointOps(), nil, nil)
	m.activeEndpoints = map[string]bool{}
	m.applyFilter()

	if len(m.filtered) != len(m.operations) {
		t.Errorf("empty endpoint filter should show all, expected %d, got %d",
			len(m.operations), len(m.filtered))
	}
}

func TestEnterEndpointPicker(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	m.enterEndpointPicker()

	if !m.pickingEndpoints {
		t.Error("expected pickingEndpoints to be true")
	}
	if m.endpointCursor != 0 {
		t.Errorf("expected endpointCursor 0, got %d", m.endpointCursor)
	}
	if m.pendingEndpoints == nil {
		t.Error("expected pendingEndpoints to be initialized")
	}
}

func TestEndpointPickerToggle(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	m.enterEndpointPicker()

	ep := m.endpoints[0]
	spaceKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}

	// all endpoints start selected, first toggle turns off
	result, _ := m.Update(spaceKey)
	model := result.(*Model)
	if model.pendingEndpoints[ep] {
		t.Errorf("expected endpoint %q to be toggled off", ep)
	}

	// second toggle turns back on
	result, _ = model.Update(spaceKey)
	model = result.(*Model)
	if !model.pendingEndpoints[ep] {
		t.Errorf("expected endpoint %q to be toggled on", ep)
	}
}

func TestEndpointPickerConfirm(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	m.enterEndpointPicker()
	m.pendingEndpoints[m.endpoints[0]] = true

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := result.(*Model)

	if model.pickingEndpoints {
		t.Error("expected picker to close on enter")
	}
	if !model.activeEndpoints[model.endpoints[0]] {
		t.Error("expected confirmed endpoint to be active")
	}
}

func TestEndpointPickerCancel(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	// deselect one endpoint before opening picker
	delete(m.activeEndpoints, m.endpoints[1])
	originalCount := len(m.activeEndpoints)

	m.enterEndpointPicker()
	// toggle an extra endpoint in pending
	m.pendingEndpoints[m.endpoints[1]] = true

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := result.(*Model)

	if model.pickingEndpoints {
		t.Error("expected picker to close on esc")
	}
	if len(model.activeEndpoints) != originalCount {
		t.Error("cancel should preserve original active endpoints")
	}
	if model.activeEndpoints[m.endpoints[1]] {
		t.Error("cancel should not apply pending changes")
	}
}

func TestEndpointPickerNavigation(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	m.enterEndpointPicker()

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

func TestEndpointPickerVimNavigation(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	m.enterEndpointPicker()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := result.(*Model)
	if model.endpointCursor != 1 {
		t.Errorf("j should move down, expected cursor 1, got %d", model.endpointCursor)
	}

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = result.(*Model)
	if model.endpointCursor != 0 {
		t.Errorf("k should move up, expected cursor 0, got %d", model.endpointCursor)
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

func TestShouldEnterEndpointPicker(t *testing.T) {
	t.Run("triggers on e:", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil)
		if !m.shouldEnterEndpointPicker("e:") {
			t.Error("should trigger on 'e:'")
		}
	})

	t.Run("triggers on E:", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil)
		if !m.shouldEnterEndpointPicker("E:") {
			t.Error("should trigger on 'E:'")
		}
	})

	t.Run("triggers after type prefix q:e:", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil)
		if !m.shouldEnterEndpointPicker("q:e:") {
			t.Error("should trigger on 'q:e:'")
		}
	})

	t.Run("no trigger with single endpoint", func(t *testing.T) {
		m := NewModel(sampleOps(), nil, nil)
		if m.shouldEnterEndpointPicker("e:") {
			t.Error("should not trigger with single endpoint")
		}
	})

	t.Run("no trigger on plain text", func(t *testing.T) {
		m := NewModel(multiEndpointOps(), nil, nil)
		if m.shouldEnterEndpointPicker("get") {
			t.Error("should not trigger on plain text")
		}
	})
}

func TestStripEndpointPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"just e:", "e:", ""},
		{"type then e:", "q:e:", "q:"},
		{"text then e:", "hello e:", "hello"},
		{"no e:", "hello", "hello"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(multiEndpointOps(), nil, nil)
			m.search.Model.SetValue(tc.input)
			m.stripEndpointPrefix()
			got := m.search.Model.Value()
			if got != tc.expected {
				t.Errorf("stripEndpointPrefix(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRenderEndpointPicker(t *testing.T) {
	m := NewModel(multiEndpointOps(), nil, nil)
	m.enterEndpointPicker()
	m.pendingEndpoints[m.endpoints[0]] = true

	content, _ := m.renderEndpointPicker()

	for _, ep := range m.endpoints {
		if !strings.Contains(content, ep) {
			t.Errorf("picker should contain endpoint %q", ep)
		}
	}
	if !strings.Contains(content, checkMark) {
		t.Error("picker should show check mark for selected endpoint")
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
	detail := renderDetail(&op, nil)
	if !strings.Contains(detail, "getUser") {
		t.Error("detail should contain operation name")
	}
}

func TestRenderDetailShowsReturnType(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil)
	if !strings.Contains(detail, "User!") {
		t.Error("detail should contain return type")
	}
}

func TestRenderDetailShowsArguments(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil)
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
	detail := renderDetail(&op, nil)
	if strings.Contains(detail, "Endpoint:") {
		t.Error("detail should not show Endpoint (already in badges and list)")
	}
}

func TestRenderDetailNoArgsOmitsSection(t *testing.T) {
	op := opsWithArgs()[1]
	detail := renderDetail(&op, nil)
	if strings.Contains(detail, "Arguments:") {
		t.Error("detail should not show Arguments section when empty")
	}
}

func TestRenderDetailOptionalArgHasNoRequiredMarker(t *testing.T) {
	op := opsWithArgs()[0]
	detail := renderDetail(&op, nil)
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
	m := NewModel(opsWithArgs(), nil, nil)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = *result.(*Model)
	view := m.View()

	if !strings.Contains(view, "Returns:") {
		t.Error("view should show detail panel with return type")
	}
}

func TestDetailPanelUpdatesOnCursorMove(t *testing.T) {
	m := NewModel(opsWithArgs(), nil, nil)
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
	detail := renderDetail(&op, inputTypes)
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
	detail := renderDetail(&op, inputTypes)
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
	detail := renderDetail(&op, nil)
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
		// containerStyle vertical frame = 2 (top+bottom border),
		// DetailTopHeight = 40%.
		// contentH = max(height-2, 1), top = max(contentH*40/100, 1)
		{"typical terminal", 40, 15},
		{"small terminal", 10, 3},
		{"minimum size", 5, 1},
		{"zero height", 0, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{height: tc.height}
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
		want   int
	}{
		{"typical terminal", 40, 23},
		{"small terminal", 10, 5},
		{"zero height", 0, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{height: tc.height}
			got := m.responseAreaHeight()
			if got != tc.want {
				t.Errorf("responseAreaHeight() = %d, want %d", got, tc.want)
			}
			if got < 1 {
				t.Errorf("responseAreaHeight() = %d, must be >= 1", got)
			}
		})
	}
}

func TestHeightPartitionSumsCorrectly(t *testing.T) {
	for h := 0; h <= 100; h++ {
		m := Model{height: h}
		total := m.contentHeight()
		top := m.detailTopHeight()
		bottom := m.responseAreaHeight()
		sum := top + bottom

		// For very small heights where max() clamps to 1, the sum may exceed
		// total. For normal heights the partition should be exact.
		if total >= 2 && sum != total {
			t.Errorf("height=%d: top(%d) + bottom(%d) = %d, want %d",
				h, top, bottom, sum, total)
		}
	}
}

func TestEnterNoFocusChangeInSinglePanel(t *testing.T) {
	m := NewModel(sampleOps(), nil, nil)
	// Width 50 → contentWidth = 50-4 = 46 < MinLeftPanelWidth+MinRightPanelWidth (58)
	// so hasTwoPanelLayout() returns false.
	result, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 40})
	model := result.(*Model)
	model.focusedPanel = focusLeft

	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(*Model)
	if model.focusedPanel != focusLeft {
		t.Errorf("expected focusLeft in single-panel layout after enter, got %v", model.focusedPanel)
	}
}
