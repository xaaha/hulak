package yamlparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
)

type HTTPMethodType string

// All supported methos
const (
	GET     HTTPMethodType = http.MethodGet
	POST    HTTPMethodType = http.MethodPost
	PUT     HTTPMethodType = http.MethodPut
	PATCH   HTTPMethodType = http.MethodPatch
	DELETE  HTTPMethodType = http.MethodDelete
	HEAD    HTTPMethodType = http.MethodHead
	OPTIONS HTTPMethodType = http.MethodOptions
	TRACE   HTTPMethodType = http.MethodTrace
	CONNECT HTTPMethodType = http.MethodConnect
)

// ToUpperCase convert the method to uppercase
func (h *HTTPMethodType) ToUpperCase() {
	*h = HTTPMethodType(strings.ToUpper(string(*h)))
}

// IsValid enforce HTTPMethodType
func (h HTTPMethodType) IsValid() bool {
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(h)))
	switch upperCasedMethod {
	case GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE, CONNECT:
		return true
	}
	return false
}

// Struct we need to call request api
type APIInfo struct {
	Body      io.Reader
	Headers   map[string]string
	URLParams map[string]string
	Method    string
	URL       string
}

type URL string

// IsValidURL URL should not be missing
func (u *URL) IsValidURL() bool {
	userProvidedURL := string(*u)
	_, err := url.ParseRequestURI(userProvidedURL)
	return err == nil
}

// APICallFile represents user's yaml file for api request
type APICallFile struct {
	URLParams map[string]string `json:"urlparams,omitempty" yaml:"urlparams"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Body      *Body             `json:"body,omitempty"      yaml:"body"`
	Method    HTTPMethodType    `json:"method,omitempty"    yaml:"method"`
	URL       URL               `json:"url,omitempty"       yaml:"url"`
}

// IsValid checks whether the user has valid file
func (user *APICallFile) IsValid(filePath string) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("requested api file is not valid")
	}

	user.Method.ToUpperCase()

	// method is required for any http request
	if !user.Method.IsValid() {
		if user.Method == "" {
			return false, fmt.Errorf("missing or empty HTTP method in '%s'", filePath)
		}
		return false, fmt.Errorf("invalid HTTP method '%s' in '%s'", user.Method, filePath)
	}

	// url is required for any http request
	if !user.URL.IsValidURL() {
		return false, fmt.Errorf("missing or invalid URL: %s in file %s", user.URL, filePath)
	}

	if !user.Body.IsValid() {
		utils.PanicRedAndExit(
			"Invalid Body in '%s'. Make sure body contains only one valid argument.\n %v",
			filePath,
			user.Body,
		)
	}
	return true, nil
}

// Returns APIInfo object for the User's API request yaml file
func (user *APICallFile) PrepareStruct() (APIInfo, error) {
	body, contentType, err := user.Body.EncodeBody()
	if err != nil {
		return APIInfo{}, utils.ColorError("#apiTypes.go", err)
	}

	if contentType != "" {
		if user.Headers == nil {
			user.Headers = make(map[string]string)
		}
		user.Headers["content-type"] = contentType
	}

	return APIInfo{
		Method:    string(user.Method),
		URL:       string(user.URL),
		URLParams: user.URLParams,
		Headers:   user.Headers,
		Body:      body,
	}, nil
}

// Body represents Body in a yaml file
// binary type is not yet configured
// Only one is possible that could be passed
type Body struct {
	FormData           map[string]string `json:"formdata,omitempty"           yaml:"formdata"`
	URLEncodedFormData map[string]string `json:"urlencodedformdata,omitempty" yaml:"urlencodedformdata"`
	Graphql            *GraphQl          `json:"graphql,omitempty"            yaml:"graphql"`
	Raw                string            `json:"raw,omitempty"                yaml:"raw"`
}

// IsValid checks whether body is valid when,
// if body is present, it's not nil
// has only one expected Body type,
// those body type is not empty,
// is not nil, and
// if the body has graphql key, it has at least query on it
func (b *Body) IsValid() bool {
	// it's allowed for yaml files to not have any body
	if b == nil {
		return true
	}
	validFieldCount := 0
	ln := reflect.ValueOf(*b)
	for i := range ln.NumField() {
		field := ln.Field(i)
		switch field.Kind() {
		case reflect.Pointer:
			// If the pointer is non-nil, it's valid
			if !field.IsNil() {
				validFieldCount++
			}
		case reflect.Map:
			// If the map has at least one element, it's valid
			if field.Len() > 0 {
				validFieldCount++
			}
		case reflect.String:
			// If the string is non-empty, it's valid
			if field.Len() > 0 {
				validFieldCount++
			}
		default:
			// If there's an unexpected kind, consider it invalid
			return false
		}
	}

	// Return true only if there's only 1 correct body type
	return validFieldCount == 1
}

// GraphQl inside body
type GraphQl struct {
	Query     string `json:"query"     yaml:"query"`
	Variables any    `json:"variables" yaml:"variables"`
}

// EncodeBody returns body for apiCall, content type header string and error if any
func (b *Body) EncodeBody() (io.Reader, string, error) {
	var body io.Reader
	var contentType string

	if b == nil {
		return nil, "", nil
	}

	switch {
	case b.Graphql != nil && b.Graphql.Query != "":
		encodedBody, err := EncodeGraphQlBody(b.Graphql.Query, b.Graphql.Variables)
		if err != nil {
			return nil, "", utils.ColorError("error encoding GraphQL body", err)
		}
		body = encodedBody

	case len(b.FormData) > 0:
		encodedBody, ct, err := EncodeFormData(b.FormData)
		if err != nil {
			return nil, "", utils.ColorError("error encoding multipart form data: %w", err)
		}
		body, contentType = encodedBody, ct

	case len(b.URLEncodedFormData) > 0:
		encodedBody, err := EncodeXwwwFormURLBody(b.URLEncodedFormData)
		if err != nil {
			return nil, "", utils.ColorError("error encoding URL-encoded form data: %w", err)
		}
		body, contentType = encodedBody, "application/x-www-form-urlencoded"

	case b.Raw != "":
		body = strings.NewReader(b.Raw)

	default:
		return nil, "", utils.ColorError("no valid body type provided")
	}

	return body, contentType, nil
}

// EncodeXwwwFormURLBody encodes key-value pairs as "application/x-www-form-urlencoded" data.
// Returns an io.Reader containing the encoded data, or an error if the input is empty.
func EncodeXwwwFormURLBody(keyValue map[string]string) (io.Reader, error) {
	// Initialize form data
	formData := url.Values{}

	// Populate form data, using Set to overwrite duplicate keys if any
	for key, val := range keyValue {
		if key != "" && val != "" {
			formData.Set(key, val)
		}
	}

	// Return an error if no valid key-value pairs were found
	if len(formData) == 0 {
		return nil, utils.ColorError("no valid key-value pairs to encode")
	}

	// Encode form data to "x-www-form-urlencoded" format
	return strings.NewReader(formData.Encode()), nil
}

// EncodeFormData encodes multipart/form-data other than x-www-form-urlencoded,
// Returns the payload, Content-Type for the headers and error
func EncodeFormData(keyValue map[string]string) (io.Reader, string, error) {
	if len(keyValue) == 0 {
		return nil, "", utils.ColorError("no key-value pairs to encode")
	}

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	defer writer.Close() // Ensure writer is closed

	for key, val := range keyValue {
		if key != "" && val != "" {
			if err := writer.WriteField(key, val); err != nil {
				return nil, "", err
			}
		}
	}

	// Return the payload and the content type for the header
	return payload, writer.FormDataContentType(), nil
}

// EncodeGraphQlBody accepts a query string and variables of any type,
// and returns an encoded GraphQL payload as an io.Reader
func EncodeGraphQlBody(query string, variables any) (io.Reader, error) {
	// Validate query
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("graphql query cannot be empty")
	}

	// Create the payload
	payload := GraphQl{
		Query: query,
	}

	// Handle variables if present
	if variables != nil {
		processed, err := processVariable(variables)
		if err != nil {
			return nil, fmt.Errorf("error processing variables: %w", err)
		}
		payload.Variables = processed
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL payload: %w", err)
	}

	return bytes.NewReader(jsonData), nil
}

// AddKeyValueToFormData is a helper function to dynamically add Key Value pair to FormData
func (b *Body) AddKeyValueToFormData(key, value string) {
	if b.FormData == nil {
		b.FormData = make(map[string]string)
	}
	b.FormData[key] = value
}

// AddKeyValueToURLEncodedFormData helper function to dynamically add Key Value pair to UrlEncodedFormData
func (b *Body) AddKeyValueToURLEncodedFormData(key, value string) {
	if b.URLEncodedFormData == nil {
		b.URLEncodedFormData = make(map[string]string)
	}
	b.URLEncodedFormData[key] = value
}

// processVariable handles different types of variables and ensures they're properly encoded
func processVariable(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case string, bool, int, int32, int64, float32, float64:
		// Basic types can be used as-is
		return val, nil

	case time.Time:
		// Convert time to ISO 8601 format
		return val.Format(time.RFC3339), nil

	case []any:
		// Process array elements
		processed := make([]any, len(val))
		for i, item := range val {
			p, err := processVariable(item)
			if err != nil {
				return nil, err
			}
			processed[i] = p
		}
		return processed, nil

	case map[string]any:
		// Process nested maps
		processed := make(map[string]any, len(val))
		for k, item := range val {
			p, err := processVariable(item)
			if err != nil {
				return nil, err
			}
			processed[k] = p
		}
		return processed, nil

	case json.RawMessage:
		// Handle raw JSON
		var parsed any
		if err := json.Unmarshal(val, &parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON in variable: %w", err)
		}
		return processVariable(parsed)

	default:
		// Try to convert other types to JSON and back to ensure they're properly encoded
		jsonData, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("unsupported variable type %T: %w", val, err)
		}

		var processed any
		if err := json.Unmarshal(jsonData, &processed); err != nil {
			return nil, fmt.Errorf("failed to process variable type %T: %w", val, err)
		}
		return processed, nil
	}
}
