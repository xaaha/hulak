// Package httpclient provides a shared HTTP client interface and factory
// for all outbound HTTP calls in hulak. Both the API runner and internal
// fetchers (e.g. GitHub key fetch) use this so timeout, redirect, and
// TLS behavior stay consistent and upgrade in one place.
package httpclient

import (
	"fmt"
	"io"
	"net/http"
)

// HTTPClient is the interface for making HTTP requests.
// Both Client (production) and test mocks satisfy this.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client wraps the standard *http.Client with shared defaults.
// Use New() for production, or inject a custom *http.Client for testing.
type Client struct {
	HTTP *http.Client
}

// New returns a Client with sensible defaults:
//   - No client-level timeout (use context for per-request deadlines)
//   - Redirects follow Go default (10), but HTTPS → HTTP downgrades are blocked
//
// Callers control timeout via context.WithTimeout on each request.
func New() *Client {
	return &Client{
		HTTP: &http.Client{
			CheckRedirect: safeRedirectPolicy,
		},
	}
}

// Do executes an HTTP request. Shorthand for c.HTTP.Do(req).
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.HTTP.Do(req) //nolint:gosec // G704: hulak is a CLI — URLs are operator-provided, not untrusted
}

// ReadBody reads the response body up to maxBytes and closes it.
// Pass 0 for no limit.
func ReadBody(resp *http.Response, maxBytes int64) ([]byte, error) {
	defer resp.Body.Close()
	var reader io.Reader = resp.Body
	if maxBytes > 0 {
		reader = io.LimitReader(resp.Body, maxBytes)
	}
	return io.ReadAll(reader)
}

// safeRedirectPolicy refuses HTTPS → HTTP downgrades.
// Redirect count is left to Go's default (10). Callers needing stricter
// limits (e.g. key fetcher) should enforce it at their own level.
func safeRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many redirects")
	}
	if req.URL.Scheme == "http" && len(via) > 0 && via[0].URL.Scheme == "https" {
		return fmt.Errorf("refusing redirect from HTTPS to HTTP")
	}
	return nil
}
