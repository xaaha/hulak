package apicalls

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return NewMockResponse(tc.statusCode, tc.responseBody), nil
				},
			}

			apiInfo := yamlparser.ApiInfo{
				Method:  tc.method,
				Url:     "http://example.com/api/test",
				Headers: map[string]string{},
			}

			resp, err := StandardCallWithClient(apiInfo, false, mockClient)
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

	apiInfo := yamlparser.ApiInfo{
		Method: "GET",
		Url:    "http://example.com/api/test",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
			"X-Custom":      "custom-value",
		},
	}

	_, err := StandardCallWithClient(apiInfo, false, mockClient)
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
	apiInfo := yamlparser.ApiInfo{
		Method:  "POST",
		Url:     "http://example.com/api/items",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    bytes.NewReader([]byte(requestBody)),
	}

	resp, err := StandardCallWithClient(apiInfo, false, mockClient)
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
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, ErrMockNetwork
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	_, err := StandardCallWithClient(apiInfo, false, mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != ErrMockNetwork {
		t.Errorf("expected ErrMockNetwork, got %v", err)
	}
}

func TestStandardCallWithClient_URLParams(t *testing.T) {
	var capturedURL string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedURL = req.URL.String()
			return NewMockResponse(200, `{}`), nil
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/search",
		Headers: map[string]string{},
		UrlParams: map[string]string{
			"q":     "test query",
			"limit": "10",
		},
	}

	_, err := StandardCallWithClient(apiInfo, false, mockClient)
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
		DoFunc: func(req *http.Request) (*http.Response, error) {
			resp := NewMockResponseWithHeaders(200, `{"data": "test"}`, map[string]string{
				"X-Server": "test-server",
			})
			return resp, nil
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/test",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}

	// Test with debug=true
	resp, err := StandardCallWithClient(apiInfo, true, mockClient)
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
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{"data": "test"}`), nil
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	// Test with debug=false
	resp, err := StandardCallWithClient(apiInfo, false, mockClient)
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
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return NewMockResponse(200, responseJSON), nil
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	resp, err := StandardCallWithClient(apiInfo, false, mockClient)
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
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return NewMockResponse(200, plainText), nil
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/text",
		Headers: map[string]string{},
	}

	resp, err := StandardCallWithClient(apiInfo, false, mockClient)
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
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{}`), nil
		},
	}

	// Test with nil Headers - should not panic
	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/test",
		Headers: nil, // explicitly nil
	}

	_, err := StandardCallWithClient(apiInfo, false, mockClient)
	if err != nil {
		t.Fatalf("unexpected error with nil headers: %v", err)
	}
}

func TestStandardCallWithClient_Duration(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return NewMockResponse(200, `{}`), nil
		},
	}

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     "http://example.com/api/test",
		Headers: map[string]string{},
	}

	resp, err := StandardCallWithClient(apiInfo, false, mockClient)
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

	apiInfo := yamlparser.ApiInfo{
		Method:  "GET",
		Url:     server.URL,
		Headers: map[string]string{},
	}

	resp, err := StandardCall(apiInfo, false)
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
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]bool
	json.Unmarshal(body, &result)

	if !result["created"] {
		t.Error("expected created=true in response")
	}
}

func TestMockServerWithHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(201)
			w.Write([]byte(`{"method": "POST"}`))
		} else {
			w.WriteHeader(200)
			w.Write([]byte(`{"method": "GET"}`))
		}
	}

	server := NewMockServerWithHandler(handler)
	defer server.Close()

	// Test GET
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET: expected 200, got %d", resp.StatusCode)
	}

	// Test POST
	resp2, err := http.Post(server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 201 {
		t.Errorf("POST: expected 201, got %d", resp2.StatusCode)
	}
}
