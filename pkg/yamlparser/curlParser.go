package yamlparser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// ParseCurlCommand parses a cURL command string and returns an ApiCallFile
func ParseCurlCommand(curlStr string) (*ApiCallFile, error) {
	// Clean up the string
	curlStr = cleanCurlString(curlStr)

	if curlStr == "" {
		return nil, utils.ColorError("empty curl command")
	}

	// Initialize ApiCallFile
	apiCallFile := &ApiCallFile{
		Headers:   make(map[string]string),
		URLParams: make(map[string]string),
	}

	// Extract URL (required)
	urlStr, err := extractURL(curlStr)
	if err != nil {
		return nil, err
	}
	apiCallFile.URL = URL(urlStr)

	// Parse query parameters from URL
	parseURLParams(urlStr, apiCallFile)

	// Extract method (-X or --request)
	method := extractMethod(curlStr)
	if method == "" {
		method = "GET"
	}
	apiCallFile.Method = HTTPMethodType(method)

	// Extract headers (-H or --header)
	headers := extractHeaders(curlStr)
	if len(headers) > 0 {
		apiCallFile.Headers = headers
	}

	// Extract basic auth (-u or --user)
	extractBasicAuth(curlStr, apiCallFile)

	// Extract cookies (--cookie or -b)
	extractCookies(curlStr, apiCallFile)

	// Extract body data (-d, --data, --data-raw, --data-binary, -F, --form)
	body, bodyType, formData, err := extractBody(curlStr)
	if err != nil {
		return nil, err
	}

	// Infer method from body if not explicitly set
	if apiCallFile.Method == "GET" && body != "" {
		apiCallFile.Method = HTTPMethodType("POST")
	}

	// Set body if any
	if body != "" || len(formData) > 0 {
		apiCallFile.Body = &Body{}

		switch bodyType {
		case "raw":
			// Try to pretty-print JSON if possible
			var prettyBody string
			var jsonData any

			if err := json.Unmarshal([]byte(body), &jsonData); err == nil {
				// Successfully parsed as JSON
				if pretty, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
					prettyBody = string(pretty)
				} else {
					prettyBody = body
				}
			} else {
				// Not JSON or error parsing, use as-is
				prettyBody = body
			}
			apiCallFile.Body.Raw = prettyBody

		case "form":
			if len(formData) > 0 {
				apiCallFile.Body.FormData = formData
			}

		case "urlencoded":
			if len(formData) > 0 {
				apiCallFile.Body.URLEncodedFormData = formData
			}

		case "graphql":
			// For GraphQL, parse the body to extract query and variables
			apiCallFile.Body.Graphql = parseGraphQLBody(body)
		}
	}

	// Check for unsupported flags and warn
	warnUnsupportedFlags(curlStr)

	return apiCallFile, nil
}

// cleanCurlString cleans up the curl command string
func cleanCurlString(s string) string {
	// Remove leading "curl" keyword if present
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "curl ")
	s = strings.TrimPrefix(s, "curl")

	// Handle multi-line with backslashes
	s = strings.ReplaceAll(s, "\\\n", " ")
	s = strings.ReplaceAll(s, "\\\r\n", " ")
	s = strings.ReplaceAll(s, "\\", " ")

	// Normalize whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// extractURL finds and extracts the URL from curl command
func extractURL(curlStr string) (string, error) {
	// Try to find URL in various positions
	// Pattern 1: Single quoted URL
	reSingleQuoted := regexp.MustCompile(`'(https?://[^']+)'`)
	if matches := reSingleQuoted.FindStringSubmatch(curlStr); len(matches) > 1 {
		return matches[1], nil
	}

	// Pattern 2: Double quoted URL
	reDoubleQuoted := regexp.MustCompile(`"(https?://[^"]+)"`)
	if matches := reDoubleQuoted.FindStringSubmatch(curlStr); len(matches) > 1 {
		return matches[1], nil
	}

	// Pattern 3: Unquoted URL (must be careful with spaces)
	parts := strings.FieldsSeq(curlStr)
	for part := range parts {
		if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			return part, nil
		}
	}

	return "", utils.ColorError("could not find URL in curl command")
}

// extractMethod extracts HTTP method from -X or --request flag
func extractMethod(curlStr string) string {
	patterns := []string{
		`-X\s+['"]?([A-Za-z]+)['"]?`,
		`--request\s+['"]?([A-Za-z]+)['"]?`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			return strings.ToUpper(matches[1])
		}
	}

	return ""
}

// extractHeaders extracts all headers from -H or --header flags
func extractHeaders(curlStr string) map[string]string {
	headers := make(map[string]string)

	patterns := []string{
		`-H\s+'([^']+)'`,
		`-H\s+"([^"]+)"`,
		`--header\s+'([^']+)'`,
		`--header\s+"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(curlStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				parseHeader(match[1], headers)
			}
		}
	}

	return headers
}

// parseHeader parses a single header string into key-value
func parseHeader(headerStr string, headers map[string]string) {
	parts := strings.SplitN(headerStr, ":", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value
	}
}

// extractBasicAuth extracts basic auth from -u or --user flag
func extractBasicAuth(curlStr string, apiCallFile *ApiCallFile) {
	patterns := []string{
		`-u\s+'([^']+)'`,
		`-u\s+"([^"]+)"`,
		`-u\s+([^\s]+)`,
		`--user\s+'([^']+)'`,
		`--user\s+"([^"]+)"`,
		`--user\s+([^\s]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			userPass := matches[1]
			// Encode as base64 for Authorization header
			encoded := base64.StdEncoding.EncodeToString([]byte(userPass))
			if apiCallFile.Headers == nil {
				apiCallFile.Headers = make(map[string]string)
			}
			apiCallFile.Headers["Authorization"] = "Basic " + encoded
			return
		}
	}
}

// extractCookies extracts cookies from --cookie or -b flag
func extractCookies(curlStr string, apiCallFile *ApiCallFile) {
	patterns := []string{
		`--cookie\s+'([^']+)'`,
		`--cookie\s+"([^"]+)"`,
		`-b\s+'([^']+)'`,
		`-b\s+"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			if apiCallFile.Headers == nil {
				apiCallFile.Headers = make(map[string]string)
			}
			apiCallFile.Headers["Cookie"] = matches[1]
			return
		}
	}
}

// extractBody extracts body data from various flags
// Returns body string, body type, form data (if any), and error
func extractBody(curlStr string) (string, string, map[string]string, error) {
	// Check for form data (-F, --form)
	formData := extractFormData(curlStr)
	if len(formData) > 0 {
		return "", "form", formData, nil
	}

	// Check for URL-encoded form (--data-urlencode)
	if strings.Contains(curlStr, "--data-urlencode") {
		data := extractDataUrlencode(curlStr)
		if len(data) > 0 {
			return "", "urlencoded", data, nil
		}
	}

	// Check for raw data (-d, --data, --data-raw, --data-binary)
	body := extractRawData(curlStr)
	if body != "" {
		// Try to detect if it's GraphQL
		if isGraphQLBody(body) {
			return body, "graphql", nil, nil
		} else if isFormURLEncoded(body) {
			// Parse as URL-encoded form data
			return "", "urlencoded", parseURLEncodedBody(body), nil
		} else {
			return body, "raw", nil, nil
		}
	}

	return "", "", nil, nil
}

// extractFormData extracts multipart form data
func extractFormData(curlStr string) map[string]string {
	formData := make(map[string]string)

	patterns := []string{
		`-F\s+'([^']+)'`,
		`-F\s+"([^"]+)"`,
		`--form\s+'([^']+)'`,
		`--form\s+"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(curlStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				parseFormField(match[1], formData)
			}
		}
	}

	return formData
}

// parseFormField parses form field like "key=value" or "key=@file"
func parseFormField(field string, formData map[string]string) {
	parts := strings.SplitN(field, "=", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Skip file uploads (starts with @)
		if !strings.HasPrefix(value, "@") {
			formData[key] = value
		} else {
			// Note file upload in value
			formData[key] = fmt.Sprintf("TODO: Upload file %s", strings.TrimPrefix(value, "@"))
		}
	}
}

// extractDataUrlencode extracts URL-encoded form data
func extractDataUrlencode(curlStr string) map[string]string {
	data := make(map[string]string)

	re := regexp.MustCompile(`--data-urlencode\s+'([^']+)'`)
	matches := re.FindAllStringSubmatch(curlStr, -1)

	for _, match := range matches {
		if len(match) > 1 {
			parseFormField(match[1], data)
		}
	}

	return data
}

// extractRawData extracts raw body data from -d, --data, etc.
func extractRawData(curlStr string) string {
	// Single-quoted patterns (easier to match and more reliable)
	singleQuotePatterns := []string{
		`-d\s+'([^']+)'`,
		`--data\s+'([^']+)'`,
		`--data-raw\s+'([^']+)'`,
		`--data-binary\s+'([^']+)'`,
	}

	// Try the single-quoted patterns first
	for _, pattern := range singleQuotePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			return matches[1]
		}
	}

	// Double-quoted patterns - be careful with escaped quotes
	// Note: This has limitations with complex escaping, prefer using single quotes in curl
	doubleQuotePatterns := []string{
		`-d\s+"([^"\\]*(?:\\.[^"\\]*)*)"`,
		`--data\s+"([^"\\]*(?:\\.[^"\\]*)*)"`,
		`--data-raw\s+"([^"\\]*(?:\\.[^"\\]*)*)"`,
		`--data-binary\s+"([^"\\]*(?:\\.[^"\\]*)*)"`,
	}

	for _, pattern := range doubleQuotePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			// Unescape the content - replace \" with "
			unescaped := strings.ReplaceAll(matches[1], "\\\"", "\"")
			return unescaped
		}
	}

	return ""
}

// parseURLParams extracts query parameters from URL
func parseURLParams(urlStr string, apiCallFile *ApiCallFile) {
	if !strings.Contains(urlStr, "?") {
		return
	}

	parts := strings.SplitN(urlStr, "?", 2)
	apiCallFile.URL = URL(parts[0])
	queryString := parts[1]

	// Parse query string
	params, err := url.ParseQuery(queryString)
	if err == nil {
		for key, values := range params {
			if len(values) > 0 {
				apiCallFile.URLParams[key] = values[0]
			}
		}
	}
}

// isGraphQLBody detects if body is a GraphQL query
func isGraphQLBody(body string) bool {
	return (strings.Contains(body, `"query"`) || strings.Contains(body, `'query'`)) &&
		(strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "'") || strings.HasPrefix(strings.TrimSpace(body), `"`))
}

// isFormURLEncoded checks if body is URL-encoded form data
func isFormURLEncoded(body string) bool {
	// Simple heuristic: contains = and & but not JSON-like syntax
	return strings.Contains(body, "=") && !strings.Contains(body, "{") &&
		!strings.Contains(body, "[")
}

// parseURLEncodedBody parses URL-encoded string into map
func parseURLEncodedBody(body string) map[string]string {
	data := make(map[string]string)
	params, err := url.ParseQuery(body)
	if err == nil {
		for key, values := range params {
			if len(values) > 0 {
				data[key] = values[0]
			}
		}
	}

	return data
}

// parseGraphQLBody extracts query and variables from a GraphQL JSON body
func parseGraphQLBody(body string) *GraphQl {
	var gqlData map[string]any
	if err := json.Unmarshal([]byte(body), &gqlData); err != nil {
		// If can't parse as JSON, return a basic GraphQL structure
		return &GraphQl{
			Query: body,
		}
	}

	gql := &GraphQl{}

	// Extract query string
	if query, ok := gqlData["query"].(string); ok {
		gql.Query = query
	}

	// Extract variables if present
	if variables, ok := gqlData["variables"]; ok && variables != nil {
		gql.Variables = variables
	}

	return gql
}

// warnUnsupportedFlags checks for unsupported cURL flags and warns user
func warnUnsupportedFlags(curlStr string) {
	unsupportedFlags := []struct {
		flag string
		desc string
	}{
		{"-k", "insecure/no certificate verification"},
		{"--insecure", "insecure/no certificate verification"},
		{"-L", "follow redirects"},
		{"--location", "follow redirects"},
		{"--compressed", "request compressed response"},
		{"-v", "verbose mode"},
		{"--verbose", "verbose mode"},
		{"-s", "silent mode"},
		{"--silent", "silent mode"},
		{"-i", "include headers in output"},
		{"--include", "include headers in output"},
		{"-I", "HEAD request"},
		{"--head", "HEAD request"},
		{"--max-time", "max time for request"},
		{"--connect-timeout", "connection timeout"},
	}

	for _, flag := range unsupportedFlags {
		if strings.Contains(curlStr, flag.flag) {
			utils.PrintWarning(
				fmt.Sprintf(
					"Flag '%s' (%s) is not supported and will be ignored",
					flag.flag,
					flag.desc,
				),
			)
		}
	}
}
