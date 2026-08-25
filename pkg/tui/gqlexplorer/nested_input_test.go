package gqlexplorer

import (
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func findFormItem(df *DetailForm, name string) *formItem {
	for i := range df.items {
		if df.items[i].name == name {
			return &df.items[i]
		}
	}
	return nil
}

func findFormItems(df *DetailForm, name string) []*formItem {
	var out []*formItem
	for i := range df.items {
		if df.items[i].name == name {
			out = append(out, &df.items[i])
		}
	}
	return out
}

// (a) A field whose type is a nested input object expands into that object's
// fields instead of collapsing into a single "<Type> value" text box.
func TestNestedInputObjectFieldExpands(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "createOrder",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "OrderInput!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"OrderInput": {
			Name: "OrderInput",
			Fields: []graphql.InputField{
				{Name: "sku", Type: "String!"},
				{Name: "address", Type: "AddressInput"},
			},
		},
		"AddressInput": {
			Name: "AddressInput",
			Fields: []graphql.InputField{
				{Name: "street", Type: "String!"},
				{Name: "city", Type: "String"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}
	if findFormItem(df, "address") != nil {
		t.Fatal("address should expand, not appear as a single value box")
	}
	for _, want := range []string{"sku", "street", "city"} {
		if findFormItem(df, want) == nil {
			t.Fatalf("expected expanded field %q", want)
		}
	}
	street := findFormItem(df, "street")
	if street.label != "input.address.street" {
		t.Fatalf("nested label: got %q want input.address.street", street.label)
	}
	if street.depth != 1 {
		t.Fatalf("nested field depth: got %d want 1", street.depth)
	}
}

// (b) A list-of-input-object argument expands into its element's fields.
func TestListOfInputObjectExpandsPerElement(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "createExpense",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "details", Type: "[ExpenseTuitionDetailInput!]!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"ExpenseTuitionDetailInput": {
			Name: "ExpenseTuitionDetailInput",
			Fields: []graphql.InputField{
				{Name: "courseCode", Type: "String!"},
				{Name: "creditHours", Type: "Int"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}
	code := findFormItem(df, "courseCode")
	if code == nil {
		t.Fatal("list-of-input-object did not expand into element fields")
	}
	if !code.listItem {
		t.Fatal("element field should be a list item")
	}
	if code.label != "details[0].courseCode" {
		t.Fatalf("list element label: got %q want details[0].courseCode", code.label)
	}
}

// (c) An input object nested inside an input object (through a list) expands
// recursively to deeper levels.
func TestInputObjectNestedInsideInputObjectExpands(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "createExpense",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "CreateExpenseInput!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"CreateExpenseInput": {
			Name: "CreateExpenseInput",
			Fields: []graphql.InputField{
				{Name: "tuitionDetails", Type: "[ExpenseTuitionDetailInput!]"},
			},
		},
		"ExpenseTuitionDetailInput": {
			Name: "ExpenseTuitionDetailInput",
			Fields: []graphql.InputField{
				{Name: "meta", Type: "TuitionMetaInput!"},
			},
		},
		"TuitionMetaInput": {
			Name: "TuitionMetaInput",
			Fields: []graphql.InputField{
				{Name: "term", Type: "String!"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}
	term := findFormItem(df, "term")
	if term == nil {
		t.Fatal("deeply nested field 'term' did not expand")
	}
	if term.label != "input.tuitionDetails[0].meta.term" {
		t.Fatalf("deep nested label: got %q want input.tuitionDetails[0].meta.term", term.label)
	}
	if term.depth != 2 {
		t.Fatalf("deep nested depth: got %d want 2", term.depth)
	}
}

// (d) A self-referential (cyclic) input type terminates without hanging; the
// recurring type becomes a leaf value box rather than expanding forever.
func TestCyclicInputTypeTerminates(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "addNode",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "node", Type: "NodeInput!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"NodeInput": {
			Name: "NodeInput",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String!"},
				{Name: "child", Type: "NodeInput"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}
	child := findFormItem(df, "child")
	if child == nil {
		t.Fatal("expected 'child' field")
	}
	if child.kind != formItemTextInput {
		t.Fatalf("cyclic field should terminate as a leaf text box, got kind %d", child.kind)
	}
}

// (e) Variables serialize the nested structure as real nested objects/arrays.
func TestBuildVariablesNestedInputSerialization(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "createExpense",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "CreateExpenseInput!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"CreateExpenseInput": {
			Name: "CreateExpenseInput",
			Fields: []graphql.InputField{
				{Name: "amount", Type: "Int!"},
				{Name: "tuitionDetails", Type: "[ExpenseTuitionDetailInput!]"},
				{Name: "student", Type: "StudentInput"},
			},
		},
		"ExpenseTuitionDetailInput": {
			Name: "ExpenseTuitionDetailInput",
			Fields: []graphql.InputField{
				{Name: "courseCode", Type: "String!"},
				{Name: "creditHours", Type: "Int"},
			},
		},
		"StudentInput": {
			Name: "StudentInput",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String!"},
				{Name: "active", Type: "Boolean"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	set := func(name, value string) {
		item := findFormItem(df, name)
		if item == nil {
			t.Fatalf("missing item %q", name)
		}
		item.enabled = true
		item.input.Model.SetValue(value)
	}
	set("amount", "100")
	set("courseCode", "CS101")
	set("creditHours", "3")
	set("name", "Alice")

	got := BuildVariablesMap(op, df)
	input, ok := got["input"].(map[string]any)
	if !ok {
		t.Fatalf("input: expected map, got %T", got["input"])
	}
	if input["amount"] != 100 {
		t.Errorf("amount: got %v want 100", input["amount"])
	}
	details, ok := input["tuitionDetails"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("tuitionDetails: expected 1-element slice, got %T %v", input["tuitionDetails"], input["tuitionDetails"])
	}
	detail := details[0].(map[string]any)
	if detail["courseCode"] != "CS101" || detail["creditHours"] != 3 {
		t.Errorf("tuitionDetails[0]: got %v", detail)
	}
	student, ok := input["student"].(map[string]any)
	if !ok {
		t.Fatalf("student: expected map, got %T", input["student"])
	}
	if student["name"] != "Alice" {
		t.Errorf("student.name: got %v want Alice", student["name"])
	}
	if _, exists := student["active"]; exists {
		t.Errorf("student.active should be omitted when disabled/empty")
	}

	str := BuildVariablesString(op, df)
	for _, want := range []string{"\"tuitionDetails\": [{", "\"courseCode\": \"CS101\"", "\"student\": {"} {
		if !strings.Contains(str, want) {
			t.Errorf("variables string missing %q, got:\n%s", want, str)
		}
	}
}

// (f) A list-of-input-object field nested inside an object argument gains a
// second element once the first element's fields are filled in, mirroring
// the existing top-level list auto-grow behavior.
func TestNestedListInputObjectAddsFollowUpGroupAfterValueEntered(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "createExpense",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "CreateExpenseInput!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"CreateExpenseInput": {
			Name: "CreateExpenseInput",
			Fields: []graphql.InputField{
				{Name: "tuitionDetails", Type: "[ExpenseTuitionDetailInput!]"},
			},
		},
		"ExpenseTuitionDetailInput": {
			Name: "ExpenseTuitionDetailInput",
			Fields: []graphql.InputField{
				{Name: "courseCode", Type: "String!"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	code := findFormItem(df, "courseCode")
	if code == nil {
		t.Fatal("expected courseCode item")
	}
	if code.listBoundary == "" {
		t.Fatal("expected nested list item to carry a list boundary")
	}

	code.input.Model.SetValue("CS101")
	df.syncNestedListBoundary(code.listBoundary)

	codes := findFormItems(df, "courseCode")
	if len(codes) != 2 {
		t.Fatalf("expected second nested list element to be added, got %d courseCode items", len(codes))
	}
	if codes[1].label != "input.tuitionDetails[1].courseCode" {
		t.Fatalf("unexpected second-element label %q", codes[1].label)
	}
}

// (g) A trailing, still-empty nested list element collapses back out once
// the element before it is cleared, mirroring the top-level list shrink
// behavior.
func TestNestedListInputObjectRemovesExtraTrailingBlankGroup(t *testing.T) {
	op := &UnifiedOperation{
		Name:     "createExpense",
		Type:     TypeMutation,
		Endpoint: "http://api/gql",
		Arguments: []graphql.Argument{
			{Name: "input", Type: "CreateExpenseInput!"},
		},
	}
	inputTypes := map[string]graphql.InputType{
		"CreateExpenseInput": {
			Name: "CreateExpenseInput",
			Fields: []graphql.InputField{
				{Name: "tuitionDetails", Type: "[ExpenseTuitionDetailInput!]"},
			},
		},
		"ExpenseTuitionDetailInput": {
			Name: "ExpenseTuitionDetailInput",
			Fields: []graphql.InputField{
				{Name: "courseCode", Type: "String!"},
			},
		},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	code := findFormItem(df, "courseCode")
	if code == nil {
		t.Fatal("expected courseCode item")
	}
	boundary := code.listBoundary

	code.input.Model.SetValue("CS101")
	df.syncNestedListBoundary(boundary)

	codes := findFormItems(df, "courseCode")
	if len(codes) != 2 {
		t.Fatalf("expected second nested list element to be added, got %d", len(codes))
	}

	codes[0].input.Model.SetValue("")
	df.syncNestedListBoundary(boundary)

	codes = findFormItems(df, "courseCode")
	if len(codes) != 1 {
		t.Fatalf("expected trailing blank nested element to collapse, got %d", len(codes))
	}
}

// (h) A list-of-input-object field nested inside a top-level list argument's
// own element (list nested inside list) still grows the outer list
// correctly. Regression guard: the nested list's leaves must keep
// contributing to the outer list's group accounting, or syncListArgRows'
// scan of listGroup gets corrupted by nested items that report group 0
// regardless of which outer element they belong to.
func TestNestedListInsideTopLevelListElementDoesNotBreakOuterGrowth(t *testing.T) {
	inputTypes := map[string]graphql.InputType{
		"OrderInput": {
			Name: "OrderInput",
			Fields: []graphql.InputField{
				{Name: "id", Type: "String!"},
				{Name: "tags", Type: "[TagInput!]"},
			},
		},
		"TagInput": {
			Name: "TagInput",
			Fields: []graphql.InputField{
				{Name: "key", Type: "String!"},
			},
		},
	}
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: "ep",
		Arguments: []graphql.Argument{{Name: "orders", Type: "[OrderInput!]!"}},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	ids := findFormItems(df, "id")
	if len(ids) != 1 {
		t.Fatalf("expected 1 id item initially, got %d", len(ids))
	}
	ids[0].input.Model.SetValue("order-0")
	df.syncListArgRows("orders")

	ids = findFormItems(df, "id")
	if len(ids) != 2 {
		t.Fatalf("expected 2 id items after filling first, got %d", len(ids))
	}
	ids[1].input.Model.SetValue("order-1")
	df.syncListArgRows("orders")

	ids = findFormItems(df, "id")
	if len(ids) != 3 {
		t.Fatalf("expected 3 id items (2 filled + 1 blank continuation) after filling second, got %d", len(ids))
	}
}

// (i) The nested list inside a top-level list element also grows on its own,
// independently of the outer list, once its own field is filled in.
func TestNestedListInsideTopLevelListElementStillGrowsItself(t *testing.T) {
	inputTypes := map[string]graphql.InputType{
		"OrderInput": {
			Name: "OrderInput",
			Fields: []graphql.InputField{
				{Name: "id", Type: "String!"},
				{Name: "tags", Type: "[TagInput!]"},
			},
		},
		"TagInput": {
			Name: "TagInput",
			Fields: []graphql.InputField{
				{Name: "key", Type: "String!"},
			},
		},
	}
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: "ep",
		Arguments: []graphql.Argument{{Name: "orders", Type: "[OrderInput!]!"}},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected detail form")
	}

	keys := findFormItems(df, "key")
	if len(keys) != 1 {
		t.Fatalf("expected 1 key item initially, got %d", len(keys))
	}
	boundary := keys[0].listBoundary
	if boundary == "" {
		t.Fatal("expected key item to carry a nested list boundary")
	}
	keys[0].input.Model.SetValue("k0")
	df.syncNestedListBoundary(boundary)

	keys = findFormItems(df, "key")
	if len(keys) != 2 {
		t.Fatalf("expected inner tags list to grow to 2 elements, got %d", len(keys))
	}
	if keys[1].label != "orders[0].tags[1].key" {
		t.Fatalf("unexpected second inner element label %q", keys[1].label)
	}

	if ids := findFormItems(df, "id"); len(ids) != 1 {
		t.Fatalf("expected outer list to remain at 1 element (id untouched), got %d", len(ids))
	}
}
