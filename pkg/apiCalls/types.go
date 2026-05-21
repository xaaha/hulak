// Package apicalls has all things related to api call
package apicalls

// CustomResponse is structure of the result to print and save
type CustomResponse struct {
	Request  *RequestInfo  `json:"request,omitempty"`
	Response *ResponseInfo `json:"response,omitempty"`
	HTTPInfo *HTTPInfo     `json:"http_info,omitempty"`
	Duration string        `json:"duration,omitempty"`

	// contentType captures the response Content-Type header so the saved
	// file can use the right extension (#208). Unexported so it doesn't
	// leak into the JSON output users see — encoding/json skips it.
	contentType string

	// rawBody is the exact bytes returned by the server. Used in default
	// (non-debug) mode so saved files and stdout output match what the
	// server sent, byte-for-byte for non-JSON content and pretty-printed
	// for JSON. Unexported — JSON encoder ignores it.
	rawBody []byte
}

// isDebug reports whether this response was built in debug mode.
// processResponse only sets Request when --debug is on, so its presence
// is a reliable signal. A dedicated bool field would push the struct
// over the lint size threshold for pass-by-value.
func (r *CustomResponse) isDebug() bool {
	return r.Request != nil
}

// RequestInfo has all the information about the  request body
type RequestInfo struct {
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

// ResponseInfo has response body info
type ResponseInfo struct {
	StatusCode int               `json:"status_code,omitempty"`
	Status     string            `json:"status,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       any               `json:"body,omitempty"`
}

// HTTPInfo Protocol, TLSVersion, CipherSuite, ServerCertInfo
type HTTPInfo struct {
	Protocol       string    `json:"protocol,omitempty"`
	TLSVersion     string    `json:"tls_version,omitempty"`
	CipherSuite    uint16    `json:"cipher_suite,omitempty"`
	ServerCertInfo *CertInfo `json:"server_cert_info,omitempty"`
}

// CertInfo has Issuer and Subject
type CertInfo struct {
	Issuer  string `json:"issuer,omitempty"`
	Subject string `json:"subject,omitempty"`
}
