package yamlParser

import (
	"net/http"

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

// enforce HTTPMethodType
func (m HTTPMethodType) IsValid() bool {
	switch m {
	case GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE, CONNECT:
		return true
	}
	return false
}

// User's yaml file
type User struct {
	Method    HTTPMethodType    `json:"method,omitempty"    yaml:"method"`
	UrlParams map[string]string `json:"urlparams,omitempty" yaml:"urlparams"`
	Url       string            `json:"url,omitempty"       yaml:"url"`
}

type GraphQl struct {
	Variable map[string]interface{}
	Query    string
}

// type of Body in a yaml file
// binary type is not yet configured
// only one is possible that could be passed
type Body struct {
	Graphql            GraphQl
	RawString          string
	FormData           []utils.KeyValuePair
	UrlEncodedFormData []utils.KeyValuePair
}
