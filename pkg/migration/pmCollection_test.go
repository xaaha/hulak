package migration

import (
	"sort"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestURLToYAML(t *testing.T) {
	// Helper function to normalize YAML for comparison
	normalizeYAML := func(yaml string) string {
		return strings.TrimSpace(yaml)
	}

	// Helper function to compare expected vs actual
	compareYAML := func(t *testing.T, expected, actual string) {
		t.Helper()
		expectedNorm := normalizeYAML(expected)
		actualNorm := normalizeYAML(actual)

		if expectedNorm != actualNorm {
			t.Errorf("YAML mismatch:\nExpected:\n%s\n\nActual:\n%s", expectedNorm, actualNorm)
		}
	}

	t.Run("Basic URL with query parameters", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/get?foo=bar1",
			Query: []KeyValuePair{
				{Key: "foo", Value: "bar1"},
			},
		}
		expected := `url: "{{.baseUrl}}/get"
urlparams:
  foo: bar1`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with multiple query parameters", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/users?id=123&name=john",
			Query: []KeyValuePair{
				{Key: "id", Value: "123"},
				{Key: "name", Value: "john"},
			},
		}
		expected := `url: "{{.baseUrl}}/users"
urlparams:
  id: "123"
  name: john`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with no query parameters at all", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/api/v1/health",
		}
		expected := `url: "{{.baseUrl}}/api/v1/health"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with multiple template variables", func(t *testing.T) {
		input := PMURL{
			Raw: "{{protocol}}://{{baseUrl}}/api/{{version}}/users",
		}
		expected := `url: "{{.protocol}}://{{.baseUrl}}/api/{{.version}}/users"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with template variables in query parameters", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/search?term={{searchTerm}}&lang={{language}}",
			Query: []KeyValuePair{
				{Key: "term", Value: "{{searchTerm}}"},
				{Key: "lang", Value: "{{language}}"},
			},
		}

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.Contains(result, "url: ") ||
			!strings.Contains(result, "{{.baseUrl}}/search") ||
			!strings.Contains(result, "urlparams:") ||
			!strings.Contains(result, "lang: ") ||
			!strings.Contains(result, "{{.language}}") ||
			!strings.Contains(result, "term: ") ||
			!strings.Contains(result, "{{.searchTerm}}") {
			t.Errorf("YAML does not contain expected values: %s", result)
		}

		// Print both values for debugging
		t.Logf("Result YAML:\n%s", result)
	})

	t.Run("URL with encoded characters", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/path%20with%20spaces?query=value%20with%20spaces",
			Query: []KeyValuePair{
				{Key: "query", Value: "value with spaces"},
			},
		}
		expected := `url: "{{.baseUrl}}/path%20with%20spaces"
urlparams:
  query: value with spaces`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with query parameters that already have dot notation", func(t *testing.T) {
		input := PMURL{
			Raw: "{{.baseUrl}}/api?token={{.token}}",
			Query: []KeyValuePair{
				{Key: "token", Value: "{{.token}}"},
			},
		}
		expected := `url: "{{.baseUrl}}/api"
urlparams:
  token: "{{.token}}"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with empty query parameters", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/search?q=",
			Query: []KeyValuePair{
				{Key: "q", Value: ""},
			},
		}
		expected := `url: "{{.baseUrl}}/search"
urlparams:
  q: ""`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with mixed template notation", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/{{path}}?id={{.id}}&type={{type}}",
			Query: []KeyValuePair{
				{Key: "id", Value: "{{.id}}"},
				{Key: "type", Value: "{{type}}"},
			},
		}
		expected := `url: "{{.baseUrl}}/{{.path}}"
urlparams:
  id: "{{.id}}"
  type: "{{.type}}"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with fragments", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/docs?section=api#overview",
			Query: []KeyValuePair{
				{Key: "section", Value: "api"},
			},
		}
		expected := `url: "{{.baseUrl}}/docs"
urlparams:
  section: api`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Absolute URL with protocol", func(t *testing.T) {
		input := PMURL{
			Raw: "https://example.com/api?key={{apiKey}}",
			Query: []KeyValuePair{
				{Key: "key", Value: "{{apiKey}}"},
			},
		}
		expected := `url: https://example.com/api
urlparams:
  key: "{{.apiKey}}"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with query parameters but empty Query field", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/api?param1=value1&param2=value2",
			// Query field is intentionally empty
		}
		expected := `url: "{{.baseUrl}}/api"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with multiple question marks", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/api/search?q=what?&sort=asc",
			Query: []KeyValuePair{
				{Key: "q", Value: "what?"},
				{Key: "sort", Value: "asc"},
			},
		}
		expected := `url: "{{.baseUrl}}/api/search"
urlparams:
  q: what?
  sort: asc`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Empty URL", func(t *testing.T) {
		input := PMURL{
			Raw: "",
		}
		expected := `url: ""`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URL with numerical template values", func(t *testing.T) {
		input := PMURL{
			Raw: "{{baseUrl}}/api/users/{{userId}}",
			Query: []KeyValuePair{
				{Key: "version", Value: "{{version}}"},
			},
		}
		expected := `url: "{{.baseUrl}}/api/users/{{.userId}}"
urlparams:
  version: "{{.version}}"`

		result, err := UrlToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})
}

func TestHeaderToYAML(t *testing.T) {
	// Helper function to normalize YAML for comparison
	normalizeYAML := func(yaml string) string {
		return strings.TrimSpace(yaml)
	}

	// Helper function to compare expected vs actual
	compareYAML := func(t *testing.T, expected, actual string) {
		t.Helper()
		expectedNorm := normalizeYAML(expected)
		actualNorm := normalizeYAML(actual)

		if expectedNorm != actualNorm {
			t.Errorf("YAML mismatch:\nExpected:\n%s\n\nActual:\n%s", expectedNorm, actualNorm)
		}
	}

	t.Run("Single header", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "Content-Type", Value: "application/json", Type: "text"},
		}
		expected := `headers:
  Content-Type: application/json`

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Multiple headers", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "Content-Type", Value: "application/json", Type: "text"},
			{Key: "Authorization", Value: "Bearer token123", Type: "text"},
			{Key: "Accept", Value: "*/*", Type: "text"},
		}

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check if result contains all headers
		if !strings.Contains(result, "headers:") {
			t.Errorf("YAML doesn't include headers section")
		}
		if !strings.Contains(result, "Content-Type: application/json") {
			t.Errorf("YAML doesn't include Content-Type header")
		}
		if !strings.Contains(result, "Authorization: Bearer token123") {
			t.Errorf("YAML doesn't include Authorization header")
		}
		if !strings.Contains(result, "Accept: \"*/*\"") {
			t.Errorf("YAML doesn't include Accept header")
		}
	})

	t.Run("Empty headers", func(t *testing.T) {
		input := []KeyValuePair{}
		expected := ""

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Headers with template variables", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "Authorization", Value: "Bearer {{token}}", Type: "text"},
			{Key: "X-API-Version", Value: "{{apiVersion}}", Type: "text"},
		}

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check for template variables with dots
		if !strings.Contains(result, "Bearer {{.token}}") {
			t.Errorf("YAML doesn't properly format template in Authorization: %s", result)
		}
		if !strings.Contains(result, "{{.apiVersion}}") {
			t.Errorf("YAML doesn't properly format template in X-API-Version: %s", result)
		}
	})

	t.Run("Headers with template variables in keys", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "X-{{customHeader}}", Value: "custom value", Type: "text"},
		}

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "X-{{.customHeader}}:") {
			t.Errorf("YAML doesn't properly format template in header key: %s", result)
		}
	})

	t.Run("Headers with empty values", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "X-Empty", Value: "", Type: "text"},
		}
		expected := `headers:
  X-Empty: ""`

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Headers with special characters", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "X-Special", Value: "value with: colon and # hash", Type: "text"},
		}

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// YAML should quote this value as it contains special characters
		if !strings.Contains(result, "X-Special:") {
			t.Errorf("YAML doesn't include X-Special header: %s", result)
		}
	})

	t.Run("Headers with already dotted templates", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "Authorization", Value: "Bearer {{.token}}", Type: "text"},
		}
		expected := `headers:
  Authorization: Bearer {{.token}}`

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Case sensitivity in header keys", func(t *testing.T) {
		input := []KeyValuePair{
			{Key: "Content-Type", Value: "application/json", Type: "text"},
			{
				Key:   "content-type",
				Value: "text/plain",
				Type:  "text",
			},
		}

		result, err := HeaderToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Both headers should be present (YAML is case-sensitive)
		if !strings.Contains(result, "Content-Type: application/json") {
			t.Errorf("YAML doesn't include Content-Type header: %s", result)
		}
		if !strings.Contains(result, "content-type: text/plain") {
			t.Errorf("YAML doesn't include content-type header: %s", result)
		}
	})
}

func TestBodyToYaml(t *testing.T) {
	// Helper function to normalize YAML for comparison
	normalizeYAML := func(yamlStr string) string {
		yamlStr = strings.TrimSpace(yamlStr)              // Remove extra spaces
		yamlStr = strings.ReplaceAll(yamlStr, `\n`, "\n") // Convert `\n` to actual newlines
		var data map[string]any
		if err := yaml.Unmarshal([]byte(yamlStr), &data); err != nil {
			return yamlStr
		}
		sortedData := make(map[string]any)
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sortedData[k] = data[k]
		}
		normalized, err := yaml.Marshal(sortedData)
		if err != nil {
			return yamlStr
		}
		return strings.TrimSpace(string(normalized))
	}

	// Helper function to compare expected vs actual
	compareYAML := func(t *testing.T, expected, actual string) {
		t.Helper()
		expectedNorm := normalizeYAML(expected)
		actualNorm := normalizeYAML(actual)

		if expectedNorm != actualNorm {
			t.Errorf("YAML mismatch:\nExpected:\n%s\n\nActual:\n%s", expectedNorm, actualNorm)
		}
	}

	t.Run("Raw body", func(t *testing.T) {
		input := Body{
			Mode: "raw",
			Raw:  `{"name": "John", "age": 30}`,
		}
		expected := `raw: '{"name": "John", "age": 30}'`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Raw body with template", func(t *testing.T) {
		input := Body{
			Mode: "raw",
			Raw:  `{"name": "{{name}}", "token": "{{token}}"}`,
		}
		expected := `raw: '{"name": "{{.name}}", "token": "{{.token}}"}'`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URLEncoded form data", func(t *testing.T) {
		input := Body{
			Mode: "urlencoded",
			URLEncoded: []KeyValuePair{
				{Key: "username", Value: "john_doe"},
				{Key: "password", Value: "secret123"},
			},
		}
		expected := `urlencodedformdata:
  username: john_doe
  password: secret123`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("URLEncoded with template variables", func(t *testing.T) {
		input := Body{
			Mode: "urlencoded",
			URLEncoded: []KeyValuePair{
				{Key: "username", Value: "{{username}}"},
				{Key: "apiKey", Value: "{{apiKey}}"},
			},
		}
		expected := `urlencodedformdata:
  username: "{{.username}}"
  apiKey: "{{.apiKey}}"`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Form data", func(t *testing.T) {
		input := Body{
			Mode: "formdata",
			FormData: []KeyValuePair{
				{Key: "file", Value: "@/path/to/file.jpg"},
				{Key: "description", Value: "Profile picture"},
			},
		}
		expected := `formdata: 
  description: Profile picture
  file: "@/path/to/file.jpg"`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Form data with template", func(t *testing.T) {
		input := Body{
			Mode: "formdata",
			FormData: []KeyValuePair{
				{Key: "token", Value: "{{authToken}}"},
				{Key: "user", Value: "{{userId}}"},
			},
		}
		expected := `formdata:
  token: "{{.authToken}}"
  user: "{{.userId}}"`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("GraphQL query", func(t *testing.T) {
		input := Body{
			Mode: "graphql",
			GraphQL: &GraphQl{
				Query: `query GetUser {
  user(id: "1") {
    name
    email
  }
}`,
				Variables: `{"id": "1"}`,
			},
		}
		expected := `graphql:
  query: "query GetUser {
  user(id: \"1\") {
    name
    email
  }
}"
  variables:
    id: "1"`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("GraphQL with template variables", func(t *testing.T) {
		input := Body{
			Mode: "graphql",
			GraphQL: &GraphQl{
				Query: `query GetUser {
  user(id: "{{userId}}") {
    name
    email
  }
}`,
				Variables: `{
                "id": "{{userId}}"
            }`,
			},
		}
		expected := `graphql:
  query: "query GetUser {
  user(id: \"{{.userId}}\") {
    name
    email
  }
}"
  variables:
    id: "{{.userId}}"`

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("No body (mode: none)", func(t *testing.T) {
		input := Body{
			Mode: "none",
		}
		expected := ``

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Empty body (no mode)", func(t *testing.T) {
		input := Body{
			Mode: "",
		}
		expected := ``

		result, err := BodyToYaml(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})

	t.Run("Unsupported body mode", func(t *testing.T) {
		input := Body{
			Mode: "unsupported",
		}

		_, err := BodyToYaml(input)
		if err == nil {
			t.Fatal("Expected error for unsupported body mode, but got nil")
		}
	})
}
