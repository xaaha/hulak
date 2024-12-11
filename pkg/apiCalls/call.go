package apicalls

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"

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
// TODO: save a json response on the path as well
func SendApiRequest(envMap map[string]string, filePath string) {
	formDataJSONString := yamlParser.ReadYamlForHttpRequest(
		filePath,
		envMap,
	)
	apiInfo, err := PrepareStruct(formDataJSONString)
	if err != nil {
		err := utils.ColorError("call.go: error occured while preparing Struct from "+filePath, err)
		fmt.Println(err)
	}
	resp := StandardCall(apiInfo)
	// but the file could be html, xml, just plain text, json or any other type. We can't simply write in json
	// err = os.WriteFile("test.json", []byte(resp), 0644)
	// if err != nil {
	// 	utils.PrintRed(err.Error())
	// }
	fmt.Println(resp)
}

func isJson(str string) bool {
	var jsBfr json.RawMessage
	return json.Unmarshal([]byte(str), &jsBfr) == nil
}

func isXML(str string) bool {
	var v interface{}
	return xml.Unmarshal([]byte(str), &v) == nil
}

func isHTML(str string) bool {
	_, err := html.Parse(strings.NewReader(str))
	return err == nil
}

func writeFile(fileName, suffixType, contentBody string) {
	err := os.WriteFile(fileName+suffixType, []byte(contentBody), 0644)
	if err != nil {
		utils.PrintRed("call.go: error while saving file \n" + err.Error())
	}
}

// Checks whether the response is of certain type, json, xml, html or text.
// Based on the evaluation, it writes the response to the provided path in respective file path
func EvalAndWriteRes(resBody, path, fileName string) {
	if isJson(resBody) {
		writeFile(fileName, ".json", resBody)
	} else if isXML(resBody) {
		writeFile(fileName, ".xml", resBody)
	} else if isHTML(resBody) {
		writeFile(fileName, ".html", resBody)
	} else {
		writeFile(fileName, ".txt", resBody)
	}
}
