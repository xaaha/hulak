package apicalls

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

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

	// Initialize the request body
	var body io.Reader

	// Handle GraphQL body if provided
	hasGraphQlBody := user.Body != nil && user.Body.Graphql != nil &&
		len(user.Body.Graphql.Query) > 0
	if hasGraphQlBody {
		body, err = EncodeGraphQlBody(user.Body.Graphql.Query, user.Body.Graphql.Variables)
		if err != nil {
			msg := "call.go: Error encoding GraphQL body of json\n" + jsonString
			err := utils.ColorError(msg, err)
			return ApiInfo{}, err
		}
	}
	var contentType string
	if user.Body != nil && user.Body.FormData != nil &&
		len(user.Body.FormData) > 0 {
		body, contentType, err = EncodeFormData(user.Body.FormData)
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
		body, err = EncodeXwwwFormUrlBody(user.Body.UrlEncodedFormData)
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

// Using the provided envMap, this function calls the PrepareStruct,
// and Makes the Api Call with StandardCall and prints the response in console
// TODO: Flag to disable and silence the std output and file save
func SendAndSaveApiRequest(secretsMap map[string]interface{}, path string) {
	formDataJSONString := yamlParser.FinalJsonForHttpRequest(
		path,
		secretsMap,
	)
	apiInfo, err := PrepareStruct(formDataJSONString)
	if err != nil {
		err := utils.ColorError("call.go: error occured while preparing Struct from "+path, err)
		utils.PrintRed(err.Error())
		return
	}
	resp := StandardCall(apiInfo)
	if resp == "" {
		utils.PrintRed("call.go: StandardCall returned an empty response")
		return
	}

	var customResponse CustomResponse
	if err := json.Unmarshal([]byte(resp), &customResponse); err != nil {
		utils.PrintWarning("call.go: error while saving the response to a file" + err.Error())
		return
	}

	// save the response file
	var strBody string
	// by default save the response body
	// if err, save the entire resp with status code
	switch body := customResponse.Body.(type) {
	case string:
		strBody = body
	case map[string]interface{}, []interface{}:
		jsonData, err := json.Marshal(body)
		if err != nil {
			strBody = resp
		} else {
			strBody = string(jsonData)
		}
	default:
		strBody = resp
	}

	err = evalAndWriteRes(strBody, path)
	if err != nil {
		utils.PrintRed("call.go: " + err.Error())
	}

	fmt.Println(resp)
}
