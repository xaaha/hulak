package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

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
