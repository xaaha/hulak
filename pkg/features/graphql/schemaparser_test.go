package graphql

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/introspection"
)

func TestParseIntrospectionResponse(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errMsg    string
	}{
		{
			name: "valid response",
			jsonData: `{
				"data": {
					"__schema": {
						"queryType": {"name": "Query"},
						"mutationType": null,
						"subscriptionType": null,
						"types": []
					}
				}
			}`,
			wantError: false,
		},
		{
			name: "response with errors",
			jsonData: `{
				"errors": [
					{"message": "Introspection is disabled"}
				]
			}`,
			wantError: true,
			errMsg:    "Introspection is disabled",
		},
		{
			name:      "invalid JSON",
			jsonData:  `{"invalid json`,
			wantError: true,
			errMsg:    "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ParseIntrospectionResponse([]byte(tt.jsonData))
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain '%s', got: %s", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if data == nil {
					t.Error("Expected data but got nil")
				}
			}
		})
	}
}

func TestConvertToSchema_NilMutationType(t *testing.T) {
	// Schema with only queries (no mutations or subscriptions)
	queryType := createQueryType()
	introspectionSchema := &introspection.Schema{
		QueryType:        queryType,
		MutationType:     nil,
		SubscriptionType: nil,
		Types:            []*introspection.FullType{&queryType},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(schema.Queries) == 0 {
		t.Error("Expected queries but got none")
	}
	if len(schema.Mutations) != 0 {
		t.Errorf("Expected no mutations but got %d", len(schema.Mutations))
	}
	if len(schema.Subscriptions) != 0 {
		t.Errorf("Expected no subscriptions but got %d", len(schema.Subscriptions))
	}
}

func TestConvertToSchema_WithMutations(t *testing.T) {
	queryType := createQueryType()
	mutationType := createMutationType()
	introspectionSchema := &introspection.Schema{
		QueryType:        queryType,
		MutationType:     &mutationType,
		SubscriptionType: nil,
		Types: []*introspection.FullType{
			&queryType,
			&mutationType,
		},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(schema.Queries) == 0 {
		t.Error("Expected queries but got none")
	}
	if len(schema.Mutations) == 0 {
		t.Error("Expected mutations but got none")
	}
	if len(schema.Subscriptions) != 0 {
		t.Errorf("Expected no subscriptions but got %d", len(schema.Subscriptions))
	}
}

func TestConvertToSchema_NilData(t *testing.T) {
	_, err := ConvertToSchema(nil)
	if err == nil {
		t.Error("Expected error for nil data but got none")
	}
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name     string
		typeRef  *introspection.TypeRef
		expected string
	}{
		{
			name:     "nil type",
			typeRef:  nil,
			expected: "",
		},
		{
			name: "scalar type",
			typeRef: &introspection.TypeRef{
				Kind: introspection.SCALAR,
				Name: stringPtr("String"),
			},
			expected: "String",
		},
		{
			name: "non-null scalar",
			typeRef: &introspection.TypeRef{
				Kind: introspection.NONNULL,
				OfType: &introspection.TypeRef{
					Kind: introspection.SCALAR,
					Name: stringPtr("String"),
				},
			},
			expected: "String!",
		},
		{
			name: "list of scalars",
			typeRef: &introspection.TypeRef{
				Kind: introspection.LIST,
				OfType: &introspection.TypeRef{
					Kind: introspection.SCALAR,
					Name: stringPtr("String"),
				},
			},
			expected: "[String]",
		},
		{
			name: "non-null list of non-null scalars",
			typeRef: &introspection.TypeRef{
				Kind: introspection.NONNULL,
				OfType: &introspection.TypeRef{
					Kind: introspection.LIST,
					OfType: &introspection.TypeRef{
						Kind: introspection.NONNULL,
						OfType: &introspection.TypeRef{
							Kind: introspection.SCALAR,
							Name: stringPtr("String"),
						},
					},
				},
			},
			expected: "[String!]!",
		},
		{
			name: "object type",
			typeRef: &introspection.TypeRef{
				Kind: introspection.OBJECT,
				Name: stringPtr("User"),
			},
			expected: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatType(tt.typeRef)
			if result != tt.expected {
				t.Errorf("Expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatSignature(t *testing.T) {
	tests := []struct {
		name      string
		operation Operation
		expected  string
	}{
		{
			name: "no arguments",
			operation: Operation{
				Name:       "hello",
				Arguments:  []Argument{},
				ReturnType: "String",
			},
			expected: "hello: String",
		},
		{
			name: "single argument",
			operation: Operation{
				Name: "user",
				Arguments: []Argument{
					{Name: "id", Type: "ID!"},
				},
				ReturnType: "User",
			},
			expected: "user(id: ID!): User",
		},
		{
			name: "multiple arguments",
			operation: Operation{
				Name: "users",
				Arguments: []Argument{
					{Name: "limit", Type: "Int"},
					{Name: "offset", Type: "Int"},
				},
				ReturnType: "[User!]!",
			},
			expected: "users(limit: Int, offset: Int): [User!]!",
		},
		{
			name: "argument with default value",
			operation: Operation{
				Name: "users",
				Arguments: []Argument{
					{Name: "limit", Type: "Int", DefaultValue: "10"},
				},
				ReturnType: "[User!]!",
			},
			expected: "users(limit: Int = 10): [User!]!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSignature(tt.operation)
			if result != tt.expected {
				t.Errorf("Expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestDisplaySchema(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	schema := Schema{
		Queries: []Operation{
			{
				Name:        "user",
				Description: "Get a user by ID",
				Arguments: []Argument{
					{Name: "id", Type: "ID!"},
				},
				ReturnType: "User",
			},
		},
		Mutations: []Operation{
			{
				Name:        "createUser",
				Description: "Create a new user",
				Arguments: []Argument{
					{Name: "input", Type: "CreateUserInput!"},
				},
				ReturnType: "User!",
			},
		},
		InputTypes: make(map[string]InputType),
	}

	DisplaySchema(schema)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected sections
	if !strings.Contains(output, "QUERIES") {
		t.Error("Output should contain QUERIES section")
	}
	if !strings.Contains(output, "MUTATIONS") {
		t.Error("Output should contain MUTATIONS section")
	}
	if !strings.Contains(output, "user(id: ID!): User") {
		t.Error("Output should contain user query signature")
	}
	if !strings.Contains(output, "Get a user by ID") {
		t.Error("Output should contain user query description")
	}
	if !strings.Contains(output, "createUser(input: CreateUserInput!): User!") {
		t.Error("Output should contain createUser mutation signature")
	}
}

func TestDisplaySchema_WithDeprecation(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	schema := Schema{
		Queries: []Operation{
			{
				Name:              "oldQuery",
				Description:       "Old query",
				Arguments:         []Argument{},
				ReturnType:        "String",
				IsDeprecated:      true,
				DeprecationReason: "Use newQuery instead",
			},
		},
		InputTypes: make(map[string]InputType),
	}

	DisplaySchema(schema)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify deprecation warning
	if !strings.Contains(output, "DEPRECATED") {
		t.Error("Output should contain DEPRECATED warning")
	}
	if !strings.Contains(output, "Use newQuery instead") {
		t.Error("Output should contain deprecation reason")
	}
}

func TestExtractBaseTypeName(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected string
	}{
		{
			name:     "scalar type",
			typeStr:  "String",
			expected: "String",
		},
		{
			name:     "non-null scalar",
			typeStr:  "String!",
			expected: "String",
		},
		{
			name:     "list of scalars",
			typeStr:  "[String]",
			expected: "String",
		},
		{
			name:     "non-null list of non-null scalars",
			typeStr:  "[String!]!",
			expected: "String",
		},
		{
			name:     "input object",
			typeStr:  "PersonInput",
			expected: "PersonInput",
		},
		{
			name:     "non-null input object",
			typeStr:  "PersonInput!",
			expected: "PersonInput",
		},
		{
			name:     "list of input objects",
			typeStr:  "[PersonInput!]",
			expected: "PersonInput",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBaseTypeName(tt.typeStr)
			if result != tt.expected {
				t.Errorf("Expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestDisplaySchema_WithInputTypes(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	schema := Schema{
		Queries: []Operation{
			{
				Name:        "hello",
				Description: "Say hello",
				Arguments: []Argument{
					{Name: "person", Type: "PersonInput"},
				},
				ReturnType: "String!",
			},
		},
		InputTypes: map[string]InputType{
			"PersonInput": {
				Name: "PersonInput",
				Fields: []InputField{
					{Name: "name", Type: "String!", Description: "Person's name"},
					{Name: "age", Type: "Int"},
				},
			},
		},
	}

	DisplaySchema(schema)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify input type details are displayed
	if !strings.Contains(output, "↳ person fields:") {
		t.Error("Output should contain input type fields header")
	}
	if !strings.Contains(output, "- name: String!") {
		t.Error("Output should contain name field")
	}
	if !strings.Contains(output, "- age: Int") {
		t.Error("Output should contain age field")
	}
	if !strings.Contains(output, "Person's name") {
		t.Error("Output should contain field description")
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func createQueryType() introspection.FullType {
	return introspection.FullType{
		Kind: introspection.OBJECT,
		Name: "Query",
		Fields: []introspection.Field{
			{
				Name:        "hello",
				Description: "Say hello",
				Args:        []introspection.InputValue{},
				Type: introspection.TypeRef{
					Kind: introspection.SCALAR,
					Name: stringPtr("String"),
				},
			},
			{
				Name:        "user",
				Description: "Get a user by ID",
				Args: []introspection.InputValue{
					{
						Name: "id",
						Type: introspection.TypeRef{
							Kind: introspection.NONNULL,
							OfType: &introspection.TypeRef{
								Kind: introspection.SCALAR,
								Name: stringPtr("ID"),
							},
						},
					},
				},
				Type: introspection.TypeRef{
					Kind: introspection.OBJECT,
					Name: stringPtr("User"),
				},
			},
		},
	}
}

func createMutationType() introspection.FullType {
	return introspection.FullType{
		Kind: introspection.OBJECT,
		Name: "Mutation",
		Fields: []introspection.Field{
			{
				Name:        "createUser",
				Description: "Create a new user",
				Args: []introspection.InputValue{
					{
						Name: "input",
						Type: introspection.TypeRef{
							Kind: introspection.NONNULL,
							OfType: &introspection.TypeRef{
								Kind: introspection.INPUTOBJECT,
								Name: stringPtr("CreateUserInput"),
							},
						},
					},
				},
				Type: introspection.TypeRef{
					Kind: introspection.NONNULL,
					OfType: &introspection.TypeRef{
						Kind: introspection.OBJECT,
						Name: stringPtr("User"),
					},
				},
			},
		},
	}
}
