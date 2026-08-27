package yamlparser

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"
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
			body:     &Body{URLEncodedFormData: map[string]string{"key": "value"}},
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
				URLEncodedFormData: map[string]string{"key": "value"},
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

func TestProcessVariable_PreservesUintFamily(t *testing.T) {
	testCases := []struct {
		name string
		in   any
	}{
		{"uint", uint(3939)},
		{"uint8", uint8(42)},
		{"uint16", uint16(8000)},
		{"uint32", uint32(70000)},
		{"uint64", uint64(3939)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := processVariable(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.in {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tc.in, tc.in)
			}
		})
	}
}

// EncodeFormData writes on a goroutine into an unbuffered pipe, so every write
// blocks until the consumer reads. A consumer that stops early (cancelled
// request, failed dry-run) must not strand that goroutine forever.
func TestEncodeFormDataReaderCloseReleasesWriter(t *testing.T) {
	fields := map[string]string{}
	for i := range 200 {
		fields[fmt.Sprintf("field%03d", i)] = strings.Repeat("x", 4096)
	}

	before := runtime.NumGoroutine()

	body, _, err := EncodeFormData(fields)
	if err != nil {
		t.Fatalf("EncodeFormData: %v", err)
	}

	// Read one byte so the writer is mid-stream and blocked, then abandon it.
	if _, err := io.ReadFull(body, make([]byte, 1)); err != nil {
		t.Fatalf("reading first byte: %v", err)
	}
	// Closeability is guaranteed by the type now; what still needs proving is
	// that closing actually releases the blocked writer.
	if err := body.Close(); err != nil {
		t.Fatalf("closing body: %v", err)
	}

	for range 100 {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("writer goroutine still running 1s after close: %d before, %d now",
		before, runtime.NumGoroutine())
}
