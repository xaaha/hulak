package graphql

// Schema represents a GraphQL schema with queries, mutations, and subscriptions.
// This is a thin domain model independent of the graphql-go-tools library,
// making it easier to test and allowing us to swap libraries in the future.
type Schema struct {
	Queries       []Operation
	Mutations     []Operation
	Subscriptions []Operation
	InputTypes    map[string]InputType // Map of input type name to InputType for TUI lookup
	EnumTypes     map[string]EnumType  // Map of enum type name to EnumType for TUI lookup
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

// InputType represents a GraphQL input object type used for operation arguments.
// These are the complex input types that the TUI needs to understand to build forms.
type InputType struct {
	Name        string
	Description string
	Fields      []InputField
}

// InputField represents a field within an input type.
type InputField struct {
	Name         string
	Type         string
	Description  string
	DefaultValue string
}

// EnumType represents a GraphQL enum type with its possible values.
type EnumType struct {
	Name        string
	Description string
	Values      []EnumValue
}

// EnumValue represents a single value within an enum type.
type EnumValue struct {
	Name              string
	Description       string
	IsDeprecated      bool
	DeprecationReason string
}
