package migration

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// PmCollection represents the overall Postman collection
type PmCollection struct {
	Info     Info           `json:"info"`
	Variable []KeyValuePair `            josn:"variable,omitempty"`
}

// Info represents the info object in a Postman collection
type Info struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ItemOrReq can represent either a folder (with sub-items) or a request
// This handles the recursive nature of the structure
type ItemOrReq struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Item        []ItemOrReq `json:"item,omitempty"`    // Present if it's a folder
	Request     *Request    `json:"request,omitempty"` // Present if it's a request
	Event       []Event     `json:"event,omitempty"`   // For scripts (pre-request, test)
	Response    []Response  `json:"response,omitempty"`
}

// Event represents script events like tests or pre-request scripts
type Event struct {
	Listen string `json:"listen"`
	Script Script `json:"script"`
}

// Script contains the script content and type
type Script struct {
	Exec     []string       `json:"exec"`
	Type     string         `json:"type"`
	Packages map[string]any `json:"packages,omitempty"`
}

// Response represents saved responses
type Response struct {
	Name            string         `json:"name"`
	OriginalRequest Request        `json:"originalRequest"`
	Status          string         `json:"status"`
	Code            int            `json:"code"`
	Header          []KeyValuePair `json:"header"`
	Cookie          []any          `json:"cookie"`
	Body            string         `json:"body"`
}

// Request represents a Postman request
type Request struct {
	Method                  yamlParser.HTTPMethodType `json:"method"`
	Header                  []KeyValuePair            `json:"header"`
	Body                    *Body                     `json:"body,omitempty"`
	URL                     *PMURL                    `json:"url"`
	Description             string                    `json:"description,omitempty"`
	ProtocolProfileBehavior *map[string]any           `json:"protocolProfileBehavior,omitempty"`
}

// PMURL represents PMURL information in a request
type PMURL struct {
	Raw   yamlParser.URL `json:"raw"`
	Host  []string       `json:"host,omitempty"`
	Path  []string       `json:"path,omitempty"`
	Query []KeyValuePair `json:"query,omitempty"`
}

// Body represents request body with different modes
type Body struct {
	Mode       string              `json:"mode"`
	Raw        string              `json:"raw,omitempty"`
	URLEncoded []KeyValuePair      `json:"urlencoded,omitempty"`
	FormData   []KeyValuePair      `json:"formdata,omitempty"`
	GraphQL    *yamlParser.GraphQl `json:"graphql,omitempty"`
	Options    *BodyOptions        `json:"options,omitempty"`
}

// BodyOptions represents options for different body modes
type BodyOptions struct {
	Raw *RawOptions `json:"raw,omitempty"`
}

// RawOptions represents options specific to raw body mode
type RawOptions struct {
	Language string `json:"language"`
}

// IsCollection determines if the JSON contains a Postman collection
func IsCollection(jsonString map[string]any) bool {
	_, infoExists := jsonString["info"]
	_, itemExists := jsonString["item"]
	return infoExists && itemExists
}

// MigrateCollection migrates a Postman collection to the desired format
// To be implemented
func MigrateCollection(collection PmCollection) error {
	// Implementation to come
	return utils.ColorError("collection migration not yet implemented")
}

func UrlToYaml(pmURL PMURL) (string, error) {
	type YAMLOutput struct {
		URL       string            `yaml:"url"`
		URLParams map[string]string `yaml:"urlparams,omitempty"`
	}
	output := YAMLOutput{
		URLParams: make(map[string]string),
	}

	rawURL := string(pmURL.Raw)
	baseURL := rawURL

	if strings.Contains(rawURL, "?") {
		baseURL = strings.Split(rawURL, "?")[0]
	}

	baseURL = addDotToTemplate(baseURL)
	output.URL = baseURL

	// Process query parameters
	for _, param := range pmURL.Query {
		output.URLParams[addDotToTemplate(param.Key)] = addDotToTemplate(param.Value)
	}

	// If no URL params, remove the field
	if len(output.URLParams) == 0 {
		output.URLParams = nil
	}

	// Convert to YAML
	yamlBytes, err := yaml.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// convet pm header from json to yaml for hulak
func HeaderToYAML(header []KeyValuePair) (string, error) {
	if len(header) == 0 {
		return "", nil
	}

	type HeaderOutput struct {
		Headers map[string]string `yaml:"headers"`
	}

	output := HeaderOutput{
		Headers: make(map[string]string),
	}

	for _, h := range header {
		key := addDotToTemplate(h.Key)
		value := addDotToTemplate(h.Value)

		output.Headers[key] = value
	}

	yamlBytes, err := yaml.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal headers to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// BodyToYaml converts a Postman Body struct to a YAML format that matches yamlParser.Body
func BodyToYaml(pmbody Body) (string, error) {
	var yamlOutput yamlParser.Body

	switch pmbody.Mode {
	case "raw":
		if pmbody.Raw != "" {
			yamlOutput = yamlParser.Body{Raw: addDotToTemplate(pmbody.Raw)}
		}

	case "urlencoded":
		if len(pmbody.URLEncoded) > 0 {
			urlEncodedMap := make(map[string]string)
			for _, pair := range pmbody.URLEncoded {
				urlEncodedMap[addDotToTemplate(pair.Key)] = addDotToTemplate(pair.Value)
			}
			yamlOutput = yamlParser.Body{UrlEncodedFormData: urlEncodedMap}
		}

	case "formdata":
		if len(pmbody.FormData) > 0 {
			formDataMap := make(map[string]string)
			for _, pair := range pmbody.FormData {
				formDataMap[addDotToTemplate(pair.Key)] = addDotToTemplate(pair.Value)
			}
			yamlOutput = yamlParser.Body{FormData: formDataMap}
		}

	case "graphql":
		if pmbody.GraphQL != nil {
			graphql := yamlParser.GraphQl{
				Query:     addDotToTemplate(pmbody.GraphQL.Query),
				Variables: make(map[string]any),
			}

			for key, value := range pmbody.GraphQL.Variables {
				if strValue, ok := value.(string); ok {
					graphql.Variables[key] = addDotToTemplate(strValue)
				} else {
					graphql.Variables[key] = value
				}
			}

			yamlOutput = yamlParser.Body{Graphql: &graphql}
		}

	case "none", "":
		return "", nil

	default:
		return "", fmt.Errorf("unsupported body mode: %s", pmbody.Mode)
	}

	yamlBytes, err := yaml.Marshal(yamlOutput)
	if err != nil {
		return "", fmt.Errorf("failed to marshal body to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// Sudo Code
// Construct URL from Raw. Substract query from raw url ✅
// Change Value of string from {{valueK}} to {{.valueK}} // add dot after {{.}} // surround with ""  ✅
// Migrate Variables to Global with the name of where it is coming from.
// First, refactor a create folder function from envparser
// Then create file with the name from the request name. Get this from envparser
// Then recursively create folder, nested folders and request on the
