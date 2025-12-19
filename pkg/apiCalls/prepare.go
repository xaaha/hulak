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

	"github.com/xaaha/hulak/pkg/utils"
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

// processResponse takes in http request, response and returns a CustomResponse struct for debugging purposes
func processResponse(
	req *http.Request,
	resp *http.Response,
	duration time.Duration,
	debug bool,
	reqBody []byte,
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
			Body:    string(reqBody),
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

// when the flag is -dir run all the requests concurrently this is the current behavior.
// When the flag is -dirseq, we run one file at a time as they are discovered in a directory

// DirPath is the paths for Concurrent or Sequential file run
type DirPath struct {
	Concurrent []string
	Sequential []string
}

// processDirectory sanitizes a directory path and returns all valid files
func processDirectory(dirPath string) ([]string, error) {
	var result []string
	if dirPath == "" {
		// -dir and -dirseq is empty by default. So, not returning error here
		return result, nil
	}

	cleanDir, err := utils.SanitizeDirPath(dirPath)
	if err != nil {
		return nil, err
	}

	files, err := utils.ListFiles(cleanDir)
	if err != nil {
		return nil, err
	}

	// since we save json as responses, examples, and such,
	// let's not allow json file to be run concurrently
	fileExtensions := []string{utils.YAML, utils.YML}
	for _, file := range files {
		fileIsValid := false
		for _, ext := range fileExtensions {
			if strings.HasSuffix(strings.ToLower(file), ext) {
				fileIsValid = true
				break
			}
		}
		if fileIsValid {
			result = append(result, file)
		}
	}

	return result, nil
}

// ListDirPaths lists directory paths for dir and dirseq flags
func ListDirPaths(dir, dirseq string) (DirPath, error) {
	var result DirPath

	// Process concurrent directory
	concurrentFiles, err := processDirectory(dir)
	if err != nil {
		return result, fmt.Errorf("error processing concurrent directory: %w", err)
	}
	result.Concurrent = concurrentFiles

	// Process sequential directory
	sequentialFiles, err := processDirectory(dirseq)
	if err != nil {
		return result, fmt.Errorf("error processing sequential directory: %w", err)
	}
	result.Sequential = sequentialFiles

	return result, nil
}
