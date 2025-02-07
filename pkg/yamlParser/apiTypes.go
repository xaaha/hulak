package yamlParser

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

type URL string

// URL should not be missing
func (u *URL) IsValidURL() bool {
	userProvidedUrl := string(*u)
	_, err := url.ParseRequestURI(userProvidedUrl)
	return err == nil
}

// User's yaml file
type User struct {
	UrlParams map[string]string `json:"urlparams,omitempty" yaml:"urlparams"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Body      *Body             `json:"body,omitempty"      yaml:"body"`
	Method    HTTPMethodType    `json:"method,omitempty"    yaml:"method"`
	Url       URL               `json:"url,omitempty"       yaml:"url"`
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

// body is valid if
// body is not empty ({}),
// has only one expected Body type,
// those body type is not empty,
// is not nil, and
// if the body has graphql key, it has at least query on it
func (b *Body) IsValid() bool {
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

// helper function to determine the body type
func (b *Body) BodyType() {}

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
		return nil, fmt.Errorf("no valid key-value pairs to encode")
	}

	// Encode form data to "x-www-form-urlencoded" format
	return strings.NewReader(formData.Encode()), nil
}

// Encodes multipart/form-data other than x-www-form-urlencoded,
// Returns the payload, Content-Type for the headers and error
func EncodeFormData(keyValue map[string]string) (io.Reader, string, error) {
	if len(keyValue) == 0 {
		return nil, "", fmt.Errorf("no key-value pairs to encode")
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
