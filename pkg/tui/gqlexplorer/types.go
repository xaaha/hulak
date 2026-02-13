package gqlexplorer

import "github.com/xaaha/hulak/pkg/features/graphql"

// OperationType identifies whether an operation is a query, mutation, or subscription.
type OperationType string

const (
	TypeQuery        OperationType = "query"
	TypeMutation     OperationType = "mutation"
	TypeSubscription OperationType = "subscription"
)

// UnifiedOperation pairs a named operation with its type and source endpoint.
type UnifiedOperation struct {
	Name     string
	Type     OperationType
	Endpoint string
}

// CollectOperations flattens a schema's queries, mutations, and subscriptions
// into a single slice tagged by type and source endpoint.
func CollectOperations(schema graphql.Schema, endpoint string) []UnifiedOperation {
	var ops []UnifiedOperation
	for _, q := range schema.Queries {
		ops = append(ops, UnifiedOperation{Name: q.Name, Type: TypeQuery, Endpoint: endpoint})
	}
	for _, m := range schema.Mutations {
		ops = append(ops, UnifiedOperation{Name: m.Name, Type: TypeMutation, Endpoint: endpoint})
	}
	for _, s := range schema.Subscriptions {
		ops = append(ops, UnifiedOperation{Name: s.Name, Type: TypeSubscription, Endpoint: endpoint})
	}
	return ops
}
