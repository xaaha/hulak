package apicalls

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func TestStandardCallWithClient_Success(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		method         string
		expectedStatus int
	}{
		{
			name:           "successful GET request with 200",
			statusCode:     200,
			responseBody:   `{"message": "success"}`,
			method:         "GET",
			expectedStatus: 200,
		},
		{
			name:           "GET request with 404",
			statusCode:     404,
			responseBody:   `{"error": "not found"}`,
			method:         "GET",
			expectedStatus: 404,
		},
		{
			name:           "GET request with 500",
			statusCode:     500,
			responseBody:   `{"error": "internal server error"}`,
			method:         "GET",
			expectedStatus: 500,
		},
		{
			name:           "successful POST request",
			statusCode:     201,
			responseBody:   `{"id": 123, "created": true}`,
			method:         "POST",
			expectedStatus: 201,
		},
		{
			name:           "successful DELETE request",
			statusCode:     204,
			responseBody:   "",
			method:         "DELETE",
			expectedStatus: 204,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(_ *http.Request) (*http.Response, error) {
					return NewMockResponse(tc.statusCode, tc.responseBody), nil
				},
			}

			apiInfo := yamlparser.APIInfo{
				Method:  tc.method,
				URL:     "http://example.com/api/test",
				Headers: map[string]string{},
			}

			resp, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Response.StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, resp.Response.StatusCode)
			}
		})
	}
}

func TestStandardCallWithClient_Headers(t *testing.T) {
	var capturedHeaders http.Header

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedHeaders = req.Header
			return NewMockResponse(200, `{"success": true}`), nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method: "GET",
		URL:    "http://example.com/api/test",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
			"X-Custom":      "custom-value",
		},
	}

	_, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedHeaders := map[string]string{
		"Authorization": "Bearer token123",
		"Content-Type":  "application/json",
		"X-Custom":      "custom-value",
	}

	for key, expected := range expectedHeaders {
		if got := capturedHeaders.Get(key); got != expected {
			t.Errorf("header %s: expected %q, got %q", key, expected, got)
		}
	}
}

func TestStandardCallWithClient_PostWithBody(t *testing.T) {
	var capturedBody []byte
	var capturedMethod string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedMethod = req.Method
			body, _ := io.ReadAll(req.Body)
			capturedBody = body
			return NewMockResponse(201, `{"id": 456}`), nil
		},
	}

	requestBody := `{"name": "test", "value": 42}`
	apiInfo := yamlparser.APIInfo{
		Method:  "POST",
		URL:     "http://example.com/api/items",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    bytes.NewReader([]byte(requestBody)),
	}

	resp, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != "POST" {
		t.Errorf("expected method POST, got %s", capturedMethod)
	}

	if string(capturedBody) != requestBody {
		t.Errorf("expected body %q, got %q", requestBody, string(capturedBody))
	}

	if resp.Response.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.Response.StatusCode)
	}
}

func TestStandardCallWithClient_NetworkError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, ErrMockNetwork
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	_, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != ErrMockNetwork {
		t.Errorf("expected ErrMockNetwork, got %v", err)
	}
}

// TestStandardCallWithClient_BodyReadError simulates a transport error mid-stream
// (e.g., TCP reset, connection drop while reading the response body). Regression
// for #204 — previously this triggered log.Fatalf and killed the entire process,
// abandoning sibling requests in the worker pool.
func TestStandardCallWithClient_BodyReadError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       &erroringReadCloser{err: ErrMockNetwork},
				Header:     make(http.Header),
			}, nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	_, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err == nil {
		t.Fatal("expected error from body read failure, got nil")
	}
	if !strings.Contains(err.Error(), "reading response body") {
		t.Errorf("expected error to wrap with 'reading response body', got %v", err)
	}
}

// erroringReadCloser returns err on every Read, simulating a broken stream.
type erroringReadCloser struct {
	err error
}

func (e *erroringReadCloser) Read(_ []byte) (int, error) { return 0, e.err }
func (e *erroringReadCloser) Close() error               { return nil }

func TestStandardCallWithClient_URLParams(t *testing.T) {
	var capturedURL string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedURL = req.URL.String()
			return NewMockResponse(200, `{}`), nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/search",
		Headers: map[string]string{},
		URLParams: map[string]string{
			"q":     "test query",
			"limit": "10",
		},
	}

	_, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that URL params are included
	if !strings.Contains(capturedURL, "q=") {
		t.Errorf("expected URL to contain 'q=' param, got %s", capturedURL)
	}
	if !strings.Contains(capturedURL, "limit=10") {
		t.Errorf("expected URL to contain 'limit=10', got %s", capturedURL)
	}
}

func TestStandardCallWithClient_DebugMode(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			resp := NewMockResponseWithHeaders(200, `{"data": "test"}`, map[string]string{
				"X-Server": "test-server",
			})
			return resp, nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}

	// Test with debug=true
	resp, err := StandardCallWithClient(context.Background(), apiInfo, true, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In debug mode, Request info should be populated
	if resp.Request == nil {
		t.Error("expected Request info in debug mode, got nil")
	}

	if resp.Request != nil {
		if resp.Request.Method != "GET" {
			t.Errorf("expected method GET in request info, got %s", resp.Request.Method)
		}
	}

	// Response headers should be populated in debug mode
	if resp.Response.Headers == nil {
		t.Error("expected Response headers in debug mode, got nil")
	}
}

func TestStandardCallWithClient_NonDebugMode(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{"data": "test"}`), nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	// Test with debug=false
	resp, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In non-debug mode, Request info should be nil
	if resp.Request != nil {
		t.Error("expected Request info to be nil in non-debug mode")
	}

	// Response body should still be present
	if resp.Response == nil || resp.Response.Body == nil {
		t.Error("expected Response body in non-debug mode")
	}
}

func TestStandardCallWithClient_JSONResponse(t *testing.T) {
	responseJSON := `{"id": 123, "name": "test", "nested": {"key": "value"}}`

	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, responseJSON), nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	resp, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Response body should be parsed as JSON
	bodyMap, ok := resp.Response.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected body to be map[string]any, got %T", resp.Response.Body)
	}

	if bodyMap["id"] != float64(123) {
		t.Errorf("expected id=123, got %v", bodyMap["id"])
	}

	if bodyMap["name"] != "test" {
		t.Errorf("expected name='test', got %v", bodyMap["name"])
	}
}

func TestStandardCallWithClient_PlainTextResponse(t *testing.T) {
	plainText := "This is plain text, not JSON"

	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, plainText), nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/text",
		Headers: map[string]string{},
	}

	resp, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Non-JSON response should be stored as string
	bodyStr, ok := resp.Response.Body.(string)
	if !ok {
		t.Fatalf("expected body to be string, got %T", resp.Response.Body)
	}

	if bodyStr != plainText {
		t.Errorf("expected %q, got %q", plainText, bodyStr)
	}
}

func TestStandardCallWithClient_NilHeaders(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{}`), nil
		},
	}

	// Test with nil Headers - should not panic
	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: nil, // explicitly nil
	}

	_, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error with nil headers: %v", err)
	}
}

func TestStandardCallWithClient_Duration(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{}`), nil
		},
	}

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	resp, err := StandardCallWithClient(context.Background(), apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duration should be set
	if resp.Duration == "" {
		t.Error("expected Duration to be set, got empty string")
	}

	// Duration should end with "ms"
	if !strings.HasSuffix(resp.Duration, "ms") {
		t.Errorf("expected Duration to end with 'ms', got %s", resp.Duration)
	}
}

func TestStandardCall_UsesDefaultClient(t *testing.T) {
	// Create a test server
	server := NewMockServer(200, `{"test": "data"}`)
	defer server.Close()

	apiInfo := yamlparser.APIInfo{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
	}

	resp, err := StandardCall(context.Background(), apiInfo, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.Response.StatusCode)
	}
}

func TestMockServer(t *testing.T) {
	// Test the mock server helper itself
	server := NewMockServer(201, `{"created": true}`)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to reach mock server: %v", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Errorf("error closing the body: %v", err)
		}
	}()

	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]bool
	_ = json.Unmarshal(body, &result)

	if !result["created"] {
		t.Error("expected created=true in response")
	}
}

func TestMockServerWithHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"method": "POST"}`))
		} else {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"method": "GET"}`))
		}
	}

	server := NewMockServerWithHandler(handler)
	defer server.Close()

	// Test GET
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Errorf("error closing the body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("GET: expected 200, got %d", resp.StatusCode)
	}

	// Test POST
	resp2, err := http.Post(server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	defer func() {
		err := resp2.Body.Close()
		if err != nil {
			t.Errorf("error closing the body: %v", err)
		}
	}()

	if resp2.StatusCode != 201 {
		t.Errorf("POST: expected 201, got %d", resp2.StatusCode)
	}
}

func TestIsDebug(t *testing.T) {
	tests := []struct {
		name string
		resp CustomResponse
		want bool
	}{
		{"nil Request is default mode", CustomResponse{}, false},
		{"set Request is debug mode", CustomResponse{Request: &RequestInfo{Method: "GET"}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.resp.isDebug(); got != tc.want {
				t.Errorf("isDebug() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDefaultBodyForOutput(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "empty input returned as-is",
			in:   []byte{},
			want: []byte{},
		},
		{
			name: "nil input returned as-is",
			in:   nil,
			want: nil,
		},
		{
			name: "valid JSON pretty-printed",
			in:   []byte(`{"a":1,"b":{"c":2}}`),
			want: []byte("{\n  \"a\": 1,\n  \"b\": {\n    \"c\": 2\n  }\n}"),
		},
		{
			name: "already-formatted JSON re-indented to two spaces",
			in:   []byte("{\n    \"a\": 1\n}"),
			want: []byte("{\n  \"a\": 1\n}"),
		},
		{
			name: "HTML preserved byte-perfect",
			in:   []byte("<!DOCTYPE html><html><body>x</body></html>"),
			want: []byte("<!DOCTYPE html><html><body>x</body></html>"),
		},
		{
			name: "XML preserved byte-perfect",
			in:   []byte("<root><k>v</k></root>"),
			want: []byte("<root><k>v</k></root>"),
		},
		{
			name: "plain text preserved byte-perfect",
			in:   []byte("hello world\nline two"),
			want: []byte("hello world\nline two"),
		},
		{
			name: "malformed JSON returned as-is (fallback path)",
			in:   []byte(`{"a": 1,`),
			want: []byte(`{"a": 1,`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultBodyForOutput(tc.in)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("defaultBodyForOutput()\n  got:  %q\n  want: %q", got, tc.want)
			}
		})
	}
}

// TestSerializeAndSaveResp_Default verifies that default mode writes only
// the response body — no request/duration/http_info wrapper. JSON bodies
// are pretty-printed, non-JSON content is preserved byte-perfect.
func TestSerializeAndSaveResp_Default(t *testing.T) {
	tests := []struct {
		name        string
		rawBody     []byte
		contentType string
		wantOnDisk  string
		wantExt     string
	}{
		{
			name:        "JSON body is pretty-printed",
			rawBody:     []byte(`{"data":{"id":42}}`),
			contentType: "application/json",
			wantOnDisk:  "{\n  \"data\": {\n    \"id\": 42\n  }\n}",
			wantExt:     ".json",
		},
		{
			name:        "HTML body kept byte-perfect",
			rawBody:     []byte("<!DOCTYPE html><html><body>ok</body></html>"),
			contentType: "text/html",
			wantOnDisk:  "<!DOCTYPE html><html><body>ok</body></html>",
			wantExt:     ".html",
		},
		{
			name:        "XML body kept byte-perfect",
			rawBody:     []byte("<root><k>v</k></root>"),
			contentType: "application/xml",
			wantOnDisk:  "<root><k>v</k></root>",
			wantExt:     ".xml",
		},
		{
			name:        "plain text body kept byte-perfect",
			rawBody:     []byte("plain text response"),
			contentType: "text/plain",
			wantOnDisk:  "plain text response",
			wantExt:     ".txt",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			// Path the runner passes is a yaml-like input; evalAndWriteRes
			// strips the trailing extension and appends the resolved one.
			inputPath := dir + "/req.hk.yaml"
			resp := &CustomResponse{
				Response:    &ResponseInfo{StatusCode: 200, Status: "200 OK"},
				Duration:    "10.00ms",
				contentType: tc.contentType,
				rawBody:     tc.rawBody,
			}
			bytesOut, err := SerializeAndSaveResp(resp, inputPath, "")
			if err != nil {
				t.Fatalf("SerializeAndSaveResp returned error: %v", err)
			}
			if string(bytesOut) != tc.wantOnDisk {
				t.Errorf("returned bytes mismatch\n  got:  %q\n  want: %q", bytesOut, tc.wantOnDisk)
			}
			// The wrapper keys must NOT appear in default mode output.
			for _, leak := range []string{`"response"`, `"duration"`, `"status_code"`, `"http_info"`, `"request"`} {
				if strings.Contains(string(bytesOut), leak) {
					t.Errorf("default-mode bytes contain wrapper key %s: %s", leak, bytesOut)
				}
			}
			diskPath := dir + "/req.hk_response" + tc.wantExt
			onDisk, readErr := readFile(t, diskPath)
			if readErr != nil {
				t.Fatalf("reading saved file %s: %v", diskPath, readErr)
			}
			if onDisk != tc.wantOnDisk {
				t.Errorf("on-disk content mismatch\n  got:  %q\n  want: %q", onDisk, tc.wantOnDisk)
			}
		})
	}
}

// TestSerializeAndSaveResp_EmptyBody verifies that an empty response body
// (HTTP 204 No Content and similar) does not error and writes no file.
func TestSerializeAndSaveResp_EmptyBody(t *testing.T) {
	dir := t.TempDir()
	inputPath := dir + "/req.hk.yaml"
	resp := &CustomResponse{
		Response:    &ResponseInfo{StatusCode: 204, Status: "204 No Content"},
		Duration:    "1.00ms",
		contentType: "",
		rawBody:     []byte{},
	}
	bytesOut, err := SerializeAndSaveResp(resp, inputPath, "")
	if err != nil {
		t.Fatalf("empty body should not error, got: %v", err)
	}
	if len(bytesOut) != 0 {
		t.Errorf("expected zero-length bytes, got %d: %q", len(bytesOut), bytesOut)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading temp dir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected no files written for empty body, found: %v", names)
	}
}

// TestSerializeAndSaveResp_Debug verifies --debug mode preserves the full
// wrapped CustomResponse shape (request, response, http_info, duration).
func TestSerializeAndSaveResp_Debug(t *testing.T) {
	dir := t.TempDir()
	inputPath := dir + "/req.hk.yaml"
	resp := &CustomResponse{
		Request: &RequestInfo{
			Method: "POST",
			URL:    "https://example.com/api",
		},
		Response: &ResponseInfo{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       map[string]any{"ok": true},
		},
		HTTPInfo:    &HTTPInfo{Protocol: "HTTP/1.1"},
		Duration:    "12.34ms",
		contentType: "application/json",
		rawBody:     []byte(`{"ok":true}`),
	}
	bytesOut, err := SerializeAndSaveResp(resp, inputPath, "")
	if err != nil {
		t.Fatalf("SerializeAndSaveResp returned error: %v", err)
	}
	// Wrapped output must include the metadata keys.
	for _, want := range []string{`"request"`, `"response"`, `"http_info"`, `"duration"`} {
		if !strings.Contains(string(bytesOut), want) {
			t.Errorf("debug bytes missing wrapper key %s\n  got: %s", want, bytesOut)
		}
	}
	// Parse to confirm the saved JSON is structurally well-formed.
	var roundTrip map[string]any
	if err := json.Unmarshal(bytesOut, &roundTrip); err != nil {
		t.Fatalf("debug bytes are not valid JSON: %v", err)
	}
	if _, ok := roundTrip["request"]; !ok {
		t.Error("debug JSON missing request key")
	}
}

// readFile is a small test helper.
func readFile(t *testing.T, path string) (string, error) {
	t.Helper()
	b, err := os.ReadFile(path)
	return string(b), err
}

// TestSendAndSaveAPIRequest_NoSave verifies NoSave returns the response bytes
// without writing the {name}_response.json file, while the default still
// writes it. Runs against a local httptest server via the real HTTP path.
func TestSendAndSaveAPIRequest_NoSave(t *testing.T) {
	server := NewMockServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()

	tests := []struct {
		name     string
		noSave   bool
		wantFile bool
	}{
		{"NoSave skips response file", true, false},
		{"default writes response file", false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "req.hk.yaml")
			doc := "---\nkind: API\nmethod: GET\nurl: " + server.URL + "\n"
			if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
				t.Fatal(err)
			}

			respBytes, status, err := SendAndSaveAPIRequest(context.Background(), RequestOptions{
				Secrets: map[string]any{},
				Path:    path,
				NoSave:  tc.noSave,
			})
			if err != nil {
				t.Fatalf("SendAndSaveAPIRequest: %v", err)
			}
			if status != "200 OK" {
				t.Errorf("status = %q, want 200 OK", status)
			}
			if !strings.Contains(string(respBytes), `"ok": true`) {
				t.Errorf("response bytes should contain the (pretty-printed) body, got:\n%s", respBytes)
			}

			matches, err := filepath.Glob(filepath.Join(dir, "*_response.*"))
			if err != nil {
				t.Fatal(err)
			}
			if gotFile := len(matches) > 0; gotFile != tc.wantFile {
				t.Errorf("response file written = %v, want %v (matches: %v)", gotFile, tc.wantFile, matches)
			}
		})
	}
}

// TestSendAndSaveAPIRequest_OutPath verifies OutPath redirects the response to
// an exact path (creating parent dirs) and suppresses the default
// <name>_response.* file next to the request. Runs against a local httptest
// server via the real HTTP path.
func TestSendAndSaveAPIRequest_OutPath(t *testing.T) {
	server := NewMockServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "req.hk.yaml")
	doc := "---\nkind: API\nmethod: GET\nurl: " + server.URL + "\n"
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "custom", "out.json")
	respBytes, status, err := SendAndSaveAPIRequest(context.Background(), RequestOptions{
		Secrets: map[string]any{},
		Path:    path,
		OutPath: outPath,
	})
	if err != nil {
		t.Fatalf("SendAndSaveAPIRequest: %v", err)
	}
	if status != "200 OK" {
		t.Errorf("status = %q, want 200 OK", status)
	}
	if !strings.Contains(string(respBytes), `"ok": true`) {
		t.Errorf("response bytes should contain the (pretty-printed) body, got:\n%s", respBytes)
	}

	// The response must land at exactly outPath (nested dir created for us).
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("expected response at %s, stat error: %v", outPath, err)
	}

	// And the default <name>_response.* next to the request must NOT exist.
	defaults, err := filepath.Glob(filepath.Join(dir, "*_response.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(defaults) > 0 {
		t.Errorf("default response file should not be written when -o is set, found: %v", defaults)
	}
}

// newUploadEcho reports back what the server actually received on the wire.
type wireRecord struct {
	contentLength    int64
	transferEncoding []string
	path             string
	fields           map[string]string
	fileNames        map[string]string
	fileTypes        map[string]string
	fileBytes        map[string]int
}

func recordUpload(t *testing.T, r *http.Request) wireRecord {
	t.Helper()
	rec := wireRecord{
		contentLength:    r.ContentLength,
		transferEncoding: r.TransferEncoding,
		path:             r.URL.Path,
		fields:           map[string]string{},
		fileNames:        map[string]string{},
		fileTypes:        map[string]string{},
		fileBytes:        map[string]int{},
	}
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "multipart/") {
		return rec
	}
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return rec
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		data, _ := io.ReadAll(part)
		if part.FileName() != "" {
			rec.fileNames[part.FormName()] = part.FileName()
			rec.fileTypes[part.FormName()] = part.Header.Get("Content-Type")
			rec.fileBytes[part.FormName()] = len(data)
			continue
		}
		rec.fields[part.FormName()] = string(data)
		_ = part.Close()
	}
	return rec
}

func attachFixture(t *testing.T, name string, size int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, bytes.Repeat([]byte("A"), size), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// An attached file must go out with a real Content-Length. Without one the
// request degrades to chunked, which S3 and many upload endpoints reject.
func TestStandardCall_AttachedFileSendsContentLength(t *testing.T) {
	got := make(chan wireRecord, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	png := attachFixture(t, "logo.png", 40000)
	fields := map[string]string{"caption": "hi", "avatar": utils.FileRef(png)}

	body, contentType, err := yamlparser.EncodeFormData(fields)
	if err != nil {
		t.Fatalf("EncodeFormData: %v", err)
	}

	if _, err = StandardCallWithClient(context.Background(), yamlparser.APIInfo{
		Method: "POST", URL: server.URL,
		Headers: map[string]string{"content-type": contentType},
		Body:    body,
	}, false, server.Client()); err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}

	rec := <-got
	if len(rec.transferEncoding) != 0 {
		t.Errorf("sent with Transfer-Encoding %v, want a plain Content-Length", rec.transferEncoding)
	}
	if rec.contentLength != body.Length {
		t.Errorf("server saw Content-Length %d, encoder predicted %d", rec.contentLength, body.Length)
	}
	if rec.fileBytes["avatar"] != 40000 {
		t.Errorf("server got %d file bytes, want 40000", rec.fileBytes["avatar"])
	}
	if rec.fileNames["avatar"] != "logo.png" {
		t.Errorf("filename = %q, want logo.png", rec.fileNames["avatar"])
	}
	if rec.fileTypes["avatar"] != "image/png" {
		t.Errorf("part Content-Type = %q, want image/png", rec.fileTypes["avatar"])
	}
	if rec.fields["caption"] != "hi" {
		t.Errorf("text field = %q, want hi", rec.fields["caption"])
	}
}

// A 307 preserves method and body only when GetBody can rebuild it. Without it
// http.Client stops following and returns the redirect as if it were the answer.
func TestStandardCall_AttachedFileReplaysAcross307(t *testing.T) {
	got := make(chan wireRecord, 2)
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		http.Redirect(w, r, "/final", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	png := attachFixture(t, "logo.png", 2048)
	call := &yamlparser.APICallFile{
		Method: "POST",
		URL:    yamlparser.URL(server.URL + "/start"),
		Body:   &yamlparser.Body{FormData: map[string]string{"avatar": utils.FileRef(png)}},
	}
	info, err := call.PrepareStruct()
	if err != nil {
		t.Fatalf("PrepareStruct: %v", err)
	}
	if sb, ok := info.Body.(*yamlparser.StreamedBody); !ok || sb.GetBody == nil {
		t.Fatal("body is not a StreamedBody with GetBody, so a redirect cannot replay it")
	}

	resp, err := StandardCallWithClient(context.Background(), info, false, server.Client())
	if err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}
	if resp.Response == nil || resp.Response.StatusCode != http.StatusOK {
		t.Fatalf("final status = %+v, want 200; the redirect was not followed", resp.Response)
	}

	close(got)
	var final *wireRecord
	for rec := range got {
		if rec.path == "/final" {
			r := rec
			final = &r
		}
	}
	if final == nil {
		t.Fatal("redirect target never received the request")
	}
	if final.fileBytes["avatar"] != 2048 {
		t.Errorf("replayed body carried %d file bytes, want 2048", final.fileBytes["avatar"])
	}
}

// Bodies with no attached file must keep the Content-Length net/http derives
// from an in-memory reader. Leaving APIInfo.ContentLength at its zero value
// would read as "unknown" and silently downgrade them to chunked.
func TestStandardCall_InMemoryBodiesKeepContentLength(t *testing.T) {
	got := make(chan wireRecord, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	raw := "grant_type=authorization_code&code=abc"
	if _, err := StandardCallWithClient(context.Background(), yamlparser.APIInfo{
		Method: "POST", URL: server.URL,
		Body: strings.NewReader(raw),
	}, false, server.Client()); err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}

	rec := <-got
	if len(rec.transferEncoding) != 0 {
		t.Errorf("sent chunked %v; an in-memory body should carry a length", rec.transferEncoding)
	}
	if rec.contentLength != int64(len(raw)) {
		t.Errorf("Content-Length = %d, want %d", rec.contentLength, len(raw))
	}
}

// An empty attachment must still send Content-Length: 0, not chunked. A
// zero-length non-nil body otherwise makes net/http fall back to chunked.
func TestStandardCall_EmptyRawAttachmentSendsContentLengthZero(t *testing.T) {
	got := make(chan wireRecord, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	empty := attachFixture(t, "empty.bin", 0)
	call := &yamlparser.APICallFile{
		Method: "PUT",
		URL:    yamlparser.URL(server.URL),
		Body:   &yamlparser.Body{Raw: utils.FileRef(empty)},
	}
	info, err := call.PrepareStruct()
	if err != nil {
		t.Fatalf("PrepareStruct: %v", err)
	}
	if _, err := StandardCallWithClient(context.Background(), info, false, server.Client()); err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}

	rec := <-got
	if len(rec.transferEncoding) != 0 {
		t.Errorf("sent with Transfer-Encoding %v, want Content-Length: 0", rec.transferEncoding)
	}
	if rec.contentLength != 0 {
		t.Errorf("Content-Length = %d, want 0", rec.contentLength)
	}
}

// A file that grows between size measurement and streaming must be truncated to
// the declared Content-Length, never sent longer than what was advertised.
func TestStandardCall_GrownFileTruncatedToContentLength(t *testing.T) {
	got := make(chan wireRecord, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	png := attachFixture(t, "logo.png", 1000)
	fields := map[string]string{"avatar": utils.FileRef(png)}
	body, contentType, err := yamlparser.EncodeFormData(fields)
	if err != nil {
		t.Fatalf("EncodeFormData: %v", err)
	}

	// Grow the file after the encoder measured it.
	if err := os.WriteFile(png, bytes.Repeat([]byte("A"), 5000), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := StandardCallWithClient(context.Background(), yamlparser.APIInfo{
		Method: "POST", URL: server.URL,
		Headers: map[string]string{"content-type": contentType},
		Body:    body,
	}, false, server.Client()); err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}

	rec := <-got
	if rec.fileBytes["avatar"] != 1000 {
		t.Errorf("server got %d file bytes, want 1000 (declared length)", rec.fileBytes["avatar"])
	}
}

// openFDCount returns the number of open descriptors for this process, or skips
// if the platform exposes no fd directory.
func openFDCount(t *testing.T) int {
	t.Helper()
	// Readdirnames, not ReadDir: on macOS /dev/fd, lstat-ing each entry fails
	// on the directory's own transient handle, so only name listing is reliable.
	for _, dir := range []string{"/proc/self/fd", "/dev/fd"} {
		f, err := os.Open(dir)
		if err != nil {
			continue
		}
		names, err := f.Readdirnames(-1)
		_ = f.Close()
		if err == nil {
			return len(names)
		}
	}
	t.Skip("no fd directory on this platform")
	return 0
}

// A raw file upload under --debug must not leak the file descriptor. The debug
// path buffers the body to print it, then must close the streamed source.
func TestStandardCall_DebugRawUploadDoesNotLeakFD(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "blob.bin")
	if err := os.WriteFile(bin, []byte("payload bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	build := func() yamlparser.APIInfo {
		call := &yamlparser.APICallFile{
			Method: "PUT",
			URL:    yamlparser.URL(server.URL),
			Body:   &yamlparser.Body{Raw: utils.FileRef(bin)},
		}
		info, err := call.PrepareStruct()
		if err != nil {
			t.Fatal(err)
		}
		return info
	}

	for range 5 {
		if _, err := StandardCallWithClient(context.Background(), build(), true, server.Client()); err != nil {
			t.Fatalf("warmup call: %v", err)
		}
	}
	before := openFDCount(t)
	for range 30 {
		if _, err := StandardCallWithClient(context.Background(), build(), true, server.Client()); err != nil {
			t.Fatalf("call: %v", err)
		}
	}
	after := openFDCount(t)
	if after-before > 10 {
		t.Errorf("leaked ~%d file descriptors across 30 debug raw uploads", after-before)
	}
}

// On a 307 replay, the body must reproduce the ORIGINAL Content-Length even if
// the file grew on disk between the first send and the replay. Without pinning,
// the replayed body would exceed the advertised length and the transport aborts.
func TestStandardCall_ReplayPinsOriginalSizeAcross307(t *testing.T) {
	png := attachFixture(t, "logo.png", 2048)

	got := make(chan wireRecord, 2)
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		// Grow the file before the client rebuilds the body via GetBody.
		if err := os.WriteFile(png, bytes.Repeat([]byte("A"), 9000), 0o600); err != nil {
			t.Error(err)
		}
		http.Redirect(w, r, "/final", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		got <- recordUpload(t, r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	call := &yamlparser.APICallFile{
		Method: "POST",
		URL:    yamlparser.URL(server.URL + "/start"),
		Body:   &yamlparser.Body{FormData: map[string]string{"avatar": utils.FileRef(png)}},
	}
	info, err := call.PrepareStruct()
	if err != nil {
		t.Fatalf("PrepareStruct: %v", err)
	}
	resp, err := StandardCallWithClient(context.Background(), info, false, server.Client())
	if err != nil {
		t.Fatalf("StandardCallWithClient: %v", err)
	}
	if resp.Response == nil || resp.Response.StatusCode != http.StatusOK {
		t.Fatalf("final status = %+v, want 200", resp.Response)
	}

	close(got)
	var final *wireRecord
	for rec := range got {
		if rec.path == "/final" {
			r := rec
			final = &r
		}
	}
	if final == nil {
		t.Fatal("redirect target never received the request")
	}
	if final.fileBytes["avatar"] != 2048 {
		t.Errorf("replay carried %d file bytes, want 2048 (original pinned size)", final.fileBytes["avatar"])
	}
}
