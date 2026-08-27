package gqlexplorer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
)

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
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
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
	df := buildDetailForm(op, inputTypes, nil, objectTypes, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
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
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
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
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)
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
		t.Errorf(
			"children should be Language fields, got %q and %q",
			df.items[2].name,
			df.items[3].name,
		)
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
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)

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
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)

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
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)
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
		t.Errorf(
			"child should be indented more than parent: parent=%d child=%d",
			parentIndent,
			childIndent,
		)
	}
}

func TestBuildDetailFormUnionReturnType(t *testing.T) {
	ep := "ep"
	unionTypes := map[string]graphql.UnionType{
		ScopedTypeKey(ep, "SearchResult"): {
			Name:          "SearchResult",
			PossibleTypes: []string{"User", "Post"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "User"): {
			Name: "User",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "ID!"},
				{Name: "name", Type: "String"},
			},
		},
		ScopedTypeKey(ep, "Post"): {
			Name: "Post",
			Fields: []graphql.ObjectField{
				{Name: "title", Type: "String"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "search",
		ReturnType: "SearchResult",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes, unionTypes, nil)
	if df == nil {
		t.Fatal("expected non-nil form for union return type")
		return
	}
	if df.argCount != 0 {
		t.Fatalf("expected 0 args, got %d", df.argCount)
	}
	if df.Len() != 2 {
		t.Fatalf("expected 2 items (2 inline fragments), got %d", df.Len())
	}
	if df.items[0].name != fragmentPrefix+"User" {
		t.Errorf("expected first fragment '... on User', got %q", df.items[0].name)
	}
	if df.items[1].name != fragmentPrefix+"Post" {
		t.Errorf("expected second fragment '... on Post', got %q", df.items[1].name)
	}
	if !df.items[0].expandable || !df.items[1].expandable {
		t.Error("fragment items should be expandable")
	}
	if !df.items[0].isField || !df.items[1].isField {
		t.Error("fragment items should be marked as fields")
	}
}

func TestBuildDetailFormUnionFragmentExpand(t *testing.T) {
	ep := "ep"
	unionTypes := map[string]graphql.UnionType{
		ScopedTypeKey(ep, "SearchResult"): {
			Name:          "SearchResult",
			PossibleTypes: []string{"User"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "User"): {
			Name: "User",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "ID!"},
				{Name: "name", Type: "String"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "search",
		ReturnType: "SearchResult",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes, unionTypes, nil)
	if df.Len() != 1 {
		t.Fatalf("expected 1 fragment item, got %d", df.Len())
	}

	df.cursor = 0
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})

	if df.Len() != 3 {
		t.Fatalf("expected 3 items after expand (1 fragment + 2 children), got %d", df.Len())
	}
	if df.items[1].name != "id" || df.items[2].name != "name" {
		t.Errorf(
			"children should be User fields, got %q and %q",
			df.items[1].name,
			df.items[2].name,
		)
	}
	if df.items[1].depth != 1 || df.items[2].depth != 1 {
		t.Error("children should have depth 1")
	}

	df.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	if df.Len() != 1 {
		t.Fatalf("expected 1 item after collapse, got %d", df.Len())
	}
}

func TestBuildDetailFormInterfaceReturnType(t *testing.T) {
	ep := "ep"
	interfaceTypes := map[string]graphql.InterfaceType{
		ScopedTypeKey(ep, "Node"): {
			Name: "Node",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "ID!"},
			},
			PossibleTypes: []string{"User", "Post"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "User"): {
			Name: "User",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "ID!"},
				{Name: "name", Type: "String"},
			},
		},
		ScopedTypeKey(ep, "Post"): {
			Name: "Post",
			Fields: []graphql.ObjectField{
				{Name: "title", Type: "String"},
			},
		},
	}
	op := &UnifiedOperation{
		Name:       "node",
		ReturnType: "Node",
		Endpoint:   ep,
	}
	df := buildDetailForm(op, nil, nil, objectTypes, nil, interfaceTypes)
	if df == nil {
		t.Fatal("expected non-nil form for interface return type")
		return
	}
	if df.Len() != 3 {
		t.Fatalf("expected 3 items (1 shared field + 2 fragments), got %d", df.Len())
	}
	if df.items[0].name != "id" || !df.items[0].isField {
		t.Error("first item should be shared field 'id'")
	}
	if df.items[1].name != fragmentPrefix+"User" {
		t.Errorf("expected second item '... on User', got %q", df.items[1].name)
	}
	if df.items[2].name != fragmentPrefix+"Post" {
		t.Errorf("expected third item '... on Post', got %q", df.items[2].name)
	}
}

func TestBuildDetailFormUnionWithArgs(t *testing.T) {
	ep := "ep"
	unionTypes := map[string]graphql.UnionType{
		ScopedTypeKey(ep, "NotificationUnion"): {
			Name:          "NotificationUnion",
			PossibleTypes: []string{"AiringNotification", "FollowingNotification"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "AiringNotification"): {
			Name: "AiringNotification",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "Int!"},
				{Name: "type", Type: "NotificationType"},
			},
		},
		ScopedTypeKey(ep, "FollowingNotification"): {
			Name: "FollowingNotification",
			Fields: []graphql.ObjectField{
				{Name: "id", Type: "Int!"},
				{Name: "userId", Type: "Int!"},
			},
		},
	}
	enums := map[string]graphql.EnumType{
		ScopedTypeKey(ep, "NotificationType"): {
			Name:   "NotificationType",
			Values: []graphql.EnumValue{{Name: "AIRING"}, {Name: "FOLLOWING"}},
		},
	}
	op := &UnifiedOperation{
		Name:       "Notification",
		ReturnType: "NotificationUnion",
		Endpoint:   ep,
		Arguments: []graphql.Argument{
			{Name: "type", Type: "NotificationType"},
			{Name: "resetNotificationCount", Type: "Boolean"},
		},
	}
	df := buildDetailForm(op, nil, enums, objectTypes, unionTypes, nil)
	if df == nil {
		t.Fatal("expected non-nil form")
		return
	}
	if df.argCount != 2 {
		t.Fatalf("expected 2 args, got %d", df.argCount)
	}
	if df.Len() != 4 {
		t.Fatalf("expected 4 items (2 args + 2 fragments), got %d", df.Len())
	}
	if df.items[0].kind != formItemDropdown {
		t.Error("first arg should be dropdown for NotificationType enum")
	}
	if df.items[1].kind != formItemToggle {
		t.Error("second arg should be toggle for Boolean")
	}
	if df.items[2].name != fragmentPrefix+"AiringNotification" {
		t.Errorf("expected fragment for AiringNotification, got %q", df.items[2].name)
	}
	if df.items[3].name != fragmentPrefix+"FollowingNotification" {
		t.Errorf("expected fragment for FollowingNotification, got %q", df.items[3].name)
	}
}

func TestNewFragmentFormItem(t *testing.T) {
	fi := newFragmentFormItem("AiringNotification")
	if fi.kind != formItemToggle {
		t.Fatal("fragment item should be a toggle")
	}
	if fi.name != fragmentPrefix+"AiringNotification" {
		t.Errorf("expected name %q, got %q", fragmentPrefix+"AiringNotification", fi.name)
	}
	if fi.typeHint != "AiringNotification" {
		t.Errorf("typeHint should be the concrete type name, got %q", fi.typeHint)
	}
	if !fi.expandable {
		t.Error("fragment item should be expandable")
	}
	if !fi.isField {
		t.Error("fragment item should be marked as field")
	}
	if fi.Value() != "false" {
		t.Error("fragment should start unchecked")
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
