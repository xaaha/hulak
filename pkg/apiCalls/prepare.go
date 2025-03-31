// Package apicalls has all things related to api call
package apicalls

import (
	"encoding/json"
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

// processResponse takes in http response and returns CustomResponse type string for debugging purposes
// TODO 1: Need to fine-tune this. Basic info like status code, request time,
// should be printed by default. Everythig else should default to false and,
// --debug should set this true.
// Then fix tests
// processResponse takes in http request, response and returns CustomResponse type string for debugging purposes
func processResponse(
	req *http.Request,
	resp *http.Response,
	duration time.Duration,
) CustomResponse {
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("prepare.go: Error while reading response: %v", err)
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
	responseData := CustomResponse{
		Request: RequestInfo{
			URL:     req.URL.String(),
			Method:  req.Method,
			Headers: requestHeaders,
			Body:    "",
		},
		Response: ResponseInfo{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    responseHeaders,
		},
		HTTPInfo: tlsInfo,
		Duration: duration.String(),
	}

	var parsedBody any
	if err := json.Unmarshal(respBody, &parsedBody); err == nil {
		responseData.Response.Body = parsedBody
	} else {
		// If the body isn't valid JSON, include it as a string
		responseData.Response.Body = string(respBody)
	}

	return responseData
}
