// Package curl handles importing cURL commands into Hulak YAML files
package curl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
)

// ConvertToYAML converts parsed cURL command to Hulak YAML format
func ConvertToYAML(cmd *CurlCommand) (string, error) {
	var yamlParts []string

	// Start with separator and method
	yamlParts = append(yamlParts, "---")
	yamlParts = append(yamlParts, fmt.Sprintf("method: %s", cmd.Method))

	// Add kind for GraphQL
	if cmd.BodyType == "graphql" {
		yamlParts = append(yamlParts, "kind: GraphQL")
	}

	// Add URL
	yamlParts = append(yamlParts, fmt.Sprintf("url: %q", cmd.URL))

	// Add URL params if any
	if len(cmd.URLParams) > 0 {
		paramsYAML, err := mapToYAML("urlparams", cmd.URLParams, 0)
		if err != nil {
			return "", err
		}
		yamlParts = append(yamlParts, paramsYAML)
	}

	// Add headers if any
	if len(cmd.Headers) > 0 {
		headersYAML, err := mapToYAML("headers", cmd.Headers, 0)
		if err != nil {
			return "", err
		}
		yamlParts = append(yamlParts, headersYAML)
	}

	// Add body
	if err := appendBody(cmd, &yamlParts); err != nil {
		return "", err
	}

	return strings.Join(yamlParts, "\n"), nil
}

// appendBody adds the body section to YAML
func appendBody(cmd *CurlCommand, yamlParts *[]string) error {
	if cmd.Body == "" && len(cmd.FormData) == 0 {
		return nil
	}

	*yamlParts = append(*yamlParts, "body:")

	switch cmd.BodyType {
	case "graphql":
		return appendGraphQLBody(cmd.Body, yamlParts)

	case "form":
		formYAML, err := mapToYAML("formdata", cmd.FormData, 1)
		if err != nil {
			return err
		}
		*yamlParts = append(*yamlParts, indentLines(formYAML, 1))

	case "urlencoded":
		formYAML, err := mapToYAML("urlencodedformdata", cmd.FormData, 1)
		if err != nil {
			return err
		}
		*yamlParts = append(*yamlParts, indentLines(formYAML, 1))

	case "raw":
		// Pretty-print JSON if possible
		prettyBody, err := prettyPrintIfJSON(cmd.Body)
		if err != nil {
			// Not JSON, use as-is
			prettyBody = cmd.Body
		}

		// Check if it's single-line or multi-line
		if strings.Contains(prettyBody, "\n") {
			// Multi-line: use | syntax
			*yamlParts = append(*yamlParts, indentLines("raw: |", 1))
			for _, line := range strings.Split(prettyBody, "\n") {
				*yamlParts = append(*yamlParts, indentLines(line, 2))
			}
		} else {
			// Single-line: inline
			escapedBody := strings.ReplaceAll(prettyBody, `"`, `\"`)
			*yamlParts = append(*yamlParts, indentLines(fmt.Sprintf("raw: %q", escapedBody), 1))
		}
	}

	return nil
}

// appendGraphQLBody adds GraphQL body with proper formatting
func appendGraphQLBody(body string, yamlParts *[]string) error {
	var gqlData map[string]any
	if err := json.Unmarshal([]byte(body), &gqlData); err != nil {
		return utils.ColorError("failed to parse GraphQL body: %w", err)
	}

	*yamlParts = append(*yamlParts, indentLines("graphql:", 1))

	// Add query (multiline string)
	if query, ok := gqlData["query"].(string); ok {
		*yamlParts = append(*yamlParts, indentLines("query: |", 2))
		queryLines := strings.Split(strings.TrimSpace(query), "\n")
		for _, line := range queryLines {
			*yamlParts = append(*yamlParts, indentLines(strings.TrimSpace(line), 3))
		}
	}

	// Add variables if present
	if variables, ok := gqlData["variables"]; ok && variables != nil {
		// Check if variables is empty object
		varsMap, isMap := variables.(map[string]any)
		if isMap && len(varsMap) == 0 {
			// Skip empty variables
			return nil
		}

		varsYAML, err := yaml.Marshal(map[string]any{"variables": variables})
		if err != nil {
			return err
		}
		varsStr := strings.TrimSpace(string(varsYAML))

		// Parse the YAML and re-indent properly
		lines := strings.Split(varsStr, "\n")
		if len(lines) > 0 {
			// First line is "variables:"
			*yamlParts = append(*yamlParts, indentLines("variables:", 2))
			// Rest are the variable fields
			for i := 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) != "" {
					*yamlParts = append(*yamlParts, indentLines(lines[i], 3))
				}
			}
		}
	}

	return nil
}

// mapToYAML converts a map to YAML with given key
func mapToYAML(key string, data map[string]string, indentLevel int) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	yamlData, err := yaml.Marshal(map[string]map[string]string{key: data})
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(yamlData))
	if indentLevel > 0 {
		result = indentLines(result, indentLevel)
	}

	return result, nil
}

// indentLines adds indentation to a string (all lines)
func indentLines(s string, level int) string {
	if level == 0 {
		return s
	}

	prefix := strings.Repeat("  ", level)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// prettyPrintIfJSON attempts to pretty-print JSON, returns original if not JSON
func prettyPrintIfJSON(body string) (string, error) {
	var jsonData any
	if err := json.Unmarshal([]byte(body), &jsonData); err != nil {
		return body, err
	}

	pretty, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return body, err
	}

	return string(pretty), nil
}
