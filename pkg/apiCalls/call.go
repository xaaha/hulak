package apicalls

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// reads the json file based on the user's flag
// if the flag is absent, it panics.
// Finally, it uses the json from yaml with StandardCall

// TODO handle the rest of the situation for body.... Raw could be xml, json, html. Does string handles everthing?
// Need to test with real apis. Also, file references
func CombineAndCall(jsonString string) ApiInfo {
	// Parse the JSON string into a User struct
	var user yamlParser.User
	err := json.Unmarshal([]byte(jsonString), &user)
	if err != nil {
		utils.ColorError("Error unmarshalling jsonString", err)
		return ApiInfo{} // we shouldn't proceed if there is an error on processing jsonString
	}

	// Initialize the request body
	var body io.Reader

	// Handle GraphQL body if provided
	if user.Body != nil && user.Body.Graphql != nil && user.Body.Graphql.Query != "" {
		body, err = EncodeGraphQlBody(user.Body.Graphql.Query, user.Body.Graphql.Variables)
		if err != nil {
			utils.ColorError("Error encoding GraphQL body", err)
			return ApiInfo{}
		}
	}

	// Handle headers and their corresponding body types
	if user.Headers != nil && len(user.Headers) > 0 {
		for key, value := range user.Headers {
			switch strings.ToLower(key) {
			case "content-type":
				switch value {
				case "multipart/form-data":
					if user.Body != nil && user.Body.FormData != nil &&
						len(user.Body.FormData) > 0 {
						body, value, err = EncodeFormData(user.Body.FormData)
						if err != nil {
							utils.ColorError("Error encoding multipart form data", err)
							return ApiInfo{}
						}
					}
				case "application/x-www-form-urlencoded":
					if user.Body != nil && user.Body.UrlEncodedFormData != nil &&
						len(user.Body.UrlEncodedFormData) > 0 {
						body, err = EncodeXwwwFormUrlBody(user.Body.UrlEncodedFormData)
						if err != nil {
							utils.ColorError("Error encoding URL-encoded form data", err)
							return ApiInfo{}
						}
					}
				}
			}
		}
	}

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
