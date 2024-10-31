package yamlParser

import (
	"github.com/xaaha/hulak/pkg/utils"
)

type KeyValuePair struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type Method struct {
	GET    string `yaml:"GET"`
	POST   string `yaml:"POST"`
	PUT    string `yaml:"PUT"`
	PATCH  string `yaml:"PATCH"`
	DELETE string `yaml:"DELETE"`
	HEAD   string `yaml:"HEAD"`
	// custom method is not supported yet
}

type User struct {
	Name   string               `yaml:"name"`
	Age    string               `yaml:"age"`
	Email  string               `yaml:"email"`
	Base   string               `yaml:"base"`
	Params []utils.KeyValuePair `yaml:"params,omitempty"`
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
