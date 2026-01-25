// Package apicalls has all things related to api call
package apicalls

import "net/http"

// HTTPClient interface allows mocking HTTP calls in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultClient is the production HTTP client
var DefaultClient HTTPClient = &http.Client{}
