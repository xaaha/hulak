package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func TestNewFieldFormItemCreatesToggle(t *testing.T) {
	f := newFieldFormItem(graphql.ObjectField{Name: "code", Type: "ID!"}, true)
	if f.kind != formItemToggle {
		t.Fatalf("expected toggle kind, got %d", f.kind)
	}
	if !f.isField {
		t.Fatal("expected isField=true for field form item")
	}
	if f.Value() != "true" {
		t.Fatal("expected initial value true for selected field")
	}
}

func TestNewFieldFormItemUnselected(t *testing.T) {
	f := newFieldFormItem(graphql.ObjectField{Name: "name", Type: "String"}, false)
	if f.Value() != "false" {
		t.Fatal("expected initial value false for unselected field")
	}
}

func TestNewArgFormItemBoolean(t *testing.T) {
	f := newArgFormItem(
		graphql.Argument{Name: "active", Type: "Boolean!"},
		nil,
		"https://api.test/graphql",
	)
	if f.kind != formItemToggle {
		t.Fatalf("expected toggle for Boolean, got %d", f.kind)
	}
	if f.isField {
		t.Fatal("expected isField=false for argument")
	}
	if !f.required {
		t.Fatal("expected required=true for Boolean!")
	}
}

func TestNewArgFormItemEnum(t *testing.T) {
	ep := "https://api.test/graphql"
	enums := map[string]graphql.EnumType{
		ScopedTypeKey(ep, "Status"): {
			Name:   "Status",
			Values: []graphql.EnumValue{{Name: "ACTIVE"}, {Name: "INACTIVE"}},
		},
	}
	f := newArgFormItem(
		graphql.Argument{Name: "status", Type: "Status"},
		enums,
		ep,
	)
	if f.kind != formItemDropdown {
		t.Fatalf("expected dropdown for enum, got %d", f.kind)
	}
	if f.Value() != "ACTIVE" {
		t.Fatalf("expected first enum value as default, got %q", f.Value())
	}
}

func TestNewArgFormItemTextInput(t *testing.T) {
	f := newArgFormItem(
		graphql.Argument{Name: "name", Type: "String!"},
		nil,
		"https://api.test/graphql",
	)
	if f.kind != formItemTextInput {
		t.Fatalf("expected text input for String, got %d", f.kind)
	}
	if !f.required {
		t.Fatal("expected required=true for String!")
	}
	if f.input.Model.Focused() {
		t.Fatal("text input should start blurred")
	}
}

func TestFormItemFocusBlur(t *testing.T) {
	tests := []struct {
		name string
		item formItem
	}{
		{"toggle", newFieldFormItem(graphql.ObjectField{Name: "a", Type: "String"}, true)},
		{
			"textInput",
			newArgFormItem(graphql.Argument{Name: "b", Type: "String"}, nil, "ep"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.item.Focused() {
				t.Fatal("should start unfocused")
			}
			tc.item.Focus()
			if !tc.item.Focused() {
				t.Fatal("should be focused after Focus()")
			}
			tc.item.Blur()
			if tc.item.Focused() {
				t.Fatal("should be unfocused after Blur()")
			}
		})
	}
}

func TestFormItemToggleHandleKey(t *testing.T) {
	f := newFieldFormItem(graphql.ObjectField{Name: "code", Type: "ID!"}, true)
	f.Focus()

	if f.Value() != "true" {
		t.Fatal("should start as true")
	}

	f.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if f.Value() != "false" {
		t.Fatal("space should toggle to false")
	}

	f.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if f.Value() != "true" {
		t.Fatal("space should toggle back to true")
	}
}

func TestFormItemConsumesTextInput(t *testing.T) {
	toggle := newFieldFormItem(graphql.ObjectField{Name: "a", Type: "String"}, true)
	toggle.Focus()
	if toggle.ConsumesTextInput() {
		t.Fatal("toggle should not consume text input")
	}

	ti := newArgFormItem(graphql.Argument{Name: "b", Type: "String"}, nil, "ep")
	if ti.ConsumesTextInput() {
		t.Fatal("blurred text input should not consume")
	}
	ti.Focus()
	if !ti.ConsumesTextInput() {
		t.Fatal("focused text input should consume")
	}
}

func TestFormItemView(t *testing.T) {
	f := newFieldFormItem(graphql.ObjectField{Name: "code", Type: "ID!"}, true)
	v := f.View()
	if !strings.Contains(v, "code") {
		t.Fatalf("view should contain field name, got %q", v)
	}
	if !strings.Contains(v, "ID!") {
		t.Fatalf("view should contain type hint, got %q", v)
	}
}

func TestBuildDetailFormFieldsAndArgs(t *testing.T) {
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
			},
		},
	}

	df := buildDetailForm(op, nil, objectTypes)
	if df == nil {
		t.Fatal("buildDetailForm returned nil")
	}
	if df.fieldCount != 2 {
		t.Fatalf("expected 2 field toggles, got %d", df.fieldCount)
	}
	if df.Len() != 3 {
		t.Fatalf("expected 3 total items (2 fields + 1 arg), got %d", df.Len())
	}

	if !df.items[0].isField {
		t.Error("first item should be a field toggle")
	}
	if !df.items[1].isField {
		t.Error("second item should be a field toggle")
	}
	if df.items[2].isField {
		t.Error("third item should be an argument, not a field")
	}
}

func TestBuildDetailFormNilForEmptyOp(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "hello",
		Endpoint: "ep",
	}
	df := buildDetailForm(op, nil, nil)
	if df != nil {
		t.Fatal("expected nil form for operation with no fields and no args")
	}
}

func TestBuildDetailFormArgsOnly(t *testing.T) {
	op := &UnifiedOperation{
		Name:       "hello",
		ReturnType: "String",
		Endpoint:   "ep",
		Arguments: []graphql.Argument{
			{Name: "name", Type: "String!"},
		},
	}
	df := buildDetailForm(op, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form for operation with args")
	}
	if df.fieldCount != 0 {
		t.Fatalf("expected 0 field toggles for scalar return, got %d", df.fieldCount)
	}
	if df.Len() != 1 {
		t.Fatalf("expected 1 item, got %d", df.Len())
	}
}

func TestDetailFormCursorNavigation(t *testing.T) {
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
			},
		},
	}
	df := buildDetailForm(op, nil, objectTypes)
	if df.cursor != 0 {
		t.Fatal("cursor should start at 0")
	}

	df.FocusCurrent()
	if !df.items[0].Focused() {
		t.Fatal("item 0 should be focused")
	}

	df.CursorDown()
	if df.cursor != 1 {
		t.Fatalf("cursor should be 1 after down, got %d", df.cursor)
	}
	if !df.items[1].Focused() {
		t.Fatal("item 1 should be focused after CursorDown")
	}
	if df.items[0].Focused() {
		t.Fatal("item 0 should be blurred after CursorDown")
	}

	df.CursorDown()
	df.CursorDown()
	if df.cursor != 2 {
		t.Fatalf("cursor should clamp at 2, got %d", df.cursor)
	}

	df.CursorUp()
	if df.cursor != 1 {
		t.Fatalf("cursor should be 1 after up, got %d", df.cursor)
	}

	df.CursorUp()
	df.CursorUp()
	if df.cursor != 0 {
		t.Fatalf("cursor should clamp at 0, got %d", df.cursor)
	}
}

func TestDetailFormBlurAll(t *testing.T) {
	ep := "https://api.test/graphql"
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   ep,
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "a", Type: "String"},
				{Name: "b", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, nil, objectTypes)
	df.FocusCurrent()
	if !df.items[0].Focused() {
		t.Fatal("item 0 should be focused")
	}

	df.BlurAll()
	for i, item := range df.items {
		if item.Focused() {
			t.Fatalf("item %d should be blurred after BlurAll", i)
		}
	}
}

func TestDetailFormViewSections(t *testing.T) {
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
			},
		},
	}
	df := buildDetailForm(op, nil, objectTypes)
	view := df.View(op)

	for _, want := range []string{"country", "Country", "Fields:", "name", "Arguments:", "code"} {
		if !strings.Contains(view, want) {
			t.Errorf("DetailForm.View missing %q", want)
		}
	}
}

func TestDetailFormViewCursorIndicator(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name:       "test",
		ReturnType: "T",
		Endpoint:   ep,
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "T"): {
			Name: "T",
			Fields: []graphql.ObjectField{
				{Name: "a", Type: "String"},
				{Name: "b", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, nil, objectTypes)

	view0 := df.View(op)
	lines0 := strings.Split(view0, "\n")
	found := false
	for _, line := range lines0 {
		if strings.Contains(line, "\u203a") && strings.Contains(line, "a") {
			found = true
			break
		}
	}
	if !found {
		t.Error("cursor indicator should be on first item 'a'")
	}

	df.CursorDown()
	view1 := df.View(op)
	if view0 == view1 {
		t.Error("view should change after CursorDown")
	}
}

func TestResolveEnumTypeScopedThenBare(t *testing.T) {
	ep := "https://api.test/graphql"
	scopedKey := ScopedTypeKey(ep, "Color")
	enums := map[string]graphql.EnumType{
		scopedKey: {Name: "Color", Values: []graphql.EnumValue{{Name: "RED"}}},
		"Color":   {Name: "Color", Values: []graphql.EnumValue{{Name: "BLUE"}}},
	}

	et, ok := resolveEnumType(enums, ep, "Color")
	if !ok {
		t.Fatal("expected to find scoped enum")
	}
	if et.Values[0].Name != "RED" {
		t.Fatalf("scoped key should take priority, got %q", et.Values[0].Name)
	}

	et2, ok := resolveEnumType(enums, "other-ep", "Color")
	if !ok {
		t.Fatal("expected to find bare enum")
	}
	if et2.Values[0].Name != "BLUE" {
		t.Fatalf("bare key should be fallback, got %q", et2.Values[0].Name)
	}
}
