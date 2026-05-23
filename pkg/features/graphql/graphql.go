package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/tui/envselect"
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
	resp, err := apicalls.StandardCall(context.Background(), apiInfo, false)
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
			truncateBody(bodyStr, 2000),
		)
	}

	if !apicalls.IsJSON(bodyStr) {
		return Schema{}, fmt.Errorf(
			"expected JSON response but received %s (status %d).\nResponse body:\n%s",
			detectContentType(bodyStr),
			statusCode,
			truncateBody(bodyStr, 2000),
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

func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "\n... (truncated)"
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

// ResolveSecretsForEnv loads secrets and returns the resolved environment name.
// If no environment is needed, the returned env name is empty. The bool is
// true when the user cancelled the interactive picker (not an error). Other
// failures (selector failure, missing env file) come back as a non-nil error.
func ResolveSecretsForEnv(
	urlToFileMap map[string]string,
	needsEnv bool,
	env string,
) (map[string]any, string, bool, error) {
	// If env is explicitly provided, always load it (user knows what they want)
	if env != "" {
		secretsMap, selectedEnv, err := loadSecretsForEnv(env)
		if err != nil {
			return nil, "", false, err
		}
		return secretsMap, selectedEnv, false, nil
	}

	if NeedsEnvResolution(urlToFileMap) || needsEnv {
		secretsMap, selectedEnv, err := loadSecretsForEnv("")
		if err != nil {
			return nil, "", false, err
		}
		if secretsMap == nil {
			return nil, "", true, nil
		}
		return secretsMap, selectedEnv, false, nil
	}

	return map[string]any{}, "", false, nil
}

// loadSecretsForEnv loads secrets from the specified environment.
// If env is empty, shows the interactive env selector TUI; a returned
// secretsMap of nil with no error means the user cancelled the picker.
func loadSecretsForEnv(env string) (map[string]any, string, error) {
	selectedEnv := env

	// If no env provided, show the interactive selector
	if selectedEnv == "" {
		picked, cancelled, err := envselect.RunEnvSelector()
		if err != nil {
			return nil, "", fmt.Errorf("environment selector: %w", err)
		}
		if cancelled {
			return nil, "", nil
		}
		selectedEnv = picked
	}

	// Load secrets from the environment
	secretsMap, err := envparser.LoadSecretsMap(selectedEnv)
	if err != nil {
		return nil, "", fmt.Errorf("loading environment %q: %w", selectedEnv, err)
	}

	return secretsMap, selectedEnv, nil
}
