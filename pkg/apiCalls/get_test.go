package apicalls

import (
	"io"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestFullUrl(t *testing.T) {
	tests := []struct {
		name     string
		baseUrl  string
		expected string
		params   []utils.KeyValuePair
	}{
		// Test with no parameters
		{
			name:     "No parameters",
			baseUrl:  "https://api.example.com/resource",
			params:   []utils.KeyValuePair{},
			expected: "https://api.example.com/resource",
		},
		// Test with a single parameter
		{
			name:    "Single parameter",
			baseUrl: "https://api.example.com/resource",
			params: []utils.KeyValuePair{
				{Key: "search", Value: "golang"},
			},
			expected: "https://api.example.com/resource?search=golang",
		},
		// Test with multiple parameters
		{
			name:    "Multiple parameters",
			baseUrl: "https://api.example.com/resource",
			params: []utils.KeyValuePair{
				{Key: "search", Value: "golang"},
				{Key: "limit", Value: "10"},
				{Key: "sort", Value: "desc"},
			},
			expected: "https://api.example.com/resource?limit=10&search=golang&sort=desc",
		},
		// Test with special characters in parameters
		{
			name:    "Special characters in parameters",
			baseUrl: "https://api.example.com/resource",
			params: []utils.KeyValuePair{
				{Key: "search", Value: "go programming"},
				{Key: "filter", Value: "name&value"},
			},
			expected: "https://api.example.com/resource?filter=name%26value&search=go+programming",
		},
		// Test with empty parameter values
		{
			name:    "Empty parameter values",
			baseUrl: "https://api.example.com/resource",
			params: []utils.KeyValuePair{
				{Key: "search", Value: ""},
				{Key: "limit", Value: "10"},
			},
			expected: "https://api.example.com/resource?limit=10&search=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullUrl := PrepareUrl(tt.baseUrl, tt.params...)
			if fullUrl != tt.expected {
				t.Errorf("FullUrl() = %v, want %v", fullUrl, tt.expected)
			}
		})
	}
}

func TestEncodeXwwwFormUrlBody(t *testing.T) {
	tests := []struct {
		name        string
		expected    string
		input       []utils.KeyValuePair
		expectError bool
	}{
		{
			name: "valid key-value pairs",
			input: []utils.KeyValuePair{
				{Key: "username", Value: "john_doe"},
				{Key: "password", Value: "secret"},
			},
			expected:    "password=secret&username=john_doe",
			expectError: false,
		},
		{
			name: "ignore empty key-value pairs",
			input: []utils.KeyValuePair{
				{Key: "username", Value: "john_doe"},
				{Key: "", Value: "secret"},
				{Key: "age", Value: ""},
				{Key: "", Value: ""},
				{Key: "location", Value: "USA"},
			},
			expected:    "location=USA&username=john_doe",
			expectError: false,
		},
		{
			name: "handle special characters",
			input: []utils.KeyValuePair{
				{Key: "name", Value: "John Doe"},
				{Key: "address", Value: "123 Main St. #500"},
				{Key: "email", Value: "john.doe@example.com"},
			},
			expected:    "address=123+Main+St.+%23500&email=john.doe%40example.com&name=John+Doe",
			expectError: false,
		},
		{
			name: "single key-value pair",
			input: []utils.KeyValuePair{
				{Key: "username", Value: "john_doe"},
			},
			expected:    "username=john_doe",
			expectError: false,
		},
		{
			input:       []utils.KeyValuePair{},
			name:        "empty input",
			expectError: true,
		},
		{
			name: "key-value pairs with same key",
			input: []utils.KeyValuePair{
				{Key: "key", Value: "first_value"},
				{Key: "key", Value: "second_value"},
			},
			expected:    "key=second_value",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := EncodeXwwwFormUrlBody(tt.input)

			// Check if an error is expected
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return // Skip further checks if error is expected and received
			}

			// No error expected; check result
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read from reader: %v", err)
			}
			result := string(body)

			// Compare the result to expected output
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEncodeFormData(t *testing.T) {
	tests := []struct {
		name                      string
		expectedContentTypePrefix string
		input                     []utils.KeyValuePair
		expectedBodyContains      []string
		expectError               bool
	}{
		{
			name: "valid key-value pairs",
			input: []utils.KeyValuePair{
				{Key: "username", Value: "john_doe"},
				{Key: "password", Value: "secret"},
			},
			expectError:               false,
			expectedContentTypePrefix: "multipart/form-data; boundary=",
			expectedBodyContains:      []string{"username", "john_doe", "password", "secret"},
		},
		{
			name: "ignore empty key-value pairs",
			input: []utils.KeyValuePair{
				{Key: "username", Value: "john_doe"},
				{Key: "", Value: "secret"},
				{Key: "age", Value: ""},
				{Key: "", Value: ""},
				{Key: "location", Value: "USA"},
			},
			expectError:               false,
			expectedContentTypePrefix: "multipart/form-data; boundary=",
			expectedBodyContains:      []string{"username", "john_doe", "location", "USA"},
		},
		{
			name:        "empty input",
			input:       []utils.KeyValuePair{},
			expectError: true,
		},
		{
			name: "single key-value pair",
			input: []utils.KeyValuePair{
				{Key: "username", Value: "john_doe"},
			},
			expectError:               false,
			expectedContentTypePrefix: "multipart/form-data; boundary=",
			expectedBodyContains:      []string{"username", "john_doe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, contentType, err := EncodeFormData(tt.input)

			// Check if an error is expected
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return // Skip further checks if error is expected and received
			}

			// No error expected; check result
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check content type prefix
			if !strings.HasPrefix(contentType, tt.expectedContentTypePrefix) {
				t.Errorf(
					"expected Content-Type prefix %v, got %v",
					tt.expectedContentTypePrefix,
					contentType,
				)
			}

			// Check if expected fields are present in the body
			body, err := io.ReadAll(payload)
			if err != nil {
				t.Fatalf("failed to read from reader: %v", err)
			}
			for _, expected := range tt.expectedBodyContains {
				if !strings.Contains(string(body), expected) {
					t.Errorf("expected body to contain %v, but it does not", expected)
				}
			}
		})
	}
}

func TestEncodeGraphQlBody(t *testing.T) {
	tests := []struct {
		variables    map[string]interface{}
		name         string
		query        string
		expectedBody string
		expectError  bool
	}{
		{
			name:         "valid query and variables",
			query:        "query Hello($name: String!) { hello(name: $name) }",
			variables:    map[string]interface{}{"name": "John"},
			expectError:  false,
			expectedBody: `{"query":"query Hello($name: String!) { hello(name: $name) }","variables":{"name":"John"}}`,
		},
		{
			name:         "no variables",
			query:        "query Hello { hello }",
			variables:    map[string]interface{}{},
			expectError:  false,
			expectedBody: `{"query":"query Hello { hello }","variables":{}}`,
		},
		{
			name:         "nil variables",
			query:        "query Hello { hello }",
			variables:    nil,
			expectError:  false,
			expectedBody: `{"query":"query Hello { hello }","variables":null}`,
		},
		{
			name:         "empty query",
			query:        "",
			variables:    map[string]interface{}{"name": "John"},
			expectError:  false,
			expectedBody: `{"query":"","variables":{"name":"John"}}`,
		},
		{
			name:        "invalid JSON in variables",
			query:       "query Hello { hello }",
			variables:   map[string]interface{}{"invalid": make(chan int)}, // unsupported type
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := EncodeGraphQlBody(tt.query, tt.variables)

			// Check if an error is expected
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return // Skip further checks if error is expected and received
			}

			// No error expected; check result
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Read the result to verify the encoded JSON
			body, err := io.ReadAll(payload)
			if err != nil {
				t.Fatalf("failed to read from reader: %v", err)
			}

			if string(body) != tt.expectedBody {
				t.Errorf("expected %v, got %v", tt.expectedBody, string(body))
			}
		})
	}
}