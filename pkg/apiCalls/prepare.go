package apicalls

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// if the url has parameters, the function perpares and returns the full url otherwise,
// the function returns the provided baseUrl
func PrepareUrl(baseUrl string, urlParams map[string]string) string {
	u, err := url.Parse(baseUrl)
	if err != nil {
		// If parsing fails, return the base URL as is
		return baseUrl
	}
	// Prepare URL query parameters if params are provided
	if urlParams != nil {
		queryParams := url.Values{}
		for key, val := range urlParams {
			queryParams.Add(key, val)
		}
		u.RawQuery = queryParams.Encode()
	}
	return u.String()
}

// Takes in the jsonString from ReadYamlForHttpRequest
// And prepares the ApiInfo struct for StandardCall function
func PrepareStruct(jsonString string) (ApiInfo, error) {
	// ReadYamlForHttpRequest should not return an empty string
	// but if it does, return nil struct
	if len(jsonString) == 0 {
		err := utils.ColorError("call.go: jsonString constructed from yamlFile is empty")
		return ApiInfo{}, err
	}
	// Parse the JSON string into a User struct
	var user yamlParser.User
	err := json.Unmarshal([]byte(jsonString), &user)
	if err != nil {
		msg := "call.go: Error unmarshalling json \n" + jsonString
		err := utils.ColorError(msg, err)
		return ApiInfo{}, err // we shouldn't proceed if there is an error on processing jsonString
	}

	var body io.Reader

	// Handle GraphQL body if provided
	hasGraphQlBody := user.Body != nil && user.Body.Graphql != nil &&
		len(user.Body.Graphql.Query) > 0
	if hasGraphQlBody {
		body, err = yamlParser.EncodeGraphQlBody(
			user.Body.Graphql.Query,
			user.Body.Graphql.Variables,
		)
		if err != nil {
			msg := "call.go: Error encoding GraphQL body of json\n" + jsonString
			err := utils.ColorError(msg, err)
			return ApiInfo{}, err
		}
	}
	var contentType string
	if user.Body != nil && user.Body.FormData != nil &&
		len(user.Body.FormData) > 0 {
		body, contentType, err = yamlParser.EncodeFormData(user.Body.FormData)
		if user.Headers == nil {
			user.Headers = make(map[string]string)
		}
		user.Headers["content-type"] = contentType
		if err != nil {
			err := utils.ColorError("call.go: Error encoding multipart form data", err)
			return ApiInfo{}, err
		}
	}

	if user.Body != nil && user.Body.UrlEncodedFormData != nil &&
		len(user.Body.UrlEncodedFormData) > 0 {
		body, err = yamlParser.EncodeXwwwFormUrlBody(user.Body.UrlEncodedFormData)
		if user.Headers == nil {
			user.Headers = make(map[string]string)
		}
		user.Headers["content-type"] = "application/x-www-form-urlencoded"
		if err != nil {
			err := utils.ColorError("call.go: Error encoding URL-encoded form data", err)
			return ApiInfo{}, err
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
	}, nil
}

// Takes in http response, adds response status code,
// and returns CustomResponse type string for the StandardCall function below
func processResponse(response *http.Response) CustomResponse {
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("prepare.go: Error while reading response %v", err)
	}
	defer response.Body.Close()

	responseData := CustomResponse{
		ResponseStatus: response.Status,
	}
	var parsedBody interface{}
	if err := json.Unmarshal(respBody, &parsedBody); err == nil {
		responseData.Body = parsedBody
	} else {
		// If the body isn't valid JSON, include it as a string
		responseData.Body = string(respBody)
	}

	return responseData
}
