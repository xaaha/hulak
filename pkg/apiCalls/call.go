package apicalls

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// Takes in the jsonString from ReadYamlForHttpRequest
// And prepares the ApiInfo struct for StandardCall function
func CombineAndCall(jsonString string) ApiInfo {
	// ReadYamlForHttpRequest should not return an empty string
	// but if it does, return nil struct
	if len(jsonString) == 0 {
		utils.ColorError("call.go: jsonString constructed from yamlFile is empty")
		return ApiInfo{}
	}
	// Parse the JSON string into a User struct
	var user yamlParser.User
	err := json.Unmarshal([]byte(jsonString), &user)
	if err != nil {
		msg := "call.go: Error unmarshalling json \n" + jsonString
		utils.ColorError(msg, err)
		return ApiInfo{} // we shouldn't proceed if there is an error on processing jsonString
	}

	// Initialize the request body
	var body io.Reader

	// Handle GraphQL body if provided
	hasGraphQlBody := user.Body != nil && user.Body.Graphql != nil &&
		len(user.Body.Graphql.Query) > 0
	if hasGraphQlBody {
		body, err = EncodeGraphQlBody(user.Body.Graphql.Query, user.Body.Graphql.Variables)
		if err != nil {
			msg := "call.go: Error encoding GraphQL body of json\n" + jsonString
			utils.ColorError(msg, err)
			return ApiInfo{}
		}
	}
	var contentType string
	if user.Body != nil && user.Body.FormData != nil &&
		len(user.Body.FormData) > 0 {
		body, contentType, err = EncodeFormData(user.Body.FormData)
		user.Headers["Content-Type"] = contentType
		if err != nil {
			utils.ColorError("call.go: Error encoding multipart form data", err)
			return ApiInfo{}
		}
	}

	if user.Body != nil && user.Body.UrlEncodedFormData != nil &&
		len(user.Body.UrlEncodedFormData) > 0 {
		body, err = EncodeXwwwFormUrlBody(user.Body.UrlEncodedFormData)
		if err != nil {
			utils.ColorError("call.go: Error encoding URL-encoded form data", err)
			return ApiInfo{}
		}
	}
	// TODO: Use if else for raw string  and fix test. Form data has unique headers now

	// Handle raw body as a fallback (e.g., JSON, XML, HTML)
	if body == nil && user.Body != nil && user.Body.Raw != "" {
		body = strings.NewReader(user.Body.Raw)
	}

	// Construct and return the ApiInfo object
	return ApiInfo{
		Method:    string(user.Method),
		Url:       string(user.Url),
		UrlParams: user.UrlParams,
		Headers:   user.Headers,
		Body:      body,
	}
}
