// Package apicalls has all things related to api call
package apicalls

// CustomResponse is structure of the result to print and save
type CustomResponse struct {
	Request  *RequestInfo  `json:"request,omitempty"`
	Response *ResponseInfo `json:"response,omitempty"`
	HTTPInfo *HTTPInfo     `json:"http_info,omitempty"`
	Duration string        `json:"duration,omitempty"`
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
