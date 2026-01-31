package graphql

import (
	"testing"
)

func TestOperationStruct(t *testing.T) {
	tests := []struct {
		name string
		op   Operation
	}{
		{
			name: "simple operation with no arguments",
			op: Operation{
				Name:        "hello",
				Description: "Say hello",
				Arguments:   []Argument{},
				ReturnType:  "String",
			},
		},
		{
			name: "operation with arguments",
			op: Operation{
				Name:        "user",
				Description: "Get a user by ID",
				Arguments: []Argument{
					{Name: "id", Type: "ID!", DefaultValue: ""},
				},
				ReturnType: "User",
			},
		},
		{
			name: "deprecated operation",
			op: Operation{
				Name:              "oldQuery",
				Description:       "Old query",
				Arguments:         []Argument{},
				ReturnType:        "String",
				IsDeprecated:      true,
				DeprecationReason: "Use newQuery instead",
			},
		},
		{
			name: "operation with multiple arguments and defaults",
			op: Operation{
				Name:        "users",
				Description: "Get paginated users",
				Arguments: []Argument{
					{Name: "limit", Type: "Int", DefaultValue: "10"},
					{Name: "offset", Type: "Int", DefaultValue: "0"},
				},
				ReturnType: "[User!]!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct fields are properly set
			if tt.op.Name == "" {
				t.Error("Operation name should not be empty")
			}
			if tt.op.ReturnType == "" {
				t.Error("Operation return type should not be empty")
			}
			if tt.op.Arguments == nil {
				t.Error("Operation arguments should not be nil")
			}
		})
	}
}

func TestSchemaStruct(t *testing.T) {
	schema := Schema{
		Queries: []Operation{
			{Name: "user", ReturnType: "User"},
		},
		Mutations: []Operation{
			{Name: "createUser", ReturnType: "User!"},
		},
		Subscriptions: []Operation{
			{Name: "userUpdated", ReturnType: "User!"},
		},
	}

	if len(schema.Queries) != 1 {
		t.Errorf("Expected 1 query, got %d", len(schema.Queries))
	}
	if len(schema.Mutations) != 1 {
		t.Errorf("Expected 1 mutation, got %d", len(schema.Mutations))
	}
	if len(schema.Subscriptions) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(schema.Subscriptions))
	}
}

func TestArgumentStruct(t *testing.T) {
	tests := []struct {
		name     string
		arg      Argument
		hasValue bool
	}{
		{
			name:     "argument without default",
			arg:      Argument{Name: "id", Type: "ID!"},
			hasValue: false,
		},
		{
			name:     "argument with default value",
			arg:      Argument{Name: "limit", Type: "Int", DefaultValue: "10"},
			hasValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.arg.Name == "" {
				t.Error("Argument name should not be empty")
			}
			if tt.arg.Type == "" {
				t.Error("Argument type should not be empty")
			}
			if tt.hasValue && tt.arg.DefaultValue == "" {
				t.Error("Argument should have default value")
			}
		})
	}
}

func TestInputTypeStruct(t *testing.T) {
	inputType := InputType{
		Name: "PersonInput",
		Fields: []InputField{
			{Name: "name", Type: "String!", Description: "Person's name"},
			{Name: "age", Type: "Int", Description: "Person's age"},
		},
	}

	if inputType.Name == "" {
		t.Error("InputType name should not be empty")
	}
	if len(inputType.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(inputType.Fields))
	}
	if inputType.Fields[0].Name != "name" {
		t.Errorf("Expected first field name to be 'name', got %s", inputType.Fields[0].Name)
	}
}

func TestSchemaWithInputTypes(t *testing.T) {
	schema := Schema{
		InputTypes: map[string]InputType{
			"PersonInput": {
				Name: "PersonInput",
				Fields: []InputField{
					{Name: "name", Type: "String!"},
					{Name: "age", Type: "Int"},
				},
			},
		},
	}

	if len(schema.InputTypes) != 1 {
		t.Errorf("Expected 1 input type, got %d", len(schema.InputTypes))
	}
	personInput, ok := schema.InputTypes["PersonInput"]
	if !ok {
		t.Error("PersonInput should be in InputTypes map")
	}
	if len(personInput.Fields) != 2 {
		t.Errorf("Expected PersonInput to have 2 fields, got %d", len(personInput.Fields))
	}
}
