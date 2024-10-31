package yamlParser

import "github.com/xaaha/hulak/pkg/utils"

type Url struct {
	Base string
}

type User struct {
	Name  string `yaml:"name"`
	Age   string `yaml:"age"`
	Email string `yaml:"email"`
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
