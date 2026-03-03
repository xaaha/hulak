package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileHasTemplateVars(t *testing.T) {
	tempDir := t.TempDir()
	gqlPath := filepath.Join(tempDir, "query.graphql")
	err := os.WriteFile(gqlPath, []byte("query { user(id: {{.userId}}) { id } }"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test gql file: %v", err)
	}
	unquotedPath := filepath.Join(tempDir, "unquoted.graphql")
	err = os.WriteFile(unquotedPath, []byte("query { viewer { id } } {{.needsEnv}}"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create unquoted gql file: %v", err)
	}

	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "env_var_in_header",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n",
			expected: true,
		},
		{
			name:     "env_var_with_spaces",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nheaders:\n  Authorization: \"Bearer {{ .token }}\"\n",
			expected: true,
		},
		{
			name:     "env_var_in_url",
			content:  "---\nkind: GraphQL\nurl: \"{{.graphqlUrl}}\"\n",
			expected: true,
		},
		{
			name:     "env_var_in_body",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nbody:\n  graphql:\n    variables:\n      name: \"{{.userName}}\"\n",
			expected: true,
		},
		{
			name:     "only_getFile_no_env_vars",
			content:  buildGetFileContent("plain.graphql"),
			expected: false,
		},
		{
			name:     "getFile_with_env_vars_in_referenced_file",
			content:  buildGetFileContent("query.graphql"),
			expected: true,
		},
		{
			name:     "getFile_unquoted_with_env_vars_in_referenced_file",
			content:  buildGetFileContentNoQuotes("unquoted.graphql"),
			expected: true,
		},
		{
			name:     "only_getValueOf_no_env_vars",
			content:  buildGetValueOfContent("token", "auth.json"),
			expected: false,
		},
		{
			name:     "no_templates_at_all",
			content:  "---\nkind: GraphQL\nurl: http://example.com/graphql\nmethod: POST\n",
			expected: false,
		},
		{
			name:     "mixed_env_var_and_getFile",
			content:  "---\nkind: GraphQL\nurl: \"{{.baseUrl}}\"\nbody:\n  graphql:\n    query: '{{" + TemplateFuncGetFile + " \"test.graphql\"}}'\n",
			expected: true,
		},
		{
			name:     "multiple_env_vars",
			content:  "---\nkind: GraphQL\nurl: \"https://{{.domain}}/graphql\"\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tc.name+".yaml")
			if tc.name == "only_getFile_no_env_vars" {
				plainPath := filepath.Join(tempDir, "plain.graphql")
				if err := os.WriteFile(plainPath, []byte("query { health }"), 0o600); err != nil {
					t.Fatalf("Failed to create plain gql file: %v", err)
				}
			}
			err := os.WriteFile(filePath, []byte(tc.content), 0o600)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			result := FileHasTemplateVars(filePath)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for content:\n%s", tc.expected, result, tc.content)
			}
		})
	}
}

func TestFileHasTemplateVars_NonexistentFile(t *testing.T) {
	result := FileHasTemplateVars("/nonexistent/path/file.yaml")
	if result != false {
		t.Errorf("Expected false for nonexistent file, got true")
	}
}

func buildGetFileContent(path string) string {
	return "---\nkind: GraphQL\nurl: http://example.com/graphql\nbody:\n  graphql:\n    query: '{{" + TemplateFuncGetFile + " \"" + path + "\"}}'\n"
}

func buildGetFileContentNoQuotes(path string) string {
	return "---\nkind: GraphQL\nurl: http://example.com/graphql\nbody:\n  graphql:\n    query: '{{" + TemplateFuncGetFile + " " + path + "}}'\n"
}

func buildGetValueOfContent(key, fileName string) string {
	return "---\nkind: GraphQL\nurl: http://example.com/graphql\nheaders:\n  Authorization: '{{" + TemplateFuncGetValueOf + " \"" + key + "\" \"" + fileName + "\"}}'\n"
}

func TestMapHasEnvVars(t *testing.T) {
	testCases := []struct {
		name     string
		data     map[string]any
		expected bool
	}{
		{
			name:     "empty_map",
			data:     map[string]any{},
			expected: false,
		},
		{
			name:     "no_template_vars",
			data:     map[string]any{"url": "http://example.com", "method": "GET"},
			expected: false,
		},
		{
			name:     "top_level_env_var",
			data:     map[string]any{"url": "{{.baseUrl}}"},
			expected: true,
		},
		{
			name: "nested_env_var",
			data: map[string]any{
				"headers": map[string]any{
					"Authorization": "Bearer {{.token}}",
				},
			},
			expected: true,
		},
		{
			name: "array_with_env_var",
			data: map[string]any{
				"items": []any{"plain", "{{.secret}}"},
			},
			expected: true,
		},
		{
			name: "getFile_only_no_env_var",
			data: map[string]any{
				"query": "{{" + TemplateFuncGetFile + " \"test.graphql\"}}",
			},
			expected: false,
		},
		{
			name: "getValueOf_only_no_env_var",
			data: map[string]any{
				"auth": "{{" + TemplateFuncGetValueOf + " \"token\" \"auth.json\"}}",
			},
			expected: false,
		},
		{
			name: "deeply_nested_env_var",
			data: map[string]any{
				"body": map[string]any{
					"graphql": map[string]any{
						"variables": map[string]any{
							"name": "{{.userName}}",
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MapHasEnvVars(tc.data)
			if result != tc.expected {
				t.Errorf("MapHasEnvVars() = %v, want %v", result, tc.expected)
			}
		})
	}
}
