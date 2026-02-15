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
	Name        string
	Description string
	Type        OperationType
	Endpoint    string
}

func CollectOperations(schema graphql.Schema, endpoint string) []UnifiedOperation {
	var ops []UnifiedOperation
	for _, pair := range []struct {
		ops  []graphql.Operation
		kind OperationType
	}{
		{schema.Queries, TypeQuery},
		{schema.Mutations, TypeMutation},
		{schema.Subscriptions, TypeSubscription},
	} {
		for _, op := range pair.ops {
			ops = append(ops, UnifiedOperation{
				Name: op.Name, Description: op.Description,
				Type: pair.kind, Endpoint: endpoint,
			})
		}
	}
	return ops
}
