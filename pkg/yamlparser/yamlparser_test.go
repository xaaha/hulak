package yamlparser

import (
	"reflect"
	"strings"
	"testing"
)

func TestStringVariableSubstitution(t *testing.T) {
	// Test substitution logic for string variables
	testCases := []struct {
		name           string
		input          map[string]any
		envVars        map[string]any
		expectedOutput map[string]any
		checkPaths     []string // Path to check in the result map (e.g., "key.subkey")
		expectedValues []any    // Expected values at the check paths
	}{
		{
			name: "String with template syntax is substituted",
			input: map[string]any{
				"templateString": "Hello {{.name}}!",
			},
			envVars: map[string]any{
				"name": "World",
			},
			expectedOutput: map[string]any{
				"templateString": "Hello World!",
			},
			checkPaths:     []string{"templateString"},
			expectedValues: []any{"Hello World!"},
		},
		{
			name: "String without template syntax is preserved",
			input: map[string]any{
				"plainString": "Hello World!",
			},
			envVars: map[string]any{
				"Hello World!": "Should not replace this",
			},
			expectedOutput: map[string]any{
				"plainString": "Hello World!",
			},
			checkPaths:     []string{"plainString"},
			expectedValues: []any{"Hello World!"},
		},
		{
			name: "String matching env var name but without template syntax is preserved",
			input: map[string]any{
				"enumValue": "email",
			},
			envVars: map[string]any{
				"email": "user@example.com",
			},
			expectedOutput: map[string]any{
				"enumValue": "email", // Should remain "email", not become "user@example.com"
			},
			checkPaths:     []string{"enumValue"},
			expectedValues: []any{"email"},
		},
		{
			name: "Nested map with template variables",
			input: map[string]any{
				"graphql": map[string]any{
					"query": "query Test { test }",
					"variables": map[string]any{
						"stringVar":   "regular value",
						"enumVar":     "email",              // Should be preserved even though "email" exists in env
						"templateVar": "{{.dynamic_value}}", // Should be substituted
					},
				},
			},
			envVars: map[string]any{
				"email":         "user@example.com",
				"dynamic_value": "substituted value",
			},
			checkPaths: []string{
				"graphql.variables.stringVar",
				"graphql.variables.enumVar",
				"graphql.variables.templateVar",
			},
			expectedValues: []any{
				"regular value",
				"email",
				"substituted value",
			},
		},
		{
			name: "Complex nested structure with mixed variable types",
			input: map[string]any{
				"api": map[string]any{
					"url": "https://{{.api_host}}/v1",
					"config": map[string]any{
						"timeout": 30,
						"retry":   true,
						"headers": map[string]any{
							"Content-Type":  "application/json",
							"Authorization": "Bearer {{.api_key}}",
						},
					},
				},
			},
			envVars: map[string]any{
				"api_host": "example.com",
				"api_key":  "secret-key-12345",
			},
			checkPaths: []string{
				"api.url",
				"api.config.timeout",
				"api.config.retry",
				"api.config.headers.Content-Type",
				"api.config.headers.Authorization",
			},
			expectedValues: []any{
				"https://example.com/v1",
				30,
				true,
				"application/json",
				"Bearer secret-key-12345",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := replaceVarsWithValues(tc.input, tc.envVars)
			if err != nil {
				t.Fatalf("replaceVarsWithValues returned an error: %v", err)
			}

			// Check each expected path and value
			for i, path := range tc.checkPaths {
				expectedVal := tc.expectedValues[i]
				actualVal := getValueByPath(result, path)

				// Use DeepEqual for complex types, direct comparison for simple types
				if !reflect.DeepEqual(actualVal, expectedVal) {
					t.Errorf("Expected value at path %s to be %v, got %v", path, expectedVal, actualVal)
				}
			}
		})
	}
}

// Helper function to get a value from a nested map using a dot-separated path
func getValueByPath(m map[string]any, path string) any {
	pathParts := strings.Split(path, ".")
	current := m

	// Navigate through all but the last part of the path
	for i := 0; i < len(pathParts)-1; i++ {
		key := pathParts[i]
		if nestedMap, ok := current[key].(map[string]any); ok {
			current = nestedMap
		} else {
			return nil // Path doesn't exist
		}
	}

	// Return the value at the final part of the path
	return current[pathParts[len(pathParts)-1]]
}
