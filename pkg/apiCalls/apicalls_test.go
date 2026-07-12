package apicalls

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			bytesOut, err := SerializeAndSaveResp(resp, inputPath)
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
	bytesOut, err := SerializeAndSaveResp(resp, inputPath)
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
	bytesOut, err := SerializeAndSaveResp(resp, inputPath)
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
