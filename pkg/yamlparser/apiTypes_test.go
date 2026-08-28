package yamlparser

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
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
			body:     &Body{Graphql: &GraphQl{Variables: map[string]any{"key": "value"}}},
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
					Variables: map[string]any{"key": "value"},
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

// liveWriterGoroutines counts the running multipart writer goroutines by their
// closure frame, so the assertion tracks exactly the goroutine under test and
// not the process-wide total, which unrelated goroutines move either way. The
// match is streamParts.func1, the closure itself, not any helper whose name
// merely contains "streamParts".
func liveWriterGoroutines() int {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	return strings.Count(string(buf[:n]), "yamlparser.streamParts.func1(")
}

// EncodeFormData writes on a goroutine into an unbuffered pipe, so every write
// blocks until the consumer reads. A consumer that stops early (cancelled
// request, failed dry-run) must not strand that goroutine forever.
func TestEncodeFormDataReaderCloseReleasesWriter(t *testing.T) {
	fields := map[string]string{}
	for i := range 200 {
		fields[fmt.Sprintf("field%03d", i)] = strings.Repeat("x", 4096)
	}

	body, _, err := EncodeFormData(fields)
	if err != nil {
		t.Fatalf("EncodeFormData: %v", err)
	}

	// Read one byte so the writer is mid-stream and blocked, then abandon it.
	if _, err := io.ReadFull(body, make([]byte, 1)); err != nil {
		t.Fatalf("reading first byte: %v", err)
	}
	// The goroutine must actually be blocked now, or "gone after close" proves
	// nothing.
	if got := liveWriterGoroutines(); got == 0 {
		t.Fatal("writer goroutine not running while body is mid-stream")
	}

	if err := body.Close(); err != nil {
		t.Fatalf("closing body: %v", err)
	}

	for range 100 {
		if liveWriterGoroutines() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("writer goroutine still running 1s after close")
}

// raw: {{attachFile "x"}} sends the file as the whole body. The type is guessed
// from the extension, but a header the user wrote always wins.
func TestRawAttachFileBodyAndContentType(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "report.pdf")
	payload := []byte("%PDF-1.7 fake")
	if err := os.WriteFile(pdf, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		headers map[string]string
		wantCT  string
	}{
		{"guesses from extension", nil, "application/pdf"},
		{"explicit lowercase wins", map[string]string{"content-type": "application/x-custom"}, "application/x-custom"},
		{"explicit capitalized wins", map[string]string{"Content-Type": "application/x-custom"}, "application/x-custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := &APICallFile{
				Method:  POST,
				URL:     "https://example.com/blobs/1",
				Headers: tt.headers,
				Body:    &Body{Raw: utils.FileRef(pdf)},
			}
			info, err := call.PrepareStruct()
			if err != nil {
				t.Fatalf("PrepareStruct: %v", err)
			}

			var got string
			for k, v := range info.Headers {
				if strings.EqualFold(k, "content-type") {
					got = v
				}
			}
			if got != tt.wantCT {
				t.Errorf("content-type = %q, want %q", got, tt.wantCT)
			}

			streamed, ok := info.Body.(*StreamedBody)
			if !ok {
				t.Fatalf("body is %T, want *StreamedBody", info.Body)
			}
			if streamed.Length != int64(len(payload)) {
				t.Errorf("Length = %d, want %d", streamed.Length, len(payload))
			}
			data, err := io.ReadAll(streamed)
			if err != nil {
				t.Fatalf("reading body: %v", err)
			}
			if !bytes.Equal(data, payload) {
				t.Errorf("body = %q, want %q", data, payload)
			}
			_ = streamed.Close()
		})
	}
}

// A multipart body must advertise exactly one Content-Type, the generated one
// with its boundary, even when the YAML author also wrote a Content-Type. Two
// headers make the server pick the boundary-less one and reject the request.
func TestMultipartContentTypeReplacesUserHeader(t *testing.T) {
	for _, userKey := range []string{"Content-Type", "content-type", "CONTENT-TYPE"} {
		t.Run(userKey, func(t *testing.T) {
			call := &APICallFile{
				Method:  POST,
				URL:     "https://example.com/upload",
				Headers: map[string]string{userKey: "multipart/form-data"},
				Body:    &Body{FormData: map[string]string{"field": "value"}},
			}
			info, err := call.PrepareStruct()
			if err != nil {
				t.Fatalf("PrepareStruct: %v", err)
			}

			var cts []string
			for k, v := range info.Headers {
				if strings.EqualFold(k, "content-type") {
					cts = append(cts, v)
				}
			}
			if len(cts) != 1 {
				t.Fatalf("got %d content-type headers %v, want 1", len(cts), cts)
			}
			if !strings.HasPrefix(cts[0], "multipart/form-data; boundary=") {
				t.Errorf("content-type = %q, want generated boundary type", cts[0])
			}
			if streamed, ok := info.Body.(*StreamedBody); ok {
				_ = streamed.Close()
			}
		})
	}
}

// A CR or LF in a field name or filename must not break out of the
// Content-Disposition line and forge extra headers.
func TestFormDataStripsHeaderInjection(t *testing.T) {
	dir := t.TempDir()
	evil := filepath.Join(dir, "evil.txt")
	if err := os.WriteFile(evil, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{
		"name\r\nX-Injected: 1": utils.FileRef(evil),
	}
	body, ct, err := EncodeFormData(fields)
	if err != nil {
		t.Fatalf("EncodeFormData: %v", err)
	}
	defer func() { _ = body.Close() }()

	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(raw), "X-Injected") && strings.Contains(string(raw), "\r\nX-Injected") {
		t.Errorf("field name CRLF survived into the body:\n%s", raw)
	}
	_ = ct
}

// attachFile only streams a file in formdata or as a whole raw body. Anywhere
// else the marker would reach the server as text, so PrepareStruct/EncodeBody
// must reject it with a clear error.
func TestAttachFileRejectedInUnsupportedContexts(t *testing.T) {
	marker := utils.FileRef("/tmp/whatever.png")

	tests := []struct {
		name string
		call *APICallFile
	}{
		{"header", &APICallFile{
			Method: POST, URL: "https://example.com",
			Headers: map[string]string{"X-Thing": marker},
			Body:    &Body{Raw: "hi"},
		}},
		{"urlparam", &APICallFile{
			Method: GET, URL: "https://example.com",
			URLParams: map[string]string{"q": marker},
		}},
		{"urlencoded", &APICallFile{
			Method: POST, URL: "https://example.com",
			Body: &Body{URLEncodedFormData: map[string]string{"f": marker}},
		}},
		{"graphql query", &APICallFile{
			Method: POST, URL: "https://example.com",
			Body: &Body{Graphql: &GraphQl{Query: "query { " + marker + " }"}},
		}},
		{"graphql variables", &APICallFile{
			Method: POST, URL: "https://example.com",
			Body: &Body{Graphql: &GraphQl{Query: "query {}", Variables: map[string]any{"v": marker}}},
		}},
		{"raw embedded", &APICallFile{
			Method: POST, URL: "https://example.com",
			Body: &Body{Raw: "prefix " + marker},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call.PrepareStruct(); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
