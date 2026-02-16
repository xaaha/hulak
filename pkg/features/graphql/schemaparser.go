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
		EnumTypes:  make(map[string]EnumType),
	}

	for _, t := range introspectionSchema.Types {
		if t.Name == "" {
			continue
		}
		switch t.Kind {
		case introspection.INPUTOBJECT:
			schema.InputTypes[t.Name] = InputType{
				Name:        t.Name,
				Description: t.Description,
				Fields:      convertInputFields(t.InputFields),
			}
		case introspection.ENUM:
			if !strings.HasPrefix(t.Name, "__") {
				schema.EnumTypes[t.Name] = EnumType{
					Name:        t.Name,
					Description: t.Description,
					Values:      convertEnumValues(t.EnumValues),
				}
			}
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

func convertEnumValues(values []introspection.EnumValue) []EnumValue {
	enumValues := make([]EnumValue, 0, len(values))
	for _, v := range values {
		deprecationReason := ""
		if v.DeprecationReason != nil {
			deprecationReason = *v.DeprecationReason
		}

		enumValues = append(enumValues, EnumValue{
			Name:              v.Name,
			Description:       v.Description,
			IsDeprecated:      v.IsDeprecated,
			DeprecationReason: deprecationReason,
		})
	}
	return enumValues
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
