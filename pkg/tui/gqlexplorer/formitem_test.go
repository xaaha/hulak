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

func TestFormItemTextInputViewBordered(t *testing.T) {
	f := newArgFormItem(
		graphql.Argument{Name: "code", Type: "ID!"},
		nil,
		"ep",
	)
	v := f.View()
	if !strings.Contains(v, "code") {
		t.Fatalf("view should contain arg name, got %q", v)
	}
	if !strings.Contains(v, "\u256d") {
		t.Fatal("text input view should have rounded border top-left corner")
	}
	if !strings.Contains(v, "\u2514") {
		t.Fatal("text input view should have connector")
	}
	if strings.Count(v, "\n") < 2 {
		t.Fatal("text input view should be multi-line (label + bordered box)")
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

	df := buildDetailForm(op, nil, nil, objectTypes)
	if df == nil {
		t.Fatal("buildDetailForm returned nil")
	}
	if df.argCount != 1 {
		t.Fatalf("expected 1 argument item, got %d", df.argCount)
	}
	if df.Len() != 3 {
		t.Fatalf("expected 3 total items (1 arg + 2 fields), got %d", df.Len())
	}

	if df.items[0].isField {
		t.Error("first item should be an argument, not a field")
	}
	if !df.items[1].isField {
		t.Error("second item should be a field toggle")
	}
	if !df.items[2].isField {
		t.Error("third item should be a field toggle")
	}
}

func TestBuildDetailFormNilForEmptyOp(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "hello",
		Endpoint: "ep",
	}
	df := buildDetailForm(op, nil, nil, nil)
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
	df := buildDetailForm(op, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form for operation with args")
	}
	if df.argCount != 1 {
		t.Fatalf("expected 1 argument item, got %d", df.argCount)
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
	df := buildDetailForm(op, nil, nil, objectTypes)
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
	df := buildDetailForm(op, nil, nil, objectTypes)
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
	df := buildDetailForm(op, nil, nil, objectTypes)
	view, _ := df.View(op)

	for _, want := range []string{"country", "Country", "name", "code"} {
		if !strings.Contains(view, want) {
			t.Errorf("DetailForm.View missing %q", want)
		}
	}
	for _, absent := range []string{"Fields:", "Arguments:"} {
		if strings.Contains(view, absent) {
			t.Errorf("DetailForm.View should not contain section header %q", absent)
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
	df := buildDetailForm(op, nil, nil, objectTypes)

	view0, _ := df.View(op)
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
	view1, _ := df.View(op)
	if view0 == view1 {
		t.Error("view should change after CursorDown")
	}
}

func TestDetailFormViewCursorLineTracking(t *testing.T) {
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
				{Name: "c", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, nil, nil, objectTypes)

	_, line0 := df.View(op)
	df.CursorDown()
	_, line1 := df.View(op)
	df.CursorDown()
	_, line2 := df.View(op)

	if line1 <= line0 {
		t.Errorf("cursorLine should increase: line0=%d, line1=%d", line0, line1)
	}
	if line2 <= line1 {
		t.Errorf("cursorLine should increase: line1=%d, line2=%d", line1, line2)
	}
}

func TestDetailFormHasExpandedDropdown(t *testing.T) {
	ep := "ep"
	enums := map[string]graphql.EnumType{
		ScopedTypeKey(ep, "Status"): {
			Name:   "Status",
			Values: []graphql.EnumValue{{Name: "A"}, {Name: "B"}},
		},
	}
	op := &UnifiedOperation{
		Name:     "test",
		Endpoint: ep,
		Arguments: []graphql.Argument{
			{Name: "status", Type: "Status"},
		},
	}
	df := buildDetailForm(op, nil, enums, nil)
	if df.hasExpandedDropdown() {
		t.Fatal("should not have expanded dropdown initially")
	}

	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !df.hasExpandedDropdown() {
		t.Fatal("should have expanded dropdown after Enter")
	}
}

func TestDetailFormArrowsAlwaysNavigate(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name:       "test",
		ReturnType: "T",
		Endpoint:   ep,
		Arguments: []graphql.Argument{
			{Name: "name", Type: "String!"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "T"): {
			Name:   "T",
			Fields: []graphql.ObjectField{{Name: "a", Type: "String"}},
		},
	}
	df := buildDetailForm(op, nil, nil, objectTypes)
	df.FocusCurrent()

	if df.cursor != 0 {
		t.Fatal("cursor should start at 0 (text input arg)")
	}
	if !df.items[0].ConsumesTextInput() {
		t.Fatal("focused text input should consume text input")
	}

	df.CursorDown()
	if df.cursor != 1 {
		t.Fatal("CursorDown should move to item 1 even from a focused text input")
	}
	if df.items[0].Focused() {
		t.Fatal("previous text input should be blurred after navigation")
	}
}

func TestBuildDetailFormExpandsInputObject(t *testing.T) {
	ep := "https://api.test/graphql"
	op := &UnifiedOperation{
		Name:       "hello",
		ReturnType: "String!",
		Endpoint:   ep,
		Arguments: []graphql.Argument{
			{Name: "person", Type: "PersonInput"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		ScopedTypeKey(ep, "PersonInput"): {
			Name: "PersonInput",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String!"},
				{Name: "age", Type: "Int"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
	}
	if df.argCount != 2 {
		t.Fatalf("expected 2 expanded argument items (name + age), got %d", df.argCount)
	}
	if df.items[0].name != "name" {
		t.Errorf("expected first item name='name', got %q", df.items[0].name)
	}
	if !df.items[0].required {
		t.Error("'name' (String!) should be required")
	}
	if df.items[1].name != "age" {
		t.Errorf("expected second item name='age', got %q", df.items[1].name)
	}
	if df.items[1].required {
		t.Error("'age' (Int) should not be required")
	}
	if df.items[0].isField || df.items[1].isField {
		t.Error("expanded input fields should not be marked as return-type fields")
	}
}

func TestBuildDetailFormExpandsInputObjectWithScalarArgs(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name:       "createPerson",
		ReturnType: "Person",
		Endpoint:   ep,
		Arguments: []graphql.Argument{
			{Name: "person", Type: "PersonInput!"},
			{Name: "notify", Type: "Boolean"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"PersonInput": {
			Name: "PersonInput",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String!"},
				{Name: "age", Type: "Int"},
			},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Person"): {
			Name: "Person",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "ID!"},
				{Name: "name", Type: "String!"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, objectTypes)
	if df == nil {
		t.Fatal("expected non-nil form")
	}
	if df.argCount != 3 {
		t.Fatalf("expected 3 arg items (2 expanded + 1 scalar), got %d", df.argCount)
	}
	if df.Len() != 5 {
		t.Fatalf("expected 5 total items (3 args + 2 fields), got %d", df.Len())
	}
	if df.items[0].name != "name" || df.items[1].name != "age" {
		t.Error("first two items should be expanded PersonInput fields")
	}
	if df.items[2].name != "notify" {
		t.Errorf("third item should be scalar arg 'notify', got %q", df.items[2].name)
	}
	if df.items[2].kind != formItemToggle {
		t.Error("Boolean arg should be a toggle")
	}
	if !df.items[3].isField || !df.items[4].isField {
		t.Error("last two items should be return-type field toggles")
	}
}

func TestBuildDetailFormObjectFieldsStartUnchecked(t *testing.T) {
	ep := "ep"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "code", Type: "ID!"},
				{Name: "language", Type: "Language"},
			},
		},
		ScopedTypeKey(ep, "Language"): {
			Name: "Language",
			Fields: []graphql.ObjectField{
				{Name: "name", Type: "String"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes)
	if df == nil {
		t.Fatal("expected non-nil form")
	}

	code := df.items[0]
	if !code.toggle.Value {
		t.Error("scalar field 'code' should start checked")
	}
	if code.expandable {
		t.Error("scalar field should not be expandable")
	}

	lang := df.items[1]
	if lang.toggle.Value {
		t.Error("object-type field 'language' should start unchecked")
	}
	if !lang.expandable {
		t.Error("object-type field should be expandable")
	}
}

func TestToggleExpandInsertsAndRemovesChildren(t *testing.T) {
	ep := "ep"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "code", Type: "ID!"},
				{Name: "language", Type: "Language"},
			},
		},
		ScopedTypeKey(ep, "Language"): {
			Name: "Language",
			Fields: []graphql.ObjectField{
				{Name: "name", Type: "String"},
				{Name: "rtl", Type: "Boolean"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes)
	if df.Len() != 2 {
		t.Fatalf("expected 2 items initially, got %d", df.Len())
	}

	df.cursor = 1
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if df.Len() != 4 {
		t.Fatalf("expected 4 items after expand (2 original + 2 children), got %d", df.Len())
	}
	if df.items[2].name != "name" || df.items[3].name != "rtl" {
		t.Errorf("children should be Language fields, got %q and %q", df.items[2].name, df.items[3].name)
	}
	if df.items[2].depth != 1 {
		t.Errorf("children should have depth 1, got %d", df.items[2].depth)
	}

	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if df.Len() != 2 {
		t.Fatalf("expected 2 items after collapse, got %d", df.Len())
	}
}

func TestToggleExpandRecursive(t *testing.T) {
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
	df := buildDetailForm(op, nil, nil, objectTypes)

	countriesIdx := 1
	if !df.items[countriesIdx].expandable {
		t.Fatal("'countries' should be expandable (recursive)")
	}

	df.cursor = countriesIdx
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if df.Len() != 4 {
		t.Fatalf("expected 4 items after first expand, got %d", df.Len())
	}
	nestedCountries := 3
	if !df.items[nestedCountries].expandable {
		t.Fatal("nested 'countries' should also be expandable")
	}
	if df.items[nestedCountries].depth != 1 {
		t.Errorf("nested countries depth should be 1, got %d", df.items[nestedCountries].depth)
	}

	df.cursor = nestedCountries
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if df.Len() != 6 {
		t.Fatalf("expected 6 items after recursive expand, got %d", df.Len())
	}
	if df.items[4].depth != 2 {
		t.Errorf("doubly-nested field depth should be 2, got %d", df.items[4].depth)
	}
}

func TestCollapseRemovesNestedChildren(t *testing.T) {
	ep := "ep"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "A"): {
			Name: "A",
			Fields: []graphql.ObjectField{
				{Name: "b", Type: "B"},
			},
		},
		ScopedTypeKey(ep, "B"): {
			Name: "B",
			Fields: []graphql.ObjectField{
				{Name: "val", Type: "String"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "test",
		ReturnType: "A",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes)

	df.cursor = 0
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	if df.Len() != 2 {
		t.Fatalf("expected 2 after expanding A.b, got %d", df.Len())
	}

	df.cursor = 0
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	if df.Len() != 1 {
		t.Fatalf("expected 1 after collapsing A.b, got %d", df.Len())
	}
}

func TestExpandedFieldIndentation(t *testing.T) {
	ep := "ep"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "T"): {
			Name: "T",
			Fields: []graphql.ObjectField{
				{Name: "child", Type: "C"},
			},
		},
		ScopedTypeKey(ep, "C"): {
			Name: "C",
			Fields: []graphql.ObjectField{
				{Name: "val", Type: "String"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "test",
		ReturnType: "T",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes)
	df.cursor = 0
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	view, _ := df.View(op)
	lines := strings.Split(view, "\n")

	parentLine := ""
	childLine := ""
	for _, l := range lines {
		if strings.Contains(l, "child") {
			parentLine = l
		}
		if strings.Contains(l, "val") {
			childLine = l
		}
	}
	if parentLine == "" || childLine == "" {
		t.Fatal("expected both parent and child lines in view")
	}
	parentIndent := len(parentLine) - len(strings.TrimLeft(parentLine, " "))
	childIndent := len(childLine) - len(strings.TrimLeft(childLine, " "))
	if childIndent <= parentIndent {
		t.Errorf("child should be indented more than parent: parent=%d child=%d", parentIndent, childIndent)
	}
}

func TestResolveEnumTypeScopedThenBare(t *testing.T) {
	ep := "https://api.test/graphql"
	scopedKey := ScopedTypeKey(ep, "Color")
	enums := map[string]graphql.EnumType{
		scopedKey: {Name: "Color", Values: []graphql.EnumValue{{Name: "RED"}}},
		"Color":   {Name: "Color", Values: []graphql.EnumValue{{Name: "BLUE"}}},
	}

	et, ok := resolveType(enums, ep, "Color")
	if !ok {
		t.Fatal("expected to find scoped enum")
	}
	if et.Values[0].Name != "RED" {
		t.Fatalf("scoped key should take priority, got %q", et.Values[0].Name)
	}

	et2, ok := resolveType(enums, "other-ep", "Color")
	if !ok {
		t.Fatal("expected to find bare enum")
	}
	if et2.Values[0].Name != "BLUE" {
		t.Fatalf("bare key should be fallback, got %q", et2.Values[0].Name)
	}
}
