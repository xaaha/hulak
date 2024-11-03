package yamlParser

import (
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
func (u URL) IsValidURL() bool {
	userProvidedUrl := string(u)
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
// not nil,
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

	// Return true only if there's at least one valid field
	return validFieldCount > 0
}

type GraphQl struct {
	Variables map[string]interface{} `json:"variables,omitempty" yaml:"variables"`
	Query     string                 `json:"query,omitempty"     yaml:"query"`
}

/*
body := Body{
	FormData: map[string]string{"key": "value"},
	Graphql:  &GraphQl{Query: "query string"}, // Only if Graphql has data
}
*/
