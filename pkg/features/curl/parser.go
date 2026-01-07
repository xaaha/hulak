// Package curl handles importing cURL commands into Hulak YAML files
package curl

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// CurlCommand represents a parsed cURL command
type CurlCommand struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        string
	BodyType    string // "raw", "form", "urlencoded", "graphql"
	FormData    map[string]string
	URLParams   map[string]string
	QueryString string
}

// ParseCurlCommand parses a cURL command string
func ParseCurlCommand(curlStr string) (*CurlCommand, error) {
	// Clean up the string
	curlStr = cleanCurlString(curlStr)

	if curlStr == "" {
		return nil, utils.ColorError("empty curl command")
	}

	cmd := &CurlCommand{
		Headers:   make(map[string]string),
		FormData:  make(map[string]string),
		URLParams: make(map[string]string),
	}

	// Extract URL (required)
	urlStr, err := extractURL(curlStr)
	if err != nil {
		return nil, err
	}
	cmd.URL = urlStr

	// Parse query parameters from URL
	cmd.parseURLParams()

	// Extract method (-X or --request)
	cmd.Method = extractMethod(curlStr)
	if cmd.Method == "" {
		cmd.Method = "GET"
	}

	// Extract headers (-H or --header)
	cmd.Headers = extractHeaders(curlStr)

	// Extract basic auth (-u or --user)
	if err := cmd.extractBasicAuth(curlStr); err != nil {
		return nil, err
	}

	// Extract cookies (--cookie or -b)
	cmd.extractCookies(curlStr)

	// Extract body data (-d, --data, --data-raw, --data-binary, -F, --form)
	if err := cmd.extractBody(curlStr); err != nil {
		return nil, err
	}

	// Infer method from body if not explicitly set
	if cmd.Method == "GET" && cmd.Body != "" {
		cmd.Method = "POST"
	}

	// Check for unsupported flags and warn
	cmd.warnUnsupportedFlags(curlStr)

	return cmd, nil
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
	parts := strings.Fields(curlStr)
	for _, part := range parts {
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
func (cmd *CurlCommand) extractBasicAuth(curlStr string) error {
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
			cmd.Headers["Authorization"] = "Basic " + encoded
			return nil
		}
	}

	return nil
}

// extractCookies extracts cookies from --cookie or -b flag
func (cmd *CurlCommand) extractCookies(curlStr string) {
	patterns := []string{
		`--cookie\s+'([^']+)'`,
		`--cookie\s+"([^"]+)"`,
		`-b\s+'([^']+)'`,
		`-b\s+"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			cmd.Headers["Cookie"] = matches[1]
			return
		}
	}
}

// extractBody extracts body data from various flags
func (cmd *CurlCommand) extractBody(curlStr string) error {
	// Check for form data (-F, --form)
	formData := extractFormData(curlStr)
	if len(formData) > 0 {
		cmd.FormData = formData
		cmd.BodyType = "form"
		return nil
	}

	// Check for URL-encoded form (--data-urlencode)
	if strings.Contains(curlStr, "--data-urlencode") {
		data := extractDataUrlencode(curlStr)
		if len(data) > 0 {
			cmd.FormData = data
			cmd.BodyType = "urlencoded"
			return nil
		}
	}

	// Check for raw data (-d, --data, --data-raw, --data-binary)
	body := extractRawData(curlStr)
	if body != "" {
		cmd.Body = body

		// Try to detect if it's GraphQL
		if isGraphQLBody(body) {
			cmd.BodyType = "graphql"
		} else if isFormURLEncoded(body) {
			// Parse as URL-encoded form data
			cmd.FormData = parseURLEncodedBody(body)
			cmd.BodyType = "urlencoded"
		} else {
			cmd.BodyType = "raw"
		}
	}

	return nil
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
	patterns := []string{
		`-d\s+'([^']+)'`,
		`-d\s+"([^"]+)"`,
		`--data\s+'([^']+)'`,
		`--data\s+"([^"]+)"`,
		`--data-raw\s+'([^']+)'`,
		`--data-raw\s+"([^"]+)"`,
		`--data-binary\s+'([^']+)'`,
		`--data-binary\s+"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(curlStr); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// parseURLParams extracts query parameters from URL
func (cmd *CurlCommand) parseURLParams() {
	if !strings.Contains(cmd.URL, "?") {
		return
	}

	parts := strings.SplitN(cmd.URL, "?", 2)
	cmd.URL = parts[0]
	cmd.QueryString = parts[1]

	// Parse query string
	params, err := url.ParseQuery(parts[1])
	if err == nil {
		for key, values := range params {
			if len(values) > 0 {
				cmd.URLParams[key] = values[0]
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
	return strings.Contains(body, "=") && !strings.Contains(body, "{") && !strings.Contains(body, "[")
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

// warnUnsupportedFlags checks for unsupported cURL flags and warns user
func (cmd *CurlCommand) warnUnsupportedFlags(curlStr string) {
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
			utils.PrintWarning(fmt.Sprintf("Flag '%s' (%s) is not supported and will be ignored", flag.flag, flag.desc))
		}
	}
}
