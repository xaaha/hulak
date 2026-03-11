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
				if strings.Contains(tt.errMsg, "failed to parse") && !strings.Contains(err.Error(), "Response body:") {
					t.Errorf("Parse error should include response body, got: %s", err.Error())
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

func TestConvertToSchemaExtractsObjectTypes(t *testing.T) {
	queryType := createQueryType()

	introspectionSchema := &introspection.Schema{
		QueryType: queryType,
		Types: []*introspection.FullType{
			&queryType,
			{
				Kind:        introspection.OBJECT,
				Name:        "User",
				Description: "A user account",
				Fields: []introspection.Field{
					{
						Name:        "id",
						Description: "Unique identifier",
						Type: introspection.TypeRef{
							Kind: introspection.NONNULL,
							OfType: &introspection.TypeRef{
								Kind: introspection.SCALAR,
								Name: stringPtr("ID"),
							},
						},
					},
					{
						Name: "name",
						Type: introspection.TypeRef{
							Kind: introspection.SCALAR,
							Name: stringPtr("String"),
						},
					},
					{
						Name: "posts",
						Type: introspection.TypeRef{
							Kind: introspection.LIST,
							OfType: &introspection.TypeRef{
								Kind: introspection.NONNULL,
								OfType: &introspection.TypeRef{
									Kind: introspection.OBJECT,
									Name: stringPtr("Post"),
								},
							},
						},
						Args: []introspection.InputValue{
							{
								Name: "limit",
								Type: introspection.TypeRef{
									Kind: introspection.SCALAR,
									Name: stringPtr("Int"),
								},
							},
						},
					},
				},
			},
			{
				Kind: introspection.OBJECT,
				Name: "Post",
				Fields: []introspection.Field{
					{
						Name: "title",
						Type: introspection.TypeRef{
							Kind: introspection.SCALAR,
							Name: stringPtr("String"),
						},
					},
				},
			},
			{
				Kind: introspection.OBJECT,
				Name: "__Type",
				Fields: []introspection.Field{
					{
						Name: "name",
						Type: introspection.TypeRef{
							Kind: introspection.SCALAR,
							Name: stringPtr("String"),
						},
					},
				},
			},
		},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(schema.ObjectTypes) != 2 {
		t.Fatalf("Expected 2 object types (User, Post), got %d", len(schema.ObjectTypes))
	}

	user, ok := schema.ObjectTypes["User"]
	if !ok {
		t.Fatal("Expected 'User' object type")
	}
	if user.Description != "A user account" {
		t.Errorf("Expected description 'A user account', got %q", user.Description)
	}
	if len(user.Fields) != 3 {
		t.Fatalf("Expected 3 fields on User, got %d", len(user.Fields))
	}
	if user.Fields[0].Name != "id" || user.Fields[0].Type != "ID!" {
		t.Errorf("Unexpected first field: %+v", user.Fields[0])
	}
	if user.Fields[0].Description != "Unique identifier" {
		t.Errorf("Expected field description preserved, got %q", user.Fields[0].Description)
	}
	if user.Fields[2].Name != "posts" || user.Fields[2].Type != "[Post!]" {
		t.Errorf("Unexpected posts field: %+v", user.Fields[2])
	}
	if len(user.Fields[2].Arguments) != 1 || user.Fields[2].Arguments[0].Name != "limit" {
		t.Errorf("Expected posts field to have 'limit' argument, got %+v", user.Fields[2].Arguments)
	}

	if _, ok := schema.ObjectTypes["Post"]; !ok {
		t.Error("Expected 'Post' object type")
	}

	if _, ok := schema.ObjectTypes["__Type"]; ok {
		t.Error("Built-in object type __Type should be filtered out")
	}

	if _, ok := schema.ObjectTypes["Query"]; ok {
		t.Error("Root operation type Query should be filtered out")
	}
}

func TestConvertToSchemaExcludesRootOperationTypes(t *testing.T) {
	queryType := createQueryType()
	mutationType := createMutationType()

	introspectionSchema := &introspection.Schema{
		QueryType:    queryType,
		MutationType: &mutationType,
		Types: []*introspection.FullType{
			&queryType,
			&mutationType,
			{
				Kind: introspection.OBJECT,
				Name: "User",
				Fields: []introspection.Field{
					{
						Name: "id",
						Type: introspection.TypeRef{
							Kind: introspection.SCALAR,
							Name: stringPtr("ID"),
						},
					},
				},
			},
		},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, ok := schema.ObjectTypes["Query"]; ok {
		t.Error("Root type Query should not be in ObjectTypes")
	}
	if _, ok := schema.ObjectTypes["Mutation"]; ok {
		t.Error("Root type Mutation should not be in ObjectTypes")
	}
	if _, ok := schema.ObjectTypes["User"]; !ok {
		t.Error("Expected User in ObjectTypes")
	}
}

func TestConvertToSchemaExtractsUnionTypes(t *testing.T) {
	queryType := createQueryType()

	introspectionSchema := &introspection.Schema{
		QueryType: queryType,
		Types: []*introspection.FullType{
			&queryType,
			{
				Kind:        introspection.UNION,
				Name:        "SearchResult",
				Description: "A search result union",
				PossibleTypes: []introspection.TypeRef{
					{Kind: introspection.OBJECT, Name: stringPtr("User")},
					{Kind: introspection.OBJECT, Name: stringPtr("Post")},
				},
			},
			{
				Kind: introspection.OBJECT,
				Name: "User",
				Fields: []introspection.Field{
					{Name: "id", Type: introspection.TypeRef{Kind: introspection.SCALAR, Name: stringPtr("ID")}},
				},
			},
			{
				Kind: introspection.OBJECT,
				Name: "Post",
				Fields: []introspection.Field{
					{Name: "title", Type: introspection.TypeRef{Kind: introspection.SCALAR, Name: stringPtr("String")}},
				},
			},
		},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(schema.UnionTypes) != 1 {
		t.Fatalf("Expected 1 union type, got %d", len(schema.UnionTypes))
	}

	ut, ok := schema.UnionTypes["SearchResult"]
	if !ok {
		t.Fatal("Expected 'SearchResult' union type")
	}
	if ut.Description != "A search result union" {
		t.Errorf("Expected description 'A search result union', got %q", ut.Description)
	}
	if len(ut.PossibleTypes) != 2 {
		t.Fatalf("Expected 2 possible types, got %d", len(ut.PossibleTypes))
	}
	if ut.PossibleTypes[0] != "User" || ut.PossibleTypes[1] != "Post" {
		t.Errorf("Expected [User, Post], got %v", ut.PossibleTypes)
	}
}

func TestConvertToSchemaExtractsInterfaceTypes(t *testing.T) {
	queryType := createQueryType()

	introspectionSchema := &introspection.Schema{
		QueryType: queryType,
		Types: []*introspection.FullType{
			&queryType,
			{
				Kind:        introspection.INTERFACE,
				Name:        "Node",
				Description: "An object with an ID",
				Fields: []introspection.Field{
					{
						Name: "id",
						Type: introspection.TypeRef{Kind: introspection.NONNULL, OfType: &introspection.TypeRef{Kind: introspection.SCALAR, Name: stringPtr("ID")}},
					},
				},
				PossibleTypes: []introspection.TypeRef{
					{Kind: introspection.OBJECT, Name: stringPtr("User")},
					{Kind: introspection.OBJECT, Name: stringPtr("Post")},
				},
			},
		},
	}

	schema, err := ConvertToSchema(introspectionSchema)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(schema.InterfaceTypes) != 1 {
		t.Fatalf("Expected 1 interface type, got %d", len(schema.InterfaceTypes))
	}

	it, ok := schema.InterfaceTypes["Node"]
	if !ok {
		t.Fatal("Expected 'Node' interface type")
	}
	if it.Description != "An object with an ID" {
		t.Errorf("Expected description 'An object with an ID', got %q", it.Description)
	}
	if len(it.Fields) != 1 {
		t.Fatalf("Expected 1 shared field, got %d", len(it.Fields))
	}
	if it.Fields[0].Name != "id" || it.Fields[0].Type != "ID!" {
		t.Errorf("Expected shared field id:ID!, got %s:%s", it.Fields[0].Name, it.Fields[0].Type)
	}
	if len(it.PossibleTypes) != 2 {
		t.Fatalf("Expected 2 possible types, got %d", len(it.PossibleTypes))
	}
	if it.PossibleTypes[0] != "User" || it.PossibleTypes[1] != "Post" {
		t.Errorf("Expected [User, Post], got %v", it.PossibleTypes)
	}
}

func TestExtractPossibleTypeNames(t *testing.T) {
	refs := []introspection.TypeRef{
		{Kind: introspection.OBJECT, Name: stringPtr("A")},
		{Kind: introspection.OBJECT, Name: stringPtr("B")},
		{Kind: introspection.OBJECT, Name: nil},
	}
	names := extractPossibleTypeNames(refs)
	if len(names) != 2 {
		t.Fatalf("Expected 2 names (nil skipped), got %d", len(names))
	}
	if names[0] != "A" || names[1] != "B" {
		t.Errorf("Expected [A, B], got %v", names)
	}

	empty := extractPossibleTypeNames(nil)
	if len(empty) != 0 {
		t.Errorf("Expected empty slice for nil input, got %d", len(empty))
	}
}

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
