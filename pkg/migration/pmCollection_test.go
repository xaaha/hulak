package migration

import (
	"strings"
	"testing"
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
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

		result, err := URLToYAML(input)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		compareYAML(t, expected, result)
	})
}
