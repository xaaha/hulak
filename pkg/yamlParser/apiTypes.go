package yamlParser

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

type HTTPMethodType string

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

// convert the method to uppercase
func (h *HTTPMethodType) ToUpperCase() {
	*h = HTTPMethodType(strings.ToUpper(string(*h)))
}

// enforce HTTPMethodType
func (m HTTPMethodType) IsValid() bool {
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(m)))
	switch upperCasedMethod {
	case GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE, CONNECT:
		return true
	}
	return false
}

// Struct we need to call request api
type ApiInfo struct {
	Body      io.Reader
	Headers   map[string]string
	UrlParams map[string]string
	Method    string
	Url       string
}

type URL string

// URL should not be missing
func (u *URL) IsValidURL() bool {
	userProvidedUrl := string(*u)
	_, err := url.ParseRequestURI(userProvidedUrl)
	return err == nil
}

// User's yaml file for Api request
type User struct {
	UrlParams map[string]string `json:"urlparams,omitempty" yaml:"urlparams"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Body      *Body             `json:"body,omitempty"      yaml:"body"`
	Method    HTTPMethodType    `json:"method,omitempty"    yaml:"method"`
	Url       URL               `json:"url,omitempty"       yaml:"url"`
}

// Returns ApiInfo object for the User's API request yaml file
func (user *User) PrepareStruct() (ApiInfo, error) {
	body, contentType, err := user.Body.EncodeBody()
	if err != nil {
		return ApiInfo{}, utils.ColorError("#apiTypes.go", err)
	}

	if contentType != "" {
		if user.Headers == nil {
			user.Headers = make(map[string]string)
		}
		user.Headers["content-type"] = contentType
	}

	return ApiInfo{
		Method:    string(user.Method),
		Url:       string(user.Url),
		UrlParams: user.UrlParams,
		Headers:   user.Headers,
		Body:      body,
	}, nil
}

// type of Body in a yaml file
// binary type is not yet configured
// Only one is possible that could be passed
type Body struct {
	FormData           map[string]string `json:"formdata,omitempty"           yaml:"formdata"`
	UrlEncodedFormData map[string]string `json:"urlencodedformdata,omitempty" yaml:"urlencodedformdata"`
	Graphql            *GraphQl          `json:"graphql,omitempty"            yaml:"graphql"`
	Raw                string            `json:"raw,omitempty"                yaml:"raw"`
}

// body is valid when,
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
	for i := 0; i < ln.NumField(); i++ {
		field := ln.Field(i)
		switch field.Kind() {
		case reflect.Ptr:
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

type GraphQl struct {
	Variables map[string]interface{} `json:"variables,omitempty" yaml:"variables"`
	Query     string                 `json:"query,omitempty"     yaml:"query"`
}

// Returns body for apiCall, content type header string and error if any
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

	case len(b.UrlEncodedFormData) > 0:
		encodedBody, err := EncodeXwwwFormUrlBody(b.UrlEncodedFormData)
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

// Encodes key-value pairs as "application/x-www-form-urlencoded" data.
// Returns an io.Reader containing the encoded data, or an error if the input is empty.
func EncodeXwwwFormUrlBody(keyValue map[string]string) (io.Reader, error) {
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

// Encodes multipart/form-data other than x-www-form-urlencoded,
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

// accepts query string and variables map[string]interface, then returns the payload
func EncodeGraphQlBody(query string, variables map[string]interface{}) (io.Reader, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(jsonData), nil
}
