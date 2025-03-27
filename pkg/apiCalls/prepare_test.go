package apicalls

import (
	"io"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/yamlparser"
)

func TestFullUrl(t *testing.T) {
	tests := []struct {
		params   map[string]string
		name     string
		baseURL  string
		expected string
	}{
		// Test with no parameters
		{
			name:     "No parameters",
			baseURL:  "https://api.example.com/resource",
			params:   map[string]string{},
			expected: "https://api.example.com/resource",
		},
		// Test with a single parameter
		{
			name:     "Single parameter",
			baseURL:  "https://api.example.com/resource",
			params:   map[string]string{"search": "golang"},
			expected: "https://api.example.com/resource?search=golang",
		},
		// Test with multiple parameters
		{
			name:    "Multiple parameters",
			baseURL: "https://api.example.com/resource",
			params: map[string]string{
				"search": "golang",
				"limit":  "10",
				"sort":   "desc",
			},
			expected: "https://api.example.com/resource?limit=10&search=golang&sort=desc",
		},
		// Test with special characters in parameters
		{
			name:    "Special characters in parameters",
			baseURL: "https://api.example.com/resource",
			params: map[string]string{
				"search": "go programming",
				"filter": "name&value",
			},
			expected: "https://api.example.com/resource?filter=name%26value&search=go+programming",
		},
		// Test with empty parameter values
		{
			name:    "Empty parameter values",
			baseURL: "https://api.example.com/resource",
			params: map[string]string{
				"search": "",
				"limit":  "10",
			},
			expected: "https://api.example.com/resource?limit=10&search=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullUrl := PrepareUrl(tt.baseURL, tt.params)
			if fullUrl != tt.expected {
				t.Errorf("FullUrl() = %v, want %v", fullUrl, tt.expected)
			}
		})
	}
}

func TestEncodeXwwwFormUrlBody(t *testing.T) {
	tests := []struct {
		input       map[string]string
		name        string
		expected    string
		expectError bool
	}{
		{
			name: "valid key-value pairs",
			input: map[string]string{
				"username": "john_doe",
				"password": "secret",
			},
			expected:    "password=secret&username=john_doe",
			expectError: false,
		},
		{
			name: "ignore empty key-value pairs",
			input: map[string]string{
				"username": "john_doe",
				"":         "secret",
				"age":      "",
				"location": "USA",
			},
			expected:    "location=USA&username=john_doe",
			expectError: false,
		},
		{
			name: "handle special characters",
			input: map[string]string{
				"name":    "John Doe",
				"address": "123 Main St. #500",
				"email":   "john.doe@example.com",
			},
			expected:    "address=123+Main+St.+%23500&email=john.doe%40example.com&name=John+Doe",
			expectError: false,
		},
		{
			name: "single key-value pair",
			input: map[string]string{
				"username": "john_doe",
			},
			expected:    "username=john_doe",
			expectError: false,
		},
		{
			input:       map[string]string{},
			name:        "empty input",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := yamlparser.EncodeXwwwFormURLBody(tt.input)

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
		input                     map[string]string
		expectedBodyContains      []string
		expectError               bool
	}{
		{
			name: "valid key-value pairs",
			input: map[string]string{
				"username": "john_doe",
				"password": "secret",
			},
			expectError:               false,
			expectedContentTypePrefix: "multipart/form-data; boundary=",
			expectedBodyContains:      []string{"username", "john_doe", "password", "secret"},
		},
		{
			name: "ignore empty key-value pairs",
			input: map[string]string{
				"username": "john_doe",
				"":         "secret",
				"age":      "",
				"location": "USA",
			},
			expectError:               false,
			expectedContentTypePrefix: "multipart/form-data; boundary=",
			expectedBodyContains:      []string{"username", "john_doe", "location", "USA"},
		},
		{
			name:        "empty input",
			input:       map[string]string{},
			expectError: true,
		},
		{
			name: "single key-value pair",
			input: map[string]string{
				"username": "john_doe",
			},
			expectError:               false,
			expectedContentTypePrefix: "multipart/form-data; boundary=",
			expectedBodyContains:      []string{"username", "john_doe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, contentType, err := yamlparser.EncodeFormData(tt.input)

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
		variables    map[string]any
		name         string
		query        string
		expectedBody string
		expectError  bool
	}{
		{
			name:         "valid query and variables",
			query:        "query Hello($name: String!) { hello(name: $name) }",
			variables:    map[string]any{"name": "John"},
			expectError:  false,
			expectedBody: `{"query":"query Hello($name: String!) { hello(name: $name) }","variables":{"name":"John"}}`,
		},
		{
			name:         "no variables (empty map)",
			query:        "query Hello { hello }",
			variables:    map[string]any{},
			expectError:  false,
			expectedBody: `{"query":"query Hello { hello }","variables":{}}`,
		},
		{
			name:         "nil variables",
			query:        "query Hello { hello }",
			variables:    nil,
			expectError:  false,
			expectedBody: `{"query":"query Hello { hello }","variables":{}}`,
		},
		{
			name:        "empty query",
			query:       "",
			variables:   map[string]any{"name": "John"},
			expectError: true,
		},
		{
			name:        "invalid JSON in variables",
			query:       "query Hello { hello }",
			variables:   map[string]any{"invalid": make(chan int)}, // unsupported type
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := yamlparser.EncodeGraphQlBody(tt.query, tt.variables)

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

			actual := string(body)
			if actual != tt.expectedBody {
				t.Errorf("expected %v, got %v", tt.expectedBody, actual)
			}
		})
	}
}
