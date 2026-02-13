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
	for _, q := range schema.Queries {
		ops = append(ops, UnifiedOperation{
			Name: q.Name, Description: q.Description,
			Type: TypeQuery, Endpoint: endpoint,
		})
	}
	for _, m := range schema.Mutations {
		ops = append(ops, UnifiedOperation{
			Name: m.Name, Description: m.Description,
			Type: TypeMutation, Endpoint: endpoint,
		})
	}
	for _, s := range schema.Subscriptions {
		ops = append(ops, UnifiedOperation{
			Name: s.Name, Description: s.Description,
			Type: TypeSubscription, Endpoint: endpoint,
		})
	}
	return ops
}
