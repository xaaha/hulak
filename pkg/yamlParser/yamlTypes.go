package yamlParser

import (
	"net/http"
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

// User's yaml file
type User struct {
	UrlParams map[string]string `json:"urlparams,omitempty" yaml:"urlparams"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Method    HTTPMethodType    `json:"method,omitempty"    yaml:"method"`
	Url       string            `json:"url,omitempty"       yaml:"url"`
}

// type of Body in a yaml file
// binary type is not yet configured
// only one is possible that could be passed
type Body struct {
	FormData           map[string]string
	UrlEncodedFormData map[string]string
	Graphql            GraphQl
	RawString          string
}

type GraphQl struct {
	Variable map[string]interface{}
	Query    string
}
