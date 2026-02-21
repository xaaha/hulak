package graphql

import (
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
		{
			name:      "HTML response",
			jsonData:  `<!DOCTYPE html><html><body><h1>404 Not Found</h1></body></html>`,
			wantError: true,
			errMsg:    "failed to parse",
		},
		{
			name:      "XML response",
			jsonData:  `<?xml version="1.0"?><error><message>Forbidden</message></error>`,
			wantError: true,
			errMsg:    "failed to parse",
		},
		{
			name:      "plain text response",
			jsonData:  `Internal Server Error`,
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
				if strings.Contains(tt.errMsg, "failed to parse") && !strings.Contains(err.Error(), "Response preview:") {
					t.Errorf("Parse error should include response preview, got: %s", err.Error())
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

func TestConvertToSchemaExtractsEnumTypes(t *testing.T) {
	deprecationReason := "Use ACTIVE instead"
	queryType := createQueryType()

	introspectionSchema := &introspection.Schema{
		QueryType: queryType,
		Types: []*introspection.FullType{
			&queryType,
			{
				Kind:        introspection.ENUM,
				Name:        "Status",
				Description: "User status",
				EnumValues: []introspection.EnumValue{
					{Name: "ACTIVE", Description: "Active user"},
					{Name: "INACTIVE", Description: "Inactive user", IsDeprecated: true, DeprecationReason: &deprecationReason},
				},
			},
			{
				Kind: introspection.ENUM,
				Name: "__DirectiveLocation",
				EnumValues: []introspection.EnumValue{
					{Name: "QUERY"},
				},
			},
		},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(schema.EnumTypes) != 1 {
		t.Fatalf("Expected 1 enum type, got %d", len(schema.EnumTypes))
	}

	statusEnum, ok := schema.EnumTypes["Status"]
	if !ok {
		t.Fatal("Expected 'Status' enum type")
	}
	if statusEnum.Description != "User status" {
		t.Errorf("Expected description 'User status', got %q", statusEnum.Description)
	}
	if len(statusEnum.Values) != 2 {
		t.Fatalf("Expected 2 enum values, got %d", len(statusEnum.Values))
	}
	if statusEnum.Values[0].Name != "ACTIVE" {
		t.Errorf("Expected first value 'ACTIVE', got %q", statusEnum.Values[0].Name)
	}
	if statusEnum.Values[1].IsDeprecated != true {
		t.Error("Expected INACTIVE to be deprecated")
	}
	if statusEnum.Values[1].DeprecationReason != "Use ACTIVE instead" {
		t.Errorf("Expected deprecation reason, got %q", statusEnum.Values[1].DeprecationReason)
	}

	if _, ok := schema.EnumTypes["__DirectiveLocation"]; ok {
		t.Error("Built-in enum __DirectiveLocation should be filtered out")
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "html_response",
			body:     `<!DOCTYPE html><html><body>Not Found</body></html>`,
			expected: "HTML",
		},
		{
			name:     "xml_response",
			body:     `<?xml version="1.0"?><error>Forbidden</error>`,
			expected: "XML",
		},
		{
			name:     "plain_text",
			body:     "Internal Server Error",
			expected: "non-JSON",
		},
		{
			name:     "empty_body",
			body:     "",
			expected: "non-JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectContentType(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxLen   int
		contains string
		exact    bool
	}{
		{
			name:     "short_body_unchanged",
			body:     "short",
			maxLen:   100,
			contains: "short",
			exact:    true,
		},
		{
			name:     "exact_limit_unchanged",
			body:     "12345",
			maxLen:   5,
			contains: "12345",
			exact:    true,
		},
		{
			name:     "long_body_truncated",
			body:     "abcdefghij",
			maxLen:   5,
			contains: "... (truncated)",
		},
		{
			name:     "empty_body",
			body:     "",
			maxLen:   100,
			contains: "",
			exact:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateBody(tt.body, tt.maxLen)
			if tt.exact {
				if result != tt.contains {
					t.Errorf("Expected exactly %q, got %q", tt.contains, result)
				}
			} else {
				if !strings.Contains(result, tt.contains) {
					t.Errorf("Expected result to contain %q, got %q", tt.contains, result)
				}
			}
		})
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
