package gqlexplorer

import (
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

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
