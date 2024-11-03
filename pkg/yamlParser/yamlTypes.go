package yamlParser

import (
	"net/http"
	"net/url"
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
// only one is possible that could be passed
type Body struct {
	FormData           map[string]string `json:"formdata,omitempty"           yaml:"formdata"`
	UrlEncodedFormData map[string]string `json:"urlencodedformdata,omitempty" yaml:"urlencodedformdata"`
	Graphql            *GraphQl          `json:"graphql,omitempty"            yaml:"graphql"`
	Raw                string            `json:"raw,omitempty"                yaml:"raw"`
}

type GraphQl struct {
	Variables map[string]interface{} `json:"variables,omitempty" yaml:"variables"`
	Query     string                 `json:"query,omitempty"     yaml:"query"`
}

// make sure the body is valid as well
// handle the case where body is an empty {}
// and any other case where other items is {}
// also when the body is nil
// make sure the body only has only one item

/*
body := Body{
	FormData: map[string]string{"key": "value"},
	Graphql:  &GraphQl{Query: "query string"}, // Only if Graphql has data
}
*/
