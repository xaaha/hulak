package graphql

// Schema represents a GraphQL schema with queries, mutations, and subscriptions.
// This is a thin domain model independent of the graphql-go-tools library,
// making it easier to test and allowing us to swap libraries in the future.
type Schema struct {
	Queries        []Operation
	Mutations      []Operation
	Subscriptions  []Operation
	InputTypes     map[string]InputType     // Map of input type name to InputType for TUI lookup
	EnumTypes      map[string]EnumType      // Map of enum type name to EnumType for TUI lookup
	ObjectTypes    map[string]ObjectType    // Map of object type name to ObjectType for TUI field display
	UnionTypes     map[string]UnionType     // Map of union type name to UnionType for inline fragments
	InterfaceTypes map[string]InterfaceType // Map of interface type name to InterfaceType for inline fragments
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

// UnionType represents a GraphQL union type (e.g., SearchResult, NotificationUnion).
// Unions have no shared fields; all field selection must use inline fragments.
type UnionType struct {
	Name          string
	Description   string
	PossibleTypes []string // concrete OBJECT type names
}

// InterfaceType represents a GraphQL interface type (e.g., Node, Character).
// Interfaces have shared fields that can be selected directly, plus concrete
// types accessed via inline fragments for type-specific fields.
type InterfaceType struct {
	Name          string
	Description   string
	Fields        []ObjectField // shared fields selectable directly
	PossibleTypes []string      // concrete types implementing this interface
}

// ObjectType represents a GraphQL output object type (e.g., Country, User).
// These are the return types that the TUI needs to display selectable fields
// for query building.
type ObjectType struct {
	Name        string
	Description string
	Fields      []ObjectField
}

// ObjectField represents a single field on a GraphQL output object type.
// Fields can themselves return object types, enabling nested field selection.
type ObjectField struct {
	Name        string
	Type        string // e.g. "String", "[Language!]!", "Continent"
	Description string
	Arguments   []Argument // some object fields accept arguments
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
