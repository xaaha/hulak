package yamlParser

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestIsValid for HTTPMethodType
func TestIsValid(t *testing.T) {
	// Valid HTTP methods
	validMethods := map[string]HTTPMethodType{
		"GET":     GET,
		"POST":    POST,
		"PUT":     PUT,
		"PATCH":   PATCH,
		"DELETE":  DELETE,
		"HEAD":    HEAD,
		"OPTIONS": OPTIONS,
		"TRACE":   TRACE,
		"CONNECT": CONNECT,
	}

	for name, method := range validMethods {
		if !method.IsValid() {
			t.Errorf("Expected %s to be valid, got invalid", name)
		}
	}

	// Invalid HTTP methods
	invalidMethods := []HTTPMethodType{
		HTTPMethodType("INVALID"),
		HTTPMethodType("FOO"),
		HTTPMethodType(""),
		HTTPMethodType("POSTING"),
		HTTPMethodType("CONNECTS"),
	}

	for _, method := range invalidMethods {
		if method.IsValid() {
			t.Errorf("Expected %s to be invalid, got valid", method)
		}
	}
}

// TestStringConversion for HTTPMethodType
func TestStringConversion(t *testing.T) {
	methodTests := []struct {
		method   HTTPMethodType
		expected string
	}{
		{GET, http.MethodGet},
		{POST, http.MethodPost},
		{PUT, http.MethodPut},
		{PATCH, http.MethodPatch},
		{DELETE, http.MethodDelete},
		{HEAD, http.MethodHead},
		{OPTIONS, http.MethodOptions},
		{TRACE, http.MethodTrace},
		{CONNECT, http.MethodConnect},
	}

	for _, test := range methodTests {
		if string(test.method) != test.expected {
			t.Errorf(
				"Expected string representation of %s to be %s, got %s",
				test.method,
				test.expected,
				string(test.method),
			)
		}
	}
}

// TestMethodSet verifies each HTTPMethodType constant is set correctly
func TestMethodSet(t *testing.T) {
	if GET != HTTPMethodType(http.MethodGet) {
		t.Errorf("Expected GET to be %s, got %s", http.MethodGet, GET)
	}
	if POST != HTTPMethodType(http.MethodPost) {
		t.Errorf("Expected POST to be %s, got %s", http.MethodPost, POST)
	}
	if PUT != HTTPMethodType(http.MethodPut) {
		t.Errorf("Expected PUT to be %s, got %s", http.MethodPut, PUT)
	}
	if PATCH != HTTPMethodType(http.MethodPatch) {
		t.Errorf("Expected PATCH to be %s, got %s", http.MethodPatch, PATCH)
	}
	if DELETE != HTTPMethodType(http.MethodDelete) {
		t.Errorf("Expected DELETE to be %s, got %s", http.MethodDelete, DELETE)
	}
	if HEAD != HTTPMethodType(http.MethodHead) {
		t.Errorf("Expected HEAD to be %s, got %s", http.MethodHead, HEAD)
	}
	if OPTIONS != HTTPMethodType(http.MethodOptions) {
		t.Errorf("Expected OPTIONS to be %s, got %s", http.MethodOptions, OPTIONS)
	}
	if TRACE != HTTPMethodType(http.MethodTrace) {
		t.Errorf("Expected TRACE to be %s, got %s", http.MethodTrace, TRACE)
	}
	if CONNECT != HTTPMethodType(http.MethodConnect) {
		t.Errorf("Expected CONNECT to be %s, got %s", http.MethodConnect, CONNECT)
	}
}

func TestBodyIsValid(t *testing.T) {
	tests := []struct {
		name     string
		body     *Body
		expected bool
	}{
		{
			name:     "nil Body",
			body:     nil,
			expected: true,
		},
		{
			name:     "all fields empty",
			body:     &Body{},
			expected: false,
		},
		{
			name:     "non-empty FormData",
			body:     &Body{FormData: map[string]string{"key": "value"}},
			expected: true,
		},
		{
			name:     "non-empty UrlEncodedFormData",
			body:     &Body{UrlEncodedFormData: map[string]string{"key": "value"}},
			expected: true,
		},
		{
			name:     "non-nil GraphQl with Variables",
			body:     &Body{Graphql: &GraphQl{Variables: map[string]interface{}{"key": "value"}}},
			expected: true,
		},
		{
			name:     "non-nil GraphQl with Query",
			body:     &Body{Graphql: &GraphQl{Query: "query content"}},
			expected: true,
		},
		{
			name:     "non-empty Raw field",
			body:     &Body{Raw: "raw content"},
			expected: true,
		},
		{
			name:     "two non-empty fields (FormData and Raw)",
			body:     &Body{FormData: map[string]string{"key": "value"}, Raw: "raw content"},
			expected: false,
		},
		{
			name: "two non-empty fields (Graphql and FormData)",
			body: &Body{
				Graphql:  &GraphQl{Query: "query content"},
				FormData: map[string]string{"key": "value"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.body.IsValid()
			if result != tt.expected {
				t.Errorf("Test %s failed: expected %v, got %v", tt.name, tt.expected, result)
			}
		})
	}
}

func TestEncodeBody(t *testing.T) {
	tests := []struct {
		name        string
		body        *Body
		expectError bool
		expectedCT  string
		expectedStr string
	}{
		{
			name:        "nil Body",
			body:        nil,
			expectError: false,
			expectedCT:  "",
			expectedStr: "",
		},
		{
			name: "GraphQL Body with Query and Variables",
			body: &Body{
				Graphql: &GraphQl{
					Query:     "query content",
					Variables: map[string]interface{}{"key": "value"},
				},
			},
			expectError: false,
			expectedCT:  "",
			expectedStr: `{"query":"query content","variables":{"key":"value"}}`,
		},
		// {
		// 	name: "Multipart Form Data",
		// 	body: &Body{
		// 		FormData: map[string]string{"key": "value"},
		// 	},
		// 	expectError: false,
		// 	expectedCT:  "multipart/form-data",
		// 	expectedStr: "key=value",
		// },
		{
			name: "URL Encoded Form Data",
			body: &Body{
				UrlEncodedFormData: map[string]string{"key": "value"},
			},
			expectError: false,
			expectedCT:  "application/x-www-form-urlencoded",
			expectedStr: "key=value",
		},
		{
			name: "Raw Body Content",
			body: &Body{
				Raw: "raw content",
			},
			expectError: false,
			expectedCT:  "",
			expectedStr: "raw content",
		},
		{
			name:        "Empty Body Struct",
			body:        &Body{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType, err := tt.body.EncodeBody()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check content type
			if contentType != tt.expectedCT {
				t.Errorf("Expected content type %s, got %s", tt.expectedCT, contentType)
			}

			// Check body content if it exists
			if body != nil {
				bodyBytes, _ := io.ReadAll(body)
				bodyStr := strings.TrimSpace(string(bodyBytes))
				if bodyStr != tt.expectedStr {
					t.Errorf("Expected body content %q, got %q", tt.expectedStr, bodyStr)
				}
			}
		})
	}
}
