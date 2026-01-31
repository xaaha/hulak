package graphql

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"
)

// IntrospectionResponse represents the standard GraphQL introspection response structure
type IntrospectionResponse struct {
	Data struct {
		Schema introspection.Schema `json:"__schema"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// ParseIntrospectionResponse parses the JSON response from a GraphQL introspection query.
// It returns the introspection.Schema object or an error if parsing fails or GraphQL errors exist.
func ParseIntrospectionResponse(jsonData []byte) (*introspection.Schema, error) {
	var response IntrospectionResponse
	if err := json.Unmarshal(jsonData, &response); err != nil {
		return nil, fmt.Errorf("failed to parse introspection response: %w", err)
	}

	// Check for GraphQL errors
	if len(response.Errors) > 0 {
		var errMsgs []string
		for _, e := range response.Errors {
			errMsgs = append(errMsgs, e.Message)
		}
		return nil, fmt.Errorf(
			"introspection query returned errors: %s",
			strings.Join(errMsgs, "; "),
		)
	}

	return &response.Data.Schema, nil
}

// ConvertToSchema converts the introspection.Schema from the library into our domain model.
// This decouples our code from the graphql-go-tools library and provides a clean interface.
func ConvertToSchema(introspectionSchema *introspection.Schema) (Schema, error) {
	if introspectionSchema == nil {
		return Schema{}, fmt.Errorf("introspection schema is nil")
	}

	// Build a type map for lookup by name
	typeMap := make(map[string]*introspection.FullType)
	for _, t := range introspectionSchema.Types {
		if t.Name != "" {
			typeMap[t.Name] = t
		}
	}

	schema := Schema{
		InputTypes: make(map[string]InputType),
	}

	// Extract input types (for TUI form building)
	for _, t := range introspectionSchema.Types {
		if t.Kind == introspection.INPUTOBJECT && t.Name != "" {
			inputType := InputType{
				Name:        t.Name,
				Description: t.Description,
				Fields:      convertInputFields(t.InputFields),
			}
			schema.InputTypes[t.Name] = inputType
		}
	}

	// Extract queries - look up QueryType by name in the types array
	if introspectionSchema.QueryType.Name != "" {
		if queryType, ok := typeMap[introspectionSchema.QueryType.Name]; ok {
			schema.Queries = convertFields(queryType.Fields)
		}
	}

	// Extract mutations (handle nil MutationType)
	if introspectionSchema.MutationType != nil && introspectionSchema.MutationType.Name != "" {
		if mutationType, ok := typeMap[introspectionSchema.MutationType.Name]; ok {
			schema.Mutations = convertFields(mutationType.Fields)
		}
	}

	// Extract subscriptions (handle nil SubscriptionType)
	if introspectionSchema.SubscriptionType != nil &&
		introspectionSchema.SubscriptionType.Name != "" {
		if subscriptionType, ok := typeMap[introspectionSchema.SubscriptionType.Name]; ok {
			schema.Subscriptions = convertFields(subscriptionType.Fields)
		}
	}

	return schema, nil
}

// convertFields converts a slice of introspection.Field to our domain Operation model
func convertFields(fields []introspection.Field) []Operation {
	operations := make([]Operation, 0, len(fields))
	for _, field := range fields {
		deprecationReason := ""
		if field.DeprecationReason != nil {
			deprecationReason = *field.DeprecationReason
		}

		op := Operation{
			Name:              field.Name,
			Description:       field.Description,
			Arguments:         convertArguments(field.Args),
			ReturnType:        formatType(&field.Type),
			IsDeprecated:      field.IsDeprecated,
			DeprecationReason: deprecationReason,
		}
		operations = append(operations, op)
	}
	return operations
}

// convertArguments converts introspection.InputValue arguments to our domain Argument model
func convertArguments(args []introspection.InputValue) []Argument {
	arguments := make([]Argument, 0, len(args))
	for _, arg := range args {
		defaultValue := ""
		if arg.DefaultValue != nil {
			defaultValue = *arg.DefaultValue
		}

		a := Argument{
			Name:         arg.Name,
			Type:         formatType(&arg.Type),
			DefaultValue: defaultValue,
		}
		arguments = append(arguments, a)
	}
	return arguments
}

// convertInputFields converts introspection.InputValue fields to our domain InputField model
func convertInputFields(fields []introspection.InputValue) []InputField {
	inputFields := make([]InputField, 0, len(fields))
	for _, field := range fields {
		defaultValue := ""
		if field.DefaultValue != nil {
			defaultValue = *field.DefaultValue
		}

		f := InputField{
			Name:         field.Name,
			Type:         formatType(&field.Type),
			Description:  field.Description,
			DefaultValue: defaultValue,
		}
		inputFields = append(inputFields, f)
	}
	return inputFields
}

// formatType recursively formats a TypeRef into a readable GraphQL type string.
// Examples:
//   - "String"
//   - "String!"
//   - "[String]"
//   - "[String!]!"
func formatType(t *introspection.TypeRef) string {
	if t == nil {
		return ""
	}

	switch t.Kind {
	case introspection.NONNULL:
		// Non-null types wrap another type and add "!"
		return formatType(t.OfType) + "!"
	case introspection.LIST:
		// List types wrap another type in brackets
		return "[" + formatType(t.OfType) + "]"
	default:
		// Scalar, Object, Interface, Union, Enum, InputObject
		if t.Name != nil {
			return *t.Name
		}
		return ""
	}
}

// TODO-gql: All display functions below are temporary for Phase 1.
// Remove once TUI consumes the Schema/Operation/InputType structs directly.
// The TUI will use schema.Queries, schema.Mutations, schema.InputTypes to build interactive forms.

// DisplaySchema prints a formatted schema to stdout.
// This provides a readable view of queries, mutations, and subscriptions.
func DisplaySchema(schema Schema) {
	if len(schema.Queries) > 0 {
		displayOperations(schema.Queries, "QUERIES", schema.InputTypes)
	}

	if len(schema.Mutations) > 0 {
		displayOperations(schema.Mutations, "MUTATIONS", schema.InputTypes)
	}

	if len(schema.Subscriptions) > 0 {
		displayOperations(schema.Subscriptions, "SUBSCRIPTIONS", schema.InputTypes)
	}
}

// displayOperations prints a group of operations (queries, mutations, or subscriptions)
func displayOperations(ops []Operation, title string, inputTypes map[string]InputType) {
	fmt.Printf("\n=== %s ===\n\n", title)
	for _, op := range ops {
		// Print the signature
		fmt.Printf("  %s\n", formatSignature(op))

		// Print description if available
		if op.Description != "" {
			// Indent and wrap description
			desc := strings.TrimSpace(op.Description)
			fmt.Printf("    %s\n", desc)
		}

		// Print deprecation warning if applicable
		if op.IsDeprecated {
			reason := "This field is deprecated"
			if op.DeprecationReason != "" {
				reason = op.DeprecationReason
			}
			fmt.Printf("    ⚠️  DEPRECATED: %s\n", reason)
		}

		// Print input type details for arguments that are input objects
		displayInputTypeDetails(op.Arguments, inputTypes, "    ")

		fmt.Println()
	}
}

// displayInputTypeDetails shows the fields of input types used in operation arguments.
// This is crucial for TUI - users need to see what fields to fill in.
// Shows nested input types up to 2 levels deep to avoid overwhelming output.
func displayInputTypeDetails(args []Argument, inputTypes map[string]InputType, indent string) {
	for _, arg := range args {
		// Extract the base type name (strip !, [], etc.)
		baseType := extractBaseTypeName(arg.Type)

		// Check if this is an input type we know about
		if inputType, ok := inputTypes[baseType]; ok {
			fmt.Printf("%s↳ %s fields:\n", indent, arg.Name)
			displayInputTypeFields(inputType, inputTypes, indent+"  ", 1)
		}
	}
}

// displayInputTypeFields recursively displays input type fields up to maxDepth.
func displayInputTypeFields(
	inputType InputType,
	inputTypes map[string]InputType,
	indent string,
	depth int,
) {
	const maxDepth = 2 // Limit nesting to avoid overwhelming output

	for _, field := range inputType.Fields {
		fieldStr := fmt.Sprintf("%s- %s: %s", indent, field.Name, field.Type)
		if field.DefaultValue != "" {
			fieldStr += fmt.Sprintf(" = %s", field.DefaultValue)
		}
		fmt.Println(fieldStr)
		if field.Description != "" {
			fmt.Printf("%s  %s\n", indent, field.Description)
		}

		// Recursively display nested input types if within depth limit
		if depth < maxDepth {
			nestedBaseType := extractBaseTypeName(field.Type)
			if nestedInputType, ok := inputTypes[nestedBaseType]; ok {
				displayInputTypeFields(nestedInputType, inputTypes, indent+"  ", depth+1)
			}
		}
	}
}

// extractBaseTypeName extracts the base type name from a GraphQL type string.
// Examples: "[String!]!" -> "String", "PersonInput!" -> "PersonInput"
func extractBaseTypeName(typeStr string) string {
	// Remove all wrapping characters: [], !, etc.
	cleaned := strings.ReplaceAll(typeStr, "[", "")
	cleaned = strings.ReplaceAll(cleaned, "]", "")
	cleaned = strings.ReplaceAll(cleaned, "!", "")
	return strings.TrimSpace(cleaned)
}

// formatSignature formats an operation signature like "user(id: ID!): User"
func formatSignature(op Operation) string {
	var sb strings.Builder
	sb.WriteString(op.Name)

	// Add arguments
	if len(op.Arguments) > 0 {
		sb.WriteString("(")
		for i, arg := range op.Arguments {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.Name)
			sb.WriteString(": ")
			sb.WriteString(arg.Type)

			// Add default value if present
			if arg.DefaultValue != "" {
				sb.WriteString(" = ")
				sb.WriteString(arg.DefaultValue)
			}
		}
		sb.WriteString(")")
	}

	// Add return type
	sb.WriteString(": ")
	sb.WriteString(op.ReturnType)

	return sb.String()
}
