// Package apicalls has all things related to api call
package apicalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// StandardCall calls the api and returns the json body string
func StandardCall(apiInfo yamlparser.ApiInfo) (CustomResponse, error) {
	if apiInfo.Headers == nil {
		apiInfo.Headers = map[string]string{}
	}
	method := apiInfo.Method
	urlStr := apiInfo.Url
	bodyReader := apiInfo.Body

	var bodyBytes []byte
	var err error
	if bodyReader != nil {
		bodyBytes, err = io.ReadAll(bodyReader)
		if err != nil {
			return CustomResponse{}, err
		}
	} else {
		bodyBytes = []byte{}
	}

	newBodyReader := bytes.NewReader(bodyBytes)
	headers := apiInfo.Headers
	urlParams := map[string]string{}
	preparedURL := PrepareURL(urlStr, urlParams)

	req, err := http.NewRequest(method, preparedURL, newBodyReader)
	if err != nil {
		return CustomResponse{}, fmt.Errorf("error occurred on '%s': %v", method, err)
	}

	if len(headers) > 0 {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return CustomResponse{}, err
	}
	return processResponse(req, response), nil
}

// SendAndSaveAPIRequest calls the PrepareStruct using the provided envMap
// and makes the Api Call with StandardCall and prints the response in console
func SendAndSaveAPIRequest(secretsMap map[string]any, path string) error {
	apiConfig, err := yamlparser.FinalStructForAPI(
		path,
		secretsMap,
	)
	if err != nil {
		return err
	}

	apiInfo, err := apiConfig.PrepareStruct()
	if err != nil {
		return err
	}

	resp, err := StandardCall(apiInfo)
	if err != nil {
		return err
	}

	PrintAndSaveFinalResp(resp, path)
	return nil
}

// PrintAndSaveFinalResp prints and saves the CustomResponse
func PrintAndSaveFinalResp(resp CustomResponse, path string) {
	var strBody string

	// Marshal the CustomResponse structure
	if jsonData, err := json.MarshalIndent(resp, "", "  "); err == nil {
		strBody = string(jsonData)
	} else {
		utils.PrintWarning("call.go: error serializing response: " + err.Error())
		strBody = fmt.Sprintf("%+v", resp) // Fallback to entire response
	}

	err := evalAndWriteRes(strBody, path)
	if err != nil {
		utils.PrintRed("call.go: " + err.Error())
	}

	fmt.Println(strBody)
}
