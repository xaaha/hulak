package yamlparser

import (
	"fmt"
)

// IsValidForGraphQL validates a GraphQL file and applies defaults.
// Unlike IsValid, this does NOT require a body/query since the TUI will provide it.
// It ensures the file has a valid URL and applies default method (POST) and
// Content-Type header (application/json) if not specified.
func (user *ApiCallFile) IsValidForGraphQL(filePath string) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("requested api file is not valid")
	}

	// Default method to POST if not specified (GraphQL typically uses POST)
	if user.Method == "" {
		user.Method = POST
	}
	user.Method.ToUpperCase()

	// Validate method
	if !user.Method.IsValid() {
		return false, fmt.Errorf("invalid HTTP method '%s' in '%s'", user.Method, filePath)
	}

	// URL is required
	if !user.URL.IsValidURL() {
		return false, fmt.Errorf("missing or invalid URL: %s in file %s", user.URL, filePath)
	}

	// Default Content-Type header to application/json
	if user.Headers == nil {
		user.Headers = make(map[string]string)
	}

	if val, exists := user.Headers["content-type"]; !exists || val != "application/json" {
		user.Headers["content-type"] = "application/json"
	}

	return true, nil
}

// PrepareGraphQLStruct returns ApiInfo for a GraphQL request.
// Unlike PrepareStruct, this does not encode body since the query
// will be provided separately (e.g., by TUI or introspection).
func (user *ApiCallFile) PrepareGraphQLStruct() ApiInfo {
	return ApiInfo{
		Method:    string(user.Method),
		Url:       string(user.URL),
		UrlParams: user.URLParams,
		Headers:   user.Headers,
		Body:      nil, // Body will be set separately for GraphQL
	}
}
