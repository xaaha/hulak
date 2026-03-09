package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/utils"
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
	if ti.ConsumesTextInput() {
		t.Fatal("selected (non-editing) text input should not consume")
	}
	ti.input.Model.Focus()
	if !ti.ConsumesTextInput() {
		t.Fatal("editing text input should consume")
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
	df.FocusCurrent()

	view0, _ := df.View(op)
	lines0 := strings.Split(view0, "\n")
	found := false
	for _, line := range lines0 {
		if strings.Contains(line, utils.ChevronRight) && strings.Contains(line, "a") {
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
	if df.items[0].ConsumesTextInput() {
		t.Fatal("selected (non-editing) text input should not consume")
	}

	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !df.items[0].ConsumesTextInput() {
		t.Fatal("text input should consume after Enter activates editing")
	}

	df.CursorDown()
	if df.cursor != 1 {
		t.Fatal("CursorDown should move to item 1 even from an editing text input")
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
	df := buildDetailForm(op, inputTypes, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
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
	df := buildDetailForm(op, nil, nil, nil)
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
	df := buildDetailForm(op, nil, nil, nil)
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

func TestEnterTogglesTextInputEditing(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String!"}},
	}
	df := buildDetailForm(op, nil, nil, nil)
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

func TestEscExitsTextInputEditing(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String!"}},
	}
	df := buildDetailForm(op, nil, nil, nil)
	df.FocusCurrent()

	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !df.items[0].input.Model.Focused() {
		t.Fatal("should be editing after Enter")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if df.items[0].input.Model.Focused() {
		t.Fatal("Esc should exit editing")
	}
}

func TestCheckboxPrefixInView(t *testing.T) {
	fi := newArgFormItem(graphql.Argument{Name: "q", Type: "String"}, nil, "ep")
	v := fi.View()
	if !strings.Contains(v, "[") || !strings.Contains(v, "]") {
		t.Fatal("non-field text input should have checkbox brackets in view")
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
	df := buildDetailForm(op, nil, nil, objectTypes)
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
	return buildDetailForm(op, nil, nil, objectTypes)
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

	df.searchInput.Model.SetValue("email")
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

	df.searchInput.Model.SetValue("name")
	df.updateSearchMatches()

	if len(df.matchIndices) != 2 {
		t.Fatalf("expected 2 matches for 'name' (firstName, lastName), got %d", len(df.matchIndices))
	}
	if df.cursor != df.matchIndices[0] {
		t.Errorf("cursor should be at first match %d, got %d", df.matchIndices[0], df.cursor)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.searchInput.Model.SetValue("EMAIL")
	df.updateSearchMatches()

	if len(df.matchIndices) == 0 {
		t.Fatal("expected match for uppercase 'EMAIL'")
	}
}

func TestSearchNoMatches(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.searchInput.Model.SetValue("zzz")
	df.updateSearchMatches()

	if len(df.matchIndices) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(df.matchIndices))
	}
	if status := df.searchStatus(); status != "no matches" {
		t.Errorf("status = %q, want 'no matches'", status)
	}
}

func TestSearchNextPrevMatch(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()

	df.searchInput.Model.SetValue("name")
	df.updateSearchMatches()
	if len(df.matchIndices) < 2 {
		t.Fatal("need at least 2 matches")
	}

	first := df.matchIndices[0]
	second := df.matchIndices[1]

	df.nextMatch()
	if df.cursor != second {
		t.Errorf("after nextMatch: cursor = %d, want %d", df.cursor, second)
	}
	if df.searchStatus() != "2/2" {
		t.Errorf("status = %q, want '2/2'", df.searchStatus())
	}

	df.nextMatch()
	if df.cursor != first {
		t.Error("nextMatch should wrap around to first match")
	}

	df.prevMatch()
	if df.cursor != second {
		t.Error("prevMatch should wrap around to last match")
	}
}

func TestSearchHandleKeyEnterConfirms(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()
	df.searchInput.Model.SetValue("email")
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
	df.searchInput.Model.SetValue("email")
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
	df.searchInput.Model.SetValue("name")
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

	df.searchInput.Model.SetValue("name")
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
	df.searchInput.Model.SetValue("name")
	df.updateSearchMatches()

	df.StopSearch(true)
	if len(df.matchIndices) != 0 {
		t.Fatal("matches should be cleared after confirm")
	}
}

func TestMatchesClearedAfterCancel(t *testing.T) {
	df := searchFormHelper()
	df.StartSearch()
	df.searchInput.Model.SetValue("name")
	df.updateSearchMatches()

	df.StopSearch(false)
	if len(df.matchIndices) != 0 {
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
	df := buildDetailForm(op, nil, nil, nil)
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
