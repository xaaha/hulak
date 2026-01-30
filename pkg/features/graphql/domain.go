package graphql

// Schema represents a GraphQL schema with queries, mutations, and subscriptions.
// This is a thin domain model independent of the graphql-go-tools library,
// making it easier to test and allowing us to swap libraries in the future.
type Schema struct {
	Queries       []Operation
	Mutations     []Operation
	Subscriptions []Operation
}

// Operation represents a query, mutation, or subscription field.
// It includes all metadata needed to display the operation signature
// and help text to the user.
type Operation struct {
	Name              string
	Description       string
	Arguments         []Argument
	ReturnType        string
	IsDeprecated      bool
	DeprecationReason string
}

// Argument represents a field argument with its type and optional default value.
type Argument struct {
	Name         string
	Type         string
	DefaultValue string
}
