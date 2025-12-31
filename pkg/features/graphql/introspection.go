// Package graphql provides GraphQL introspection capabilities
package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
)

// IntrospectionQuery is the standard GraphQL introspection query
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    types {
      ...FullType
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`

// IntrospectionResult represents the result of an introspection query
type IntrospectionResult struct {
	Data struct {
		Schema struct {
			QueryType    *TypeRef           `json:"queryType"`
			MutationType *TypeRef           `json:"mutationType"`
			Types        []IntrospectedType `json:"types"`
		} `json:"__schema"`
	} `json:"data"`
}

// IntrospectedType represents a type from introspection
type IntrospectedType struct {
	Kind          string              `json:"kind"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Fields        []IntrospectedField `json:"fields"`
	InputFields   []IntrospectedInput `json:"inputFields"`
	Interfaces    []TypeRef           `json:"interfaces"`
	EnumValues    []EnumValue         `json:"enumValues"`
	PossibleTypes []TypeRef           `json:"possibleTypes"`
}

// IntrospectedField represents a field in a type
type IntrospectedField struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Args        []IntrospectedInput `json:"args"`
	Type        TypeRef             `json:"type"`
}

// IntrospectedInput represents an input value
type IntrospectedInput struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Type         TypeRef `json:"type"`
	DefaultValue *string `json:"defaultValue"`
}

// TypeRef represents a type reference
type TypeRef struct {
	Kind   string   `json:"kind"`
	Name   *string  `json:"name"`
	OfType *TypeRef `json:"ofType"`
}

// EnumValue represents an enum value
type EnumValue struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// IntrospectSchema performs GraphQL introspection on the given endpoint
func IntrospectSchema(endpoint string, headers map[string]string) (*GraphQLSchema, error) {
	// Create the introspection query request
	queryBody := map[string]string{
		"query": IntrospectionQuery,
	}

	jsonBody, err := json.Marshal(queryBody)
	if err != nil {
		return nil, utils.ColorError("failed to marshal introspection query: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, utils.ColorError("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, utils.ColorError("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, utils.ColorError("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, utils.ColorError(fmt.Sprintf("unexpected status code: %d, body: %s", resp.StatusCode, string(body)))
	}

	// Parse introspection result
	var result IntrospectionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, utils.ColorError("failed to parse introspection result: %w", err)
	}

	// Convert to our schema format
	schema := convertIntrospectionToSchema(&result)

	return schema, nil
}

// convertIntrospectionToSchema converts introspection result to our schema format
func convertIntrospectionToSchema(result *IntrospectionResult) *GraphQLSchema {
	schema := &GraphQLSchema{
		Queries:   make([]SchemaType, 0),
		Mutations: make([]SchemaType, 0),
		Types:     make([]SchemaType, 0),
	}

	// Build type map
	typeMap := make(map[string]IntrospectedType)
	for _, t := range result.Data.Schema.Types {
		// Skip internal types
		if len(t.Name) > 0 && t.Name[0] == '_' {
			continue
		}
		typeMap[t.Name] = t
	}

	// Get query type
	if result.Data.Schema.QueryType != nil && result.Data.Schema.QueryType.Name != nil {
		if queryType, ok := typeMap[*result.Data.Schema.QueryType.Name]; ok {
			schema.Queries = append(schema.Queries, convertType(queryType))
		}
	}

	// Get mutation type
	if result.Data.Schema.MutationType != nil && result.Data.Schema.MutationType.Name != nil {
		if mutationType, ok := typeMap[*result.Data.Schema.MutationType.Name]; ok {
			schema.Mutations = append(schema.Mutations, convertType(mutationType))
		}
	}

	// Get all other types
	for _, t := range typeMap {
		// Skip Query and Mutation types as they're already added
		if result.Data.Schema.QueryType != nil && result.Data.Schema.QueryType.Name != nil &&
			t.Name == *result.Data.Schema.QueryType.Name {
			continue
		}
		if result.Data.Schema.MutationType != nil && result.Data.Schema.MutationType.Name != nil &&
			t.Name == *result.Data.Schema.MutationType.Name {
			continue
		}
		schema.Types = append(schema.Types, convertType(t))
	}

	return schema
}

// convertType converts an introspected type to our schema type
func convertType(introspectedType IntrospectedType) SchemaType {
	schemaType := SchemaType{
		Name:        introspectedType.Name,
		Description: introspectedType.Description,
		Kind:        introspectedType.Kind,
		Fields:      make([]SchemaField, 0),
	}

	for _, field := range introspectedType.Fields {
		schemaField := SchemaField{
			Name:        field.Name,
			Description: field.Description,
			Type:        formatTypeRef(field.Type),
			IsRequired:  isRequired(field.Type),
			Args:        make([]SchemaArg, 0),
		}

		for _, arg := range field.Args {
			schemaArg := SchemaArg{
				Name:        arg.Name,
				Description: arg.Description,
				Type:        formatTypeRef(arg.Type),
				IsRequired:  isRequired(arg.Type),
			}
			schemaField.Args = append(schemaField.Args, schemaArg)
		}

		schemaType.Fields = append(schemaType.Fields, schemaField)
	}

	return schemaType
}

// formatTypeRef formats a type reference as a string
func formatTypeRef(typeRef TypeRef) string {
	if typeRef.Kind == "NON_NULL" {
		if typeRef.OfType != nil {
			return formatTypeRef(*typeRef.OfType) + "!"
		}
	}

	if typeRef.Kind == "LIST" {
		if typeRef.OfType != nil {
			return "[" + formatTypeRef(*typeRef.OfType) + "]"
		}
	}

	if typeRef.Name != nil {
		return *typeRef.Name
	}

	if typeRef.OfType != nil {
		return formatTypeRef(*typeRef.OfType)
	}

	return "Unknown"
}

// isRequired checks if a type is required (NON_NULL)
func isRequired(typeRef TypeRef) bool {
	return typeRef.Kind == "NON_NULL"
}

// BuildQueryTemplate builds a query template from a field
func BuildQueryTemplate(field SchemaField, includeDocs bool) string {
	var builder strings.Builder

	if includeDocs && field.Description != "" {
		builder.WriteString(fmt.Sprintf("# %s\n", field.Description))
	}

	builder.WriteString(field.Name)

	// Add arguments if present
	if len(field.Args) > 0 {
		builder.WriteString("(")
		for i, arg := range field.Args {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("%s: ", arg.Name))
			if arg.IsRequired {
				builder.WriteString("$" + arg.Name)
			} else {
				builder.WriteString("null")
			}
		}
		builder.WriteString(")")
	}

	builder.WriteString(" {\n  # Add fields here\n}")

	return builder.String()
}

// ValidateQuery performs basic validation on a GraphQL query
func ValidateQuery(query string) error {
	query = strings.TrimSpace(query)

	if query == "" {
		return utils.ColorError("query cannot be empty")
	}

	// Check for balanced braces
	braceCount := 0
	for _, char := range query {
		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
		}
		if braceCount < 0 {
			return utils.ColorError("unbalanced braces in query")
		}
	}

	if braceCount != 0 {
		return utils.ColorError("unbalanced braces in query")
	}

	// Check if query starts with query or mutation keyword
	lowerQuery := strings.ToLower(query)
	if !strings.HasPrefix(lowerQuery, "query") &&
		!strings.HasPrefix(lowerQuery, "mutation") &&
		!strings.HasPrefix(lowerQuery, "{") {
		return utils.ColorError("query must start with 'query', 'mutation', or '{'")
	}

	return nil
}
