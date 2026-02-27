package gqlexplorer

import (
	"strings"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

// OperationType identifies whether an operation is a query, mutation, or subscription.
type OperationType string

const (
	TypeQuery        OperationType = "query"
	TypeMutation     OperationType = "mutation"
	TypeSubscription OperationType = "subscription"
)

// UnifiedOperation pairs a named operation with its type and source endpoint.
// Arguments and ReturnType come from the introspection schema so the detail
// panel can display the full operation signature without a second lookup.
type UnifiedOperation struct {
	Name          string
	NameLower     string
	Description   string
	Type          OperationType
	Endpoint      string
	EndpointShort string
	Arguments     []graphql.Argument
	ReturnType    string
}

func ScopedTypeKey(endpoint string, typeName string) string {
	return endpoint + "\x1f" + typeName
}

func ExtractBaseType(t string) string {
	t = strings.TrimSuffix(t, "!")
	t = strings.TrimPrefix(t, "[")
	t = strings.TrimSuffix(t, "]")
	t = strings.TrimSuffix(t, "!")
	return t
}

func CollectOperations(schema *graphql.Schema, endpoint string) []UnifiedOperation {
	if schema == nil {
		return nil
	}

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
				Name: op.Name, NameLower: strings.ToLower(op.Name), Description: op.Description,
				Type: pair.kind, Endpoint: endpoint, EndpointShort: shortenEndpoint(endpoint),
				Arguments: op.Arguments, ReturnType: op.ReturnType,
			})
		}
	}
	return ops
}
