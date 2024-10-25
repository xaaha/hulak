package apicalls

import (
	"io"
	"testing"
)

func TestFullUrl(t *testing.T) {
	tests := []struct {
		name     string
		baseUrl  string
		expected string
		params   []KeyValuePair
	}{
		// Test with no parameters
		{
			name:     "No parameters",
			baseUrl:  "https://api.example.com/resource",
			params:   []KeyValuePair{},
			expected: "https://api.example.com/resource",
		},
		// Test with a single parameter
		{
			name:    "Single parameter",
			baseUrl: "https://api.example.com/resource",
			params: []KeyValuePair{
				{"search", "golang"},
			},
			expected: "https://api.example.com/resource?search=golang",
		},
		// Test with multiple parameters
		{
			name:    "Multiple parameters",
			baseUrl: "https://api.example.com/resource",
			params: []KeyValuePair{
				{"search", "golang"},
				{"limit", "10"},
				{"sort", "desc"},
			},
			expected: "https://api.example.com/resource?limit=10&search=golang&sort=desc",
		},
		// Test with special characters in parameters
		{
			name:    "Special characters in parameters",
			baseUrl: "https://api.example.com/resource",
			params: []KeyValuePair{
				{"search", "go programming"},
				{"filter", "name&value"},
			},
			expected: "https://api.example.com/resource?filter=name%26value&search=go+programming",
		},
		// Test with empty parameter values
		{
			name:    "Empty parameter values",
			baseUrl: "https://api.example.com/resource",
			params: []KeyValuePair{
				{"search", ""},
				{"limit", "10"},
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
		input       []KeyValuePair
		expectError bool
	}{
		{
			name: "valid key-value pairs",
			input: []KeyValuePair{
				{Key: "username", Value: "john_doe"},
				{Key: "password", Value: "secret"},
			},
			expected:    "password=secret&username=john_doe",
			expectError: false,
		},
		{
			name: "ignore empty key-value pairs",
			input: []KeyValuePair{
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
			input: []KeyValuePair{
				{Key: "name", Value: "John Doe"},
				{Key: "address", Value: "123 Main St. #500"},
				{Key: "email", Value: "john.doe@example.com"},
			},
			expected:    "address=123+Main+St.+%23500&email=john.doe%40example.com&name=John+Doe",
			expectError: false,
		},
		{
			name: "single key-value pair",
			input: []KeyValuePair{
				{Key: "username", Value: "john_doe"},
			},
			expected:    "username=john_doe",
			expectError: false,
		},
		{
			name:        "empty input",
			input:       []KeyValuePair{},
			expectError: true,
		},
		{
			name: "key-value pairs with same key",
			input: []KeyValuePair{
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
