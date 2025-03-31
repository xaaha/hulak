// Package apicalls has all things related to api call
package apicalls

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PrepareURL perpares and returns the full url.
// If the url has parameters, then the function returns the provided baseUrl
func PrepareURL(baseURL string, urlParams map[string]string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		// If parsing fails, return the base URL as is
		return baseURL
	}
	// Prepare URL query parameters if params are provided
	if urlParams != nil {
		queryParams := url.Values{}
		for key, val := range urlParams {
			queryParams.Add(key, val)
		}
		u.RawQuery = queryParams.Encode()
	}
	return u.String()
}

// TODO: 1 Then fix tests
// processResponse takes in http request, response and returns CustomResponse type string for debugging purposes
func processResponse(
	req *http.Request,
	resp *http.Response,
	duration time.Duration,
	debug bool,
) CustomResponse {
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("prepare.go: Error while reading response: %v", err)
	}

	// Formatting the duration to two decimal points
	durationFormatted := fmt.Sprintf(
		"%.2fms",
		float64(duration.Milliseconds())+float64(duration.Microseconds()%1000)/1000.0,
	)

	var responseBody any
	if err := json.Unmarshal(respBody, &responseBody); err != nil {
		responseBody = string(respBody)
	}

	if !debug {
		// Return minimal set of data
		return CustomResponse{
			Response: &ResponseInfo{
				StatusCode: resp.StatusCode,
				Body:       responseBody,
			},
			Duration: durationFormatted,
		}
	}

	// Reading Response Headers
	responseHeaders := make(map[string]string)
	for name, values := range resp.Header {
		responseHeaders[name] = strings.Join(values, ", ")
	}

	// Reading Request Headers
	requestHeaders := make(map[string]string)
	for name, values := range req.Header {
		requestHeaders[name] = strings.Join(values, ", ")
	}

	// Preparing TLS Info
	var tlsInfo HTTPInfo
	if resp.TLS != nil {
		var issuers []string
		var subjects []string
		for _, cert := range resp.TLS.PeerCertificates {
			issuers = append(issuers, cert.Issuer.String())
			subjects = append(subjects, cert.Subject.String())
		}

		tlsInfo = HTTPInfo{
			Protocol:    resp.Proto,
			TLSVersion:  resp.TLS.NegotiatedProtocol,
			CipherSuite: resp.TLS.CipherSuite,
			ServerCertInfo: &CertInfo{
				Issuer:  strings.Join(issuers, ", "),
				Subject: strings.Join(subjects, ", "),
			},
		}
	} else {
		tlsInfo = HTTPInfo{
			Protocol: resp.Proto,
		}
	}
	return CustomResponse{
		Request: &RequestInfo{
			URL:     req.URL.String(),
			Method:  req.Method,
			Headers: requestHeaders,
		},
		Response: &ResponseInfo{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    responseHeaders,
			Body:       responseBody,
		},
		HTTPInfo: &tlsInfo,
		Duration: durationFormatted,
	}
}
