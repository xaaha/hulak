package migration

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// PmCollection represents the overall Postman collection
type PmCollection struct {
	Info     Info           `json:"info"`
	Variable []KeyValuePair `json:"variable,omitempty"`
	Item     []ItemOrReq    `json:"item"`
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

type GraphQl struct {
	Variables string `json:"variables,omitempty" yaml:"variables"`
	Query     string `json:"query,omitempty"     yaml:"query"`
}

// Body represents request body with different modes
type Body struct {
	Mode       string         `json:"mode"`
	Raw        string         `json:"raw,omitempty"`
	URLEncoded []KeyValuePair `json:"urlencoded,omitempty"`
	FormData   []KeyValuePair `json:"formdata,omitempty"`
	GraphQL    *GraphQl       `json:"graphql,omitempty"`
	Options    *BodyOptions   `json:"options,omitempty"`
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

func MethodToYaml(method yamlParser.HTTPMethodType) (string, error) {
	type YAMLOutput struct {
		Method string `yaml:"method"`
	}

	output := YAMLOutput{
		Method: string(method),
	}

	yamlBytes, err := yaml.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal method to YAML: %w", err)
	}

	return string(yamlBytes), nil
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
	yamlOutput := make(map[string]any)

	switch pmbody.Mode {
	case "raw":
		if pmbody.Raw != "" {
			yamlOutput["raw"] = addDotToTemplate(pmbody.Raw)
		}

	case "urlencoded":
		if len(pmbody.URLEncoded) > 0 {
			urlEncodedMap := make(map[string]string)
			for _, pair := range pmbody.URLEncoded {
				urlEncodedMap[addDotToTemplate(pair.Key)] = addDotToTemplate(pair.Value)
			}
			yamlOutput["urlencodedformdata"] = urlEncodedMap
		}

	case "formdata":
		if len(pmbody.FormData) > 0 {
			formDataMap := make(map[string]string)
			for _, pair := range pmbody.FormData {
				formDataMap[addDotToTemplate(pair.Key)] = addDotToTemplate(pair.Value)
			}
			yamlOutput["formdata"] = formDataMap
		}

	case "graphql":
		if pmbody.GraphQL != nil {
			graphql := make(map[string]any)
			graphql["query"] = addDotToTemplate(pmbody.GraphQL.Query)

			if len(pmbody.GraphQL.Variables) > 0 {
				variables := make(map[string]any)
				gqlVarmap := createMap(pmbody.GraphQL.Variables)
				for key, value := range gqlVarmap {
					if strValue, ok := value.(string); ok {
						variables[key] = addDotToTemplate(strValue)
					} else {
						variables[key] = value
					}
				}
				graphql["variables"] = variables
			}

			yamlOutput["graphql"] = graphql
		}

	case "none", "":
		return "", nil

	default:
		return "", fmt.Errorf("unsupported body mode: %s", pmbody.Mode)
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(yamlOutput)
	if err != nil {
		return "", fmt.Errorf("failed to marshal body to YAML: %w", err)
	}

	return strings.TrimSpace(string(yamlBytes)), nil
}

// ConvertRequestToYAML converts a Postman collection file to YAML format
func ConvertRequestToYAML(jsonStr map[string]any) (string, error) {
	// Convert the map[string]any to JSON bytes for unmarshaling into PmCollection
	jsonBytes, err := json.Marshal(jsonStr)
	if err != nil {
		return "", fmt.Errorf("failed to marshal content: %w", err)
	}

	// Parse JSON into PmCollection struct
	var collection PmCollection
	if err := json.Unmarshal(jsonBytes, &collection); err != nil {
		return "", fmt.Errorf("failed to parse collection structure: %w", err)
	}

	var yamlParts []string

	// Add collection info as a comment
	collectionInfo := fmt.Sprintf("# Collection: %s\n", collection.Info.Name)
	if collection.Info.Description != "" {
		collectionInfo += fmt.Sprintf("# Description: %s\n", collection.Info.Description)
	}
	yamlParts = append(yamlParts, collectionInfo)

	// Process each item in the collection
	for _, item := range collection.Item {
		if item.Request == nil {
			continue
		}

		// Convert method to YAML
		methodYAML, err := MethodToYaml(item.Request.Method)
		if err != nil {
			return "", fmt.Errorf("failed to convert method for request '%s': %w", item.Name, err)
		}

		// Convert URL to YAML
		urlYAML, err := UrlToYaml(*item.Request.URL)
		if err != nil {
			return "", fmt.Errorf("failed to convert URL for request '%s': %w", item.Name, err)
		}

		// Convert headers to YAML
		headerYAML, err := HeaderToYAML(item.Request.Header)
		if err != nil {
			return "", fmt.Errorf("failed to convert headers for request '%s': %w", item.Name, err)
		}

		// Convert body to YAML if it exists
		var bodyYAML string
		if item.Request.Body != nil {
			var err error
			bodyYAML, err = BodyToYaml(*item.Request.Body)
			if err != nil {
				return "", fmt.Errorf("failed to convert body for request '%s': %w", item.Name, err)
			}
		}

		// Build request YAML
		requestYAML := fmt.Sprintf("# Request: %s\n", item.Name)
		if item.Description != "" {
			requestYAML += fmt.Sprintf("# Description: %s\n", item.Description)
		}

		// Remove prefixes and clean up the components
		methodYAML = strings.TrimPrefix(strings.TrimSpace(methodYAML), "method:")
		urlYAML = strings.TrimSpace(urlYAML)
		headerYAML = strings.TrimSpace(headerYAML)
		bodyYAML = strings.TrimSpace(bodyYAML)

		// Combine all parts with proper indentation
		requestYAML += fmt.Sprintf("method:%s\n", methodYAML)
		requestYAML += urlYAML + "\n"

		if headerYAML != "" {
			requestYAML += headerYAML + "\n"
		}

		if bodyYAML != "" {
			requestYAML += bodyYAML + "\n"
		}

		yamlParts = append(yamlParts, requestYAML)
	}

	// Add metadata as comments
	metadata := fmt.Sprintf("# Generated: %s\n# User: %s\n",
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		"xaaha")

	// Combine everything with separators
	finalYAML := metadata + strings.Join(yamlParts, "\n---\n\n")
	return finalYAML, nil
}

// Sudo Code
// Migrate Variables to Global with the name of where it is coming from.
// First, refactor a create folder function from envparser
// Then create file with the name from the request name. Get this from envparser
// Then recursively create folder, nested folders and request on the
