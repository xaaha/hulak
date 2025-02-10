package apicalls

import "github.com/xaaha/hulak/pkg/yamlParser"

// structure of the result to print in the console as the std output
type CustomResponse struct {
	Body           interface{} `json:"Body"`
	ResponseStatus string      `json:"Response Status"`
}

// Implements GetApiConfig function that returns ApiInfo struct
type ApiInfoProvider interface {
	GetApiConfig(secretsMap map[string]interface{}, path string) (yamlParser.ApiInfo, error)
}
