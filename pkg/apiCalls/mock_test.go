package apicalls

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
)

// MockHTTPClient implements HTTPClient interface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do executes the mock function
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// NewMockResponse creates a mock HTTP response with the given status code and body
func NewMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// NewMockResponseWithHeaders creates a mock HTTP response with custom headers
func NewMockResponseWithHeaders(
	statusCode int,
	body string,
	headers map[string]string,
) *http.Response {
	resp := NewMockResponse(statusCode, body)
	for key, val := range headers {
		resp.Header.Set(key, val)
	}
	return resp
}

// NewMockServer creates a test server that returns a fixed response
// Caller is responsible for calling server.Close()
func NewMockServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

// NewMockServerWithHandler creates a test server with a custom handler
// Caller is responsible for calling server.Close()
func NewMockServerWithHandler(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// ErrMockNetwork is a mock network error for testing
var ErrMockNetwork = errors.New("mock network error")
