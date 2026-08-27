package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/utils"
)

func TestCheckboxPrefixInView(t *testing.T) {
	fi := newArgFormItem(graphql.Argument{Name: "q", Type: "String"}, nil, "ep")
	v := fi.View()
	if !strings.Contains(v, "[") || !strings.Contains(v, "]") {
		t.Fatal("non-field text input should have checkbox brackets in view")
	}
}

func TestContinuationListInputViewShowsConnectorWithoutCheckbox(t *testing.T) {
	fi := newListArgFormItems(graphql.Argument{Name: "ids", Type: "[ID!]!"}, nil, nil, "ep", 1, true, nil)[0]
	v := fi.View()
	if strings.Contains(v, "[") || strings.Contains(v, "]") {
		t.Fatal("continuation list input should not render a checkbox prefix")
	}
	if !strings.Contains(v, utils.Connector) {
		t.Fatal("continuation list input should render a connector")
	}
}

func TestEnabledArgNames(t *testing.T) {
	df := &DetailForm{
		argCount: 4,
		items: []formItem{
			{name: "a", argName: "a", enabled: true},
			{name: "b", argName: "b", enabled: false},
			{name: "kw", argName: "filter", enabled: true},
			{name: "cat", argName: "filter", enabled: false},
			{name: "field1", isField: true},
		},
	}
	got := df.enabledArgNames()
	if !got["a"] {
		t.Error("a should be enabled")
	}
	if got["b"] {
		t.Error("b should not be enabled")
	}
	if !got["filter"] {
		t.Error("filter should be enabled (kw child is enabled)")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 enabled args, got %d", len(got))
	}
}

func TestDetailFormCursorToTopBottom(t *testing.T) {
	ep := "https://api.test/graphql"
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   ep,
		Arguments: []graphql.Argument{
			{Name: "code", Type: "ID!"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "name", Type: "String"},
				{Name: "capital", Type: "String"},
				{Name: "phone", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)
	last := len(df.items) - 1

	df.CursorToBottom()
	if df.cursor != last {
		t.Errorf("CursorToBottom: cursor = %d, want %d", df.cursor, last)
	}
	if !df.items[last].Focused() {
		t.Error("last item should be focused after CursorToBottom")
	}

	df.CursorToTop()
	if df.cursor != 0 {
		t.Error("CursorToTop: cursor should be 0")
	}
	if !df.items[0].Focused() {
		t.Error("first item should be focused after CursorToTop")
	}
	if df.items[last].Focused() {
		t.Error("last item should be blurred after CursorToTop")
	}
}

func searchFormHelper() *DetailForm {
	op := &UnifiedOperation{
		Name:       "findUsers",
		Type:       TypeQuery,
		Endpoint:   "http://api/gql",
		ReturnType: "UserConnection!",
		Arguments: []graphql.Argument{
			{Name: "firstName", Type: "String"},
			{Name: "lastName", Type: "String"},
			{Name: "email", Type: "String!"},
			{Name: "age", Type: "Int"},
			{Name: "country", Type: "String"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		"UserConnection": {Name: "UserConnection", Fields: []graphql.ObjectField{
			{Name: "totalCount", Type: "Int!"},
			{Name: "edges", Type: "[UserEdge]"},
		}},
	}
	return buildDetailForm(op, nil, nil, objectTypes, nil, nil)
}

func TestSearchStartAndStop(t *testing.T) {
	df := searchFormHelper()
	if df.IsSearching() {
		t.Fatal("should not be searching initially")
	}

	df.cursor = 2
	df.FocusCurrent()
	df.StartSearch()

	if !df.IsSearching() {
		t.Fatal("should be searching after StartSearch")
	}
	if df.preSearchCursor != 2 {
		t.Errorf("preSearchCursor = %d, want 2", df.preSearchCursor)
	}

	df.StopSearch(false)
	if df.IsSearching() {
		t.Fatal("should not be searching after StopSearch")
	}
	if df.cursor != 2 {
		t.Errorf("cursor should revert to %d on cancel, got %d", 2, df.cursor)
	}
}

func TestSearchConfirmKeepsCursor(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.search.SetQuery("email")
	df.updateSearchMatches()

	matched := df.cursor
	df.StopSearch(true)

	if df.cursor != matched {
		t.Errorf("cursor should stay at %d after confirm, got %d", matched, df.cursor)
	}
}

func TestSearchMatchesByName(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.search.SetQuery("name")
	df.updateSearchMatches()

	if df.search.MatchCount() != 2 {
		t.Fatalf(
			"expected 2 matches for 'name' (firstName, lastName), got %d",
			df.search.MatchCount(),
		)
	}
	if df.cursor != df.search.CurrentMatch() {
		t.Errorf("cursor should be at first match %d, got %d", df.search.CurrentMatch(), df.cursor)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.search.SetQuery("EMAIL")
	df.updateSearchMatches()

	if df.search.MatchCount() == 0 {
		t.Fatal("expected match for uppercase 'EMAIL'")
	}
}

func TestSearchNoMatches(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.search.SetQuery("zzz")
	df.updateSearchMatches()

	if df.search.MatchCount() != 0 {
		t.Fatalf("expected 0 matches, got %d", df.search.MatchCount())
	}
	footer := df.SearchFooter()
	if !strings.Contains(footer, "no matches") {
		t.Errorf("footer = %q, want to contain 'no matches'", footer)
	}
}

func TestSearchNextPrevMatch(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.search.SetQuery("name")
	df.updateSearchMatches()
	if df.search.MatchCount() < 2 {
		t.Fatal("need at least 2 matches")
	}

	first := df.cursor

	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyDown})
	second := df.cursor
	if second == first {
		t.Error("Down should cycle to next match")
	}

	footer := df.SearchFooter()
	if !strings.Contains(footer, "2/2") {
		t.Errorf("footer = %q, want to contain '2/2'", footer)
	}

	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyDown})
	if df.cursor != first {
		t.Error("nextMatch should wrap around to first match")
	}

	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyUp})
	if df.cursor != second {
		t.Error("prevMatch should wrap around to last match")
	}
}

func TestSearchHandleKeyEnterConfirms(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()
	df.search.SetQuery("email")
	df.updateSearchMatches()

	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyEnter})

	if df.IsSearching() {
		t.Fatal("Enter should close search")
	}
}

func TestSearchHandleKeyEscCancels(t *testing.T) {
	df := searchFormHelper()
	original := df.cursor
	df.StartSearch()
	df.search.SetQuery("email")
	df.updateSearchMatches()

	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyEscape})

	if df.IsSearching() {
		t.Fatal("Esc should close search")
	}
	if df.cursor != original {
		t.Errorf("Esc should revert cursor to %d, got %d", original, df.cursor)
	}
}

func TestSearchHandleKeyArrowsCycleMatches(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()
	df.search.SetQuery("name")
	df.updateSearchMatches()

	first := df.cursor
	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyDown})
	if df.cursor == first {
		t.Error("Down arrow should cycle to next match")
	}

	df.HandleSearchKey(tea.KeyMsg{Type: tea.KeyUp})
	if df.cursor != first {
		t.Error("Up arrow should cycle back to first match")
	}
}

func TestSearchFooterRendering(t *testing.T) {
	df := searchFormHelper()
	if df.SearchFooter() != "" {
		t.Fatal("footer should be empty when not searching")
	}

	df.StartSearch()
	footer := df.SearchFooter()
	if !strings.Contains(footer, "Search(/)") {
		t.Fatalf("footer should contain label, got %q", footer)
	}

	df.search.SetQuery("name")
	df.updateSearchMatches()
	footer = df.SearchFooter()
	if !strings.Contains(footer, "1/2") {
		t.Fatalf("footer should show match count, got %q", footer)
	}

	df.StopSearch(true)
	if df.SearchFooter() != "" {
		t.Fatal("footer should be empty after confirming search")
	}
}

func TestMatchesClearedAfterConfirm(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()
	df.search.SetQuery("name")
	df.updateSearchMatches()

	df.StopSearch(true)
	if df.search.MatchCount() != 0 {
		t.Fatal("matches should be cleared after confirm")
	}
}

func TestMatchesClearedAfterCancel(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()
	df.search.SetQuery("name")
	df.updateSearchMatches()

	df.StopSearch(false)
	if df.search.MatchCount() != 0 {
		t.Fatal("matches should be cleared after cancel")
	}
}

func TestEnterTogglesBooleanArgument(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "setFlag",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "isAdult", Type: "Boolean"},
		},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	df.FocusCurrent()

	item := &df.items[0]
	if item.kind != formItemToggle {
		t.Fatalf("expected toggle, got %d", item.kind)
	}
	if item.enabled {
		t.Fatal("optional boolean should start disabled")
	}

	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !item.toggle.Value {
		t.Error("Enter should toggle the boolean value to true")
	}
	if !item.enabled {
		t.Error("Enter should enable the argument")
	}

	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if item.toggle.Value {
		t.Error("second Enter should toggle back to false")
	}
	if item.enabled {
		t.Error("second Enter should disable the argument")
	}
}

func TestEnterExpandsRecursiveToggleLikeSpace(t *testing.T) {
	ep := "ep"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "name", Type: "String!"},
				{Name: "countries", Type: "[Country!]!"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)

	df.cursor = 1
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if df.Len() != 4 {
		t.Fatalf("expected Enter to expand children like Space, got %d items", df.Len())
	}

	nestedCountries := 3
	if !df.items[nestedCountries].expandable {
		t.Fatal("nested 'countries' should also be expandable")
	}

	df.cursor = nestedCountries
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if df.Len() != 6 {
		t.Fatalf("expected Enter to recursively expand children like Space, got %d items", df.Len())
	}
}
