package gqlexplorer

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
)

func TestArgFormItemEnabledDefaults(t *testing.T) {
	tests := []struct {
		name   string
		arg    graphql.Argument
		wantOn bool
	}{
		{"required string", graphql.Argument{Name: "id", Type: "ID!"}, true},
		{"optional string", graphql.Argument{Name: "search", Type: "String"}, false},
		{"required bool", graphql.Argument{Name: "active", Type: "Boolean!"}, true},
		{"optional bool", graphql.Argument{Name: "verbose", Type: "Boolean"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fi := newArgFormItem(tc.arg, nil, "ep")
			if fi.enabled != tc.wantOn {
				t.Errorf("enabled = %v, want %v", fi.enabled, tc.wantOn)
			}
		})
	}
}

func TestOptionalInputTypeFieldsStartDisabled(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{
			{Name: "requiredArg", Type: "ReqInput!"},
			{Name: "optionalArg", Type: "OptInput"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		ScopedTypeKey(ep, "ReqInput"): {
			Name: "ReqInput",
			Fields: []graphql.InputField{
				{Name: "field1", Type: "String!"},
				{Name: "field2", Type: "String"},
			},
		},
		ScopedTypeKey(ep, "OptInput"): {
			Name: "OptInput",
			Fields: []graphql.InputField{
				{Name: "field3", Type: "String!"},
				{Name: "field4", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
	}
	if df.argCount != 4 {
		t.Fatalf("expected 4 arg items, got %d", df.argCount)
	}

	// Required parent arg: sub-field enabled follows its own required status.
	if !df.items[0].enabled {
		t.Error("required field inside required arg should start enabled")
	}
	if df.items[1].enabled {
		t.Error("optional field inside required arg should start disabled")
	}

	// Optional parent arg: all sub-fields start disabled regardless.
	if df.items[2].enabled {
		t.Error("required field inside optional arg should start disabled")
	}
	if df.items[3].enabled {
		t.Error("optional field inside optional arg should start disabled")
	}

	// The required flag itself should still be set (for UI hints like asterisks).
	if !df.items[2].required {
		t.Error("required field inside optional arg should still have required=true")
	}
}

func TestBuildDetailFormSetsArgName(t *testing.T) {
	ep := "https://api.test/graphql"
	op := &UnifiedOperation{
		Name:     "Search",
		Type:     TypeQuery,
		Endpoint: ep,
		Arguments: []graphql.Argument{
			{Name: "id", Type: "Int!"},
			{Name: "filter", Type: "FilterInput"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		ScopedTypeKey(ep, "FilterInput"): {
			Name: "FilterInput",
			Fields: []graphql.InputField{
				{Name: "keyword", Type: "String"},
				{Name: "category", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
	}
	if df.argCount != 3 {
		t.Fatalf("expected 3 arg items (1 simple + 2 expanded), got %d", df.argCount)
	}
	if df.items[0].argName != "id" {
		t.Errorf("item 0 argName = %q, want %q", df.items[0].argName, "id")
	}
	if df.items[1].argName != "filter" {
		t.Errorf("item 1 argName = %q, want %q", df.items[1].argName, "filter")
	}
	if df.items[2].argName != "filter" {
		t.Errorf("item 2 argName = %q, want %q", df.items[2].argName, "filter")
	}
}

func TestSpaceTogglesEnabledOnTextInput(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	df.FocusCurrent()

	if df.items[0].enabled {
		t.Fatal("optional arg should start disabled")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !df.items[0].enabled {
		t.Fatal("Space should enable the arg")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if df.items[0].enabled {
		t.Fatal("second Space should disable the arg")
	}
}

func TestSpaceTogglesBooleanArgEnabled(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "active", Type: "Boolean"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	df.FocusCurrent()

	if df.items[0].enabled {
		t.Fatal("optional bool should start disabled")
	}
	if df.items[0].toggle.Value {
		t.Fatal("toggle value should start false")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !df.items[0].enabled {
		t.Fatal("Space should enable boolean arg")
	}
	if !df.items[0].toggle.Value {
		t.Fatal("toggle value should be true after Space")
	}
}

func TestSpaceOnExpandedInputTypeTogglesOnlyTargetField(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Search", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{
			{Name: "filter", Type: "FilterInput"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		ScopedTypeKey(ep, "FilterInput"): {
			Name: "FilterInput",
			Fields: []graphql.InputField{
				{Name: "keyword", Type: "String"},
				{Name: "category", Type: "String"},
				{Name: "active", Type: "Boolean"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
	}
	if df.argCount != 3 {
		t.Fatalf("expected 3 arg items, got %d", df.argCount)
	}

	// All optional fields should start disabled.
	for i := 0; i < df.argCount; i++ {
		if df.items[i].enabled {
			t.Fatalf("item %d (%s) should start disabled", i, df.items[i].name)
		}
	}

	// Toggle "keyword" (item 0) with Space — only it should become enabled.
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !df.items[0].enabled {
		t.Fatal("keyword should be enabled after Space")
	}
	if df.items[1].enabled {
		t.Fatal("category should remain disabled when keyword is toggled")
	}
	if df.items[2].enabled {
		t.Fatal("active should remain disabled when keyword is toggled")
	}

	// Move to "active" (boolean toggle, item 2) and toggle it.
	df.CursorDown()
	df.CursorDown()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !df.items[2].enabled {
		t.Fatal("active should be enabled after Space")
	}
	if df.items[1].enabled {
		t.Fatal("category should still be disabled")
	}

	// Toggle "keyword" off again.
	df.CursorUp()
	df.CursorUp()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if df.items[0].enabled {
		t.Fatal("keyword should be disabled after second Space")
	}
	if !df.items[2].enabled {
		t.Fatal("active should remain enabled after keyword is toggled off")
	}
}

func TestEnterTogglesTextInputEditing(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String!"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	df.FocusCurrent()

	if df.items[0].input.Model.Focused() {
		t.Fatal("text input should not be focused initially")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !df.items[0].input.Model.Focused() {
		t.Fatal("Enter should activate editing")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if df.items[0].input.Model.Focused() {
		t.Fatal("second Enter should deactivate editing")
	}
}
