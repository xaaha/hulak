package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// FetchAndParseSchema makes an introspection query and parses the schema.
// It takes an APIInfo, sets the introspection query as the body, makes the HTTP call,
// parses the response, and converts it to our domain Schema model.
func FetchAndParseSchema(apiInfo yamlparser.APIInfo) (Schema, error) {
	// Prepare introspection query body
	introspectionBody := map[string]any{"query": IntrospectionQuery}
	jsonData, err := json.Marshal(introspectionBody)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to marshal introspection query: %w", err)
	}

	// Set the body
	apiInfo.Body = bytes.NewReader(jsonData)

	// Make the HTTP call
	resp, err := apicalls.StandardCall(apiInfo, false)
	if err != nil {
		return Schema{}, fmt.Errorf("introspection request failed: %w", err)
	}

	// Extract response body
	if resp.Response == nil {
		return Schema{}, fmt.Errorf("no response data received")
	}

	// Convert body to JSON bytes
	var bodyBytes []byte
	switch v := resp.Response.Body.(type) {
	case string:
		bodyBytes = []byte(v)
	case []byte:
		bodyBytes = v
	default:
		// Body might already be parsed JSON, marshal it back
		bodyBytes, err = json.Marshal(v)
		if err != nil {
			return Schema{}, fmt.Errorf("failed to process response body: %w", err)
		}
	}

	statusCode := resp.Response.StatusCode
	bodyStr := string(bodyBytes)

	if statusCode < 200 || statusCode >= 300 {
		return Schema{}, fmt.Errorf(
			"introspection request returned status %d (%s).\nResponse body:\n%s",
			statusCode,
			resp.Response.Status,
			truncateBody(bodyStr, 500),
		)
	}

	if !apicalls.IsJSON(bodyStr) {
		return Schema{}, fmt.Errorf(
			"expected JSON response but received %s (status %d).\nResponse body:\n%s",
			detectContentType(bodyStr),
			statusCode,
			truncateBody(bodyStr, 500),
		)
	}

	introspectionData, err := ParseIntrospectionResponse(bodyBytes)
	if err != nil {
		return Schema{}, err
	}

	schema, err := ConvertToSchema(introspectionData)
	if err != nil {
		return Schema{}, err
	}

	return schema, nil
}

func detectContentType(body string) string {
	switch {
	case apicalls.IsHTML(body):
		return "HTML"
	case apicalls.IsXML(body):
		return "XML"
	default:
		return "non-JSON"
	}
}

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "\n... (truncated)"
}

// GetSecretsForEnv loads secrets based on whether templates exist in the files.
// If env is provided, uses that environment directly.
// If env is empty and templates are needed, shows the interactive selector.
// Returns empty map if no templates needed, nil if user cancelled.
func GetSecretsForEnv(urlToFileMap map[string]string, needsEnv bool, env string) map[string]any {
	// If env is explicitly provided, always load it (user knows what they want)
	if env != "" {
		return loadSecretsForEnv(env)
	}

	if NeedsEnvResolution(urlToFileMap) || needsEnv {
		return loadSecretsForEnv("")
	}

	return map[string]any{}
}

// loadSecretsForEnv loads secrets from the specified environment.
// If env is empty, shows the interactive env selector TUI.
// Returns nil if user cancelled the selector.
func loadSecretsForEnv(env string) map[string]any {
	selectedEnv := env

	// If no env provided, show the interactive selector
	if selectedEnv == "" {
		var err error
		selectedEnv, err = envselect.RunEnvSelector()
		if err != nil {
			utils.PanicRedAndExit("Environment selector error: %v", err)
		}
		if selectedEnv == "" {
			return nil
		}
	}

	// Load secrets from the environment
	secretsMap, err := envparser.LoadSecretsMap(selectedEnv)
	if err != nil {
		utils.PanicRedAndExit("Failed to load environment '%s': %v", selectedEnv, err)
	}

	return secretsMap
}
