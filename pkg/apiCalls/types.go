// Package apicalls has all things related to api call
package apicalls

// CustomResponse is structure of the result to print in the console as the std output
type CustomResponse struct {
	Request  RequestInfo  `json:"request"`
	Response ResponseInfo `json:"response"`
	HTTPInfo HTTPInfo     `json:"http_info"`
}

// RequestInfo has all the information about the  request body
type RequestInfo struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// ResponseInfo has response body info
type ResponseInfo struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       any               `json:"body"`
}

// HTTPInfo Protocol, TLSVersion, CipherSuite, ServerCertInfo
type HTTPInfo struct {
	Protocol       string    `json:"protocol"`
	TLSVersion     string    `json:"tls_version,omitempty"`
	CipherSuite    uint16    `json:"cipher_suite,omitempty"`
	ServerCertInfo *CertInfo `json:"server_cert_info,omitempty"`
}

// CertInfo has Issuer and Subject
type CertInfo struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}
