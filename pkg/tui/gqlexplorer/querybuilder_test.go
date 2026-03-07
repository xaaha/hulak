package gqlexplorer

import (
	"testing"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

func TestBuildQueryString(t *testing.T) {
	tests := []struct {
		name     string
		op       *UnifiedOperation
		df       *DetailForm
		expected string
	}{
		{
			name:     "nil operation returns empty",
			op:       nil,
			df:       nil,
			expected: "",
		},
		{
			name: "query no args no fields",
			op: &UnifiedOperation{
				Name: "GetUsers",
				Type: TypeQuery,
			},
			df:       nil,
			expected: "query GetUsers {\n  GetUsers\n}",
		},
		{
			name: "query single arg",
			op: &UnifiedOperation{
				Name: "GetUser",
				Type: TypeQuery,
				Arguments: []graphql.Argument{
					{Name: "id", Type: "ID!"},
				},
			},
			df:       nil,
			expected: "query GetUser($id: ID!) {\n  GetUser(id: $id)\n}",
		},
		{
			name: "mutation multiple args",
			op: &UnifiedOperation{
				Name: "CreateUser",
				Type: TypeMutation,
				Arguments: []graphql.Argument{
					{Name: "name", Type: "String!"},
					{Name: "age", Type: "Int"},
					{Name: "role", Type: "Role!"},
				},
			},
			df:       nil,
			expected: "mutation CreateUser($name: String!, $age: Int, $role: Role!) {\n  CreateUser(name: $name, age: $age, role: $role)\n}",
		},
		{
			name: "subscription type",
			op: &UnifiedOperation{
				Name: "OnMessage",
				Type: TypeSubscription,
			},
			df:       nil,
			expected: "subscription OnMessage {\n  OnMessage\n}",
		},
		{
			name: "query with selected return fields",
			op: &UnifiedOperation{
				Name: "GetUser",
				Type: TypeQuery,
				Arguments: []graphql.Argument{
					{Name: "id", Type: "ID!"},
				},
			},
			df: &DetailForm{
				argCount: 1,
				items: []formItem{
					{kind: formItemTextInput, name: "id"},
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", true)},
					{kind: formItemToggle, name: "email", isField: true, toggle: tui.NewToggle("email", true)},
					{kind: formItemToggle, name: "phone", isField: true, toggle: tui.NewToggle("phone", false)},
				},
			},
			expected: "query GetUser($id: ID!) {\n  GetUser(id: $id) {\n    name\n    email\n  }\n}",
		},
		{
			name: "fields only no args",
			op: &UnifiedOperation{
				Name: "GetUsers",
				Type: TypeQuery,
			},
			df: &DetailForm{
				argCount: 0,
				items: []formItem{
					{kind: formItemToggle, name: "id", isField: true, toggle: tui.NewToggle("id", true)},
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", true)},
				},
			},
			expected: "query GetUsers {\n  GetUsers {\n    id\n    name\n  }\n}",
		},
		{
			name: "nested expandable fields",
			op: &UnifiedOperation{
				Name: "GetUser",
				Type: TypeQuery,
			},
			df: &DetailForm{
				argCount: 0,
				items: []formItem{
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", true), depth: 0},
					{kind: formItemToggle, name: "address", isField: true, toggle: tui.NewToggle("address", true), depth: 0, expandable: true},
					{kind: formItemToggle, name: "street", isField: true, toggle: tui.NewToggle("street", true), depth: 1},
					{kind: formItemToggle, name: "city", isField: true, toggle: tui.NewToggle("city", true), depth: 1},
					{kind: formItemToggle, name: "phone", isField: true, toggle: tui.NewToggle("phone", false), depth: 0},
				},
			},
			expected: "query GetUser {\n  GetUser {\n    name\n    address {\n      street\n      city\n    }\n  }\n}",
		},
		{
			name: "deeply nested fields",
			op: &UnifiedOperation{
				Name: "GetCountry",
				Type: TypeQuery,
				Arguments: []graphql.Argument{
					{Name: "code", Type: "String!"},
				},
			},
			df: &DetailForm{
				argCount: 1,
				items: []formItem{
					{kind: formItemTextInput, name: "code"},
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", true), depth: 0},
					{kind: formItemToggle, name: "continent", isField: true, toggle: tui.NewToggle("continent", true), depth: 0, expandable: true},
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", true), depth: 1},
					{kind: formItemToggle, name: "countries", isField: true, toggle: tui.NewToggle("countries", true), depth: 1, expandable: true},
					{kind: formItemToggle, name: "code", isField: true, toggle: tui.NewToggle("code", true), depth: 2},
					{kind: formItemToggle, name: "languages", isField: true, toggle: tui.NewToggle("languages", false), depth: 0, expandable: true},
				},
			},
			expected: "query GetCountry($code: String!) {\n  GetCountry(code: $code) {\n    name\n    continent {\n      name\n      countries {\n        code\n      }\n    }\n  }\n}",
		},
		{
			name: "all fields toggled off produces no selection set",
			op: &UnifiedOperation{
				Name: "GetUser",
				Type: TypeQuery,
			},
			df: &DetailForm{
				argCount: 0,
				items: []formItem{
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", false), depth: 0},
					{kind: formItemToggle, name: "email", isField: true, toggle: tui.NewToggle("email", false), depth: 0},
				},
			},
			expected: "query GetUser {\n  GetUser\n}",
		},
		{
			name: "expandable field on but all children off is skipped",
			op: &UnifiedOperation{
				Name: "GetUser",
				Type: TypeQuery,
			},
			df: &DetailForm{
				argCount: 0,
				items: []formItem{
					{kind: formItemToggle, name: "name", isField: true, toggle: tui.NewToggle("name", true), depth: 0},
					{kind: formItemToggle, name: "address", isField: true, toggle: tui.NewToggle("address", true), depth: 0, expandable: true},
					{kind: formItemToggle, name: "street", isField: true, toggle: tui.NewToggle("street", false), depth: 1},
					{kind: formItemToggle, name: "city", isField: true, toggle: tui.NewToggle("city", false), depth: 1},
				},
			},
			expected: "query GetUser {\n  GetUser {\n    name\n  }\n}",
		},
		{
			name: "empty detail form produces no selection set",
			op: &UnifiedOperation{
				Name: "Ping",
				Type: TypeQuery,
			},
			df:       &DetailForm{argCount: 0, items: nil},
			expected: "query Ping {\n  Ping\n}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildQueryString(tc.op, tc.df)
			if got != tc.expected {
				t.Errorf("BuildQueryString()\ngot:\n%s\nwant:\n%s", got, tc.expected)
			}
		})
	}
}

func TestBuildSelectionLines(t *testing.T) {
	tests := []struct {
		name         string
		items        []formItem
		parentDepth  int
		level        int
		wantLines    []string
		wantConsumed int
	}{
		{
			name:         "empty items",
			items:        nil,
			parentDepth:  -1,
			level:        2,
			wantLines:    nil,
			wantConsumed: 0,
		},
		{
			name: "flat fields",
			items: []formItem{
				{kind: formItemToggle, name: "a", toggle: tui.NewToggle("a", true), depth: 0},
				{kind: formItemToggle, name: "b", toggle: tui.NewToggle("b", false), depth: 0},
				{kind: formItemToggle, name: "c", toggle: tui.NewToggle("c", true), depth: 0},
			},
			parentDepth:  -1,
			level:        2,
			wantLines:    []string{"    a", "    c"},
			wantConsumed: 3,
		},
		{
			name: "stops at parent depth",
			items: []formItem{
				{kind: formItemToggle, name: "child1", toggle: tui.NewToggle("child1", true), depth: 1},
				{kind: formItemToggle, name: "child2", toggle: tui.NewToggle("child2", true), depth: 1},
				{kind: formItemToggle, name: "sibling", toggle: tui.NewToggle("sibling", true), depth: 0},
			},
			parentDepth:  0,
			level:        3,
			wantLines:    []string{"      child1", "      child2"},
			wantConsumed: 2,
		},
		{
			name: "non-toggle items are skipped",
			items: []formItem{
				{kind: formItemTextInput, name: "input", depth: 0},
				{kind: formItemToggle, name: "field", toggle: tui.NewToggle("field", true), depth: 0},
			},
			parentDepth:  -1,
			level:        2,
			wantLines:    []string{"    field"},
			wantConsumed: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotLines, gotConsumed := buildSelectionLines(tc.items, 0, tc.parentDepth, tc.level)
			if gotConsumed != tc.wantConsumed {
				t.Errorf("consumed = %d, want %d", gotConsumed, tc.wantConsumed)
			}
			if len(gotLines) != len(tc.wantLines) {
				t.Fatalf("lines count = %d, want %d\ngot:  %v\nwant: %v", len(gotLines), len(tc.wantLines), gotLines, tc.wantLines)
			}
			for i := range gotLines {
				if gotLines[i] != tc.wantLines[i] {
					t.Errorf("line[%d] = %q, want %q", i, gotLines[i], tc.wantLines[i])
				}
			}
		})
	}
}
