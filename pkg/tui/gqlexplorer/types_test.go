package gqlexplorer

import (
	"testing"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func TestCollectOperationsEmpty(t *testing.T) {
	schema := graphql.Schema{}
	ops := CollectOperations(schema, "http://example.com/graphql")

	if len(ops) != 0 {
		t.Errorf("expected 0 operations, got %d", len(ops))
	}
}

func TestCollectOperationsQueriesOnly(t *testing.T) {
	schema := graphql.Schema{
		Queries: []graphql.Operation{
			{Name: "getUser", Description: "fetch a user"},
			{Name: "listUsers"},
		},
	}
	ops := CollectOperations(schema, "http://api.test/gql")

	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}
	for _, op := range ops {
		if op.Type != TypeQuery {
			t.Errorf("expected type %q, got %q", TypeQuery, op.Type)
		}
		if op.Endpoint != "http://api.test/gql" {
			t.Errorf("expected endpoint 'http://api.test/gql', got %q", op.Endpoint)
		}
	}
	if ops[0].Name != "getUser" || ops[0].Description != "fetch a user" {
		t.Errorf("unexpected first op: %+v", ops[0])
	}
	if ops[1].Name != "listUsers" || ops[1].Description != "" {
		t.Errorf("unexpected second op: %+v", ops[1])
	}
}

func TestCollectOperationsMutationsOnly(t *testing.T) {
	schema := graphql.Schema{
		Mutations: []graphql.Operation{
			{Name: "createUser"},
		},
	}
	ops := CollectOperations(schema, "http://api.test/gql")

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Type != TypeMutation {
		t.Errorf("expected type %q, got %q", TypeMutation, ops[0].Type)
	}
	if ops[0].Name != "createUser" {
		t.Errorf("expected name 'createUser', got %q", ops[0].Name)
	}
}

func TestCollectOperationsSubscriptionsOnly(t *testing.T) {
	schema := graphql.Schema{
		Subscriptions: []graphql.Operation{
			{Name: "onMessage", Description: "new messages"},
		},
	}
	ops := CollectOperations(schema, "ws://api.test/gql")

	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].Type != TypeSubscription {
		t.Errorf("expected type %q, got %q", TypeSubscription, ops[0].Type)
	}
	if ops[0].Endpoint != "ws://api.test/gql" {
		t.Errorf("expected endpoint 'ws://api.test/gql', got %q", ops[0].Endpoint)
	}
}

func TestCollectOperationsAllTypes(t *testing.T) {
	schema := graphql.Schema{
		Queries:       []graphql.Operation{{Name: "q1"}, {Name: "q2"}},
		Mutations:     []graphql.Operation{{Name: "m1"}},
		Subscriptions: []graphql.Operation{{Name: "s1"}, {Name: "s2"}, {Name: "s3"}},
	}
	ops := CollectOperations(schema, "http://example.com/graphql")

	if len(ops) != 6 {
		t.Fatalf("expected 6 operations, got %d", len(ops))
	}

	typeCounts := map[OperationType]int{}
	for _, op := range ops {
		typeCounts[op.Type]++
	}
	if typeCounts[TypeQuery] != 2 {
		t.Errorf("expected 2 queries, got %d", typeCounts[TypeQuery])
	}
	if typeCounts[TypeMutation] != 1 {
		t.Errorf("expected 1 mutation, got %d", typeCounts[TypeMutation])
	}
	if typeCounts[TypeSubscription] != 3 {
		t.Errorf("expected 3 subscriptions, got %d", typeCounts[TypeSubscription])
	}
}

func TestCollectOperationsOrderQueriesMutationsSubscriptions(t *testing.T) {
	schema := graphql.Schema{
		Queries:       []graphql.Operation{{Name: "q1"}},
		Mutations:     []graphql.Operation{{Name: "m1"}},
		Subscriptions: []graphql.Operation{{Name: "s1"}},
	}
	ops := CollectOperations(schema, "http://example.com/graphql")

	expected := []struct {
		name   string
		opType OperationType
	}{
		{"q1", TypeQuery},
		{"m1", TypeMutation},
		{"s1", TypeSubscription},
	}
	for i, want := range expected {
		if ops[i].Name != want.name || ops[i].Type != want.opType {
			t.Errorf("index %d: expected {%s, %s}, got {%s, %s}",
				i, want.name, want.opType, ops[i].Name, ops[i].Type)
		}
	}
}

func TestCollectOperationsPreservesDescription(t *testing.T) {
	schema := graphql.Schema{
		Queries: []graphql.Operation{
			{Name: "getUser", Description: "Fetch user by ID"},
		},
		Mutations: []graphql.Operation{
			{Name: "deleteUser", Description: ""},
		},
	}
	ops := CollectOperations(schema, "http://example.com/graphql")

	if ops[0].Description != "Fetch user by ID" {
		t.Errorf("expected description preserved, got %q", ops[0].Description)
	}
	if ops[1].Description != "" {
		t.Errorf("expected empty description, got %q", ops[1].Description)
	}
}

func TestOperationTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		opType   OperationType
		expected string
	}{
		{"query", TypeQuery, "query"},
		{"mutation", TypeMutation, "mutation"},
		{"subscription", TypeSubscription, "subscription"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.opType) != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.opType)
			}
		})
	}
}
