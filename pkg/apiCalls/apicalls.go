// Package apicalls has all things related to api call
package apicalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// StandardCall calls the api and returns the json body string
// Uses the DefaultClient for HTTP calls
func StandardCall(apiInfo yamlparser.ApiInfo, debug bool) (CustomResponse, error) {
	return StandardCallWithClient(apiInfo, debug, DefaultClient)
}

// StandardCallWithClient calls the api with a custom HTTP client and returns the json body string
// This function allows dependency injection for testing purposes
func StandardCallWithClient(apiInfo yamlparser.ApiInfo, debug bool, client HTTPClient) (CustomResponse, error) {
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
	preparedURL := PrepareURL(urlStr, apiInfo.UrlParams)

	reqBodyForDebug := make([]byte, len(bodyBytes))
	copy(reqBodyForDebug, bodyBytes)

	req, err := http.NewRequest(method, preparedURL, newBodyReader)
	if err != nil {
		return CustomResponse{}, fmt.Errorf("error occurred on '%s': %v", method, err)
	}

	if len(headers) > 0 {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}

	start := time.Now()

	response, err := client.Do(req)
	if err != nil {
		return CustomResponse{}, err
	}
	end := time.Now()

	duration := end.Sub(start)

	return processResponse(req, response, duration, debug, reqBodyForDebug), nil
}

// SendAndSaveAPIRequest calls the PrepareStruct using the provided envMap
// and makes the Api Call with StandardCall and prints the response in console
func SendAndSaveAPIRequest(secretsMap map[string]any, path string, debug bool) error {
	apiConfig, _, err := yamlparser.FinalStructForAPI(
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

	resp, err := StandardCall(apiInfo, debug)
	if err != nil {
		return err
	}

	PrintAndSaveFinalResp(resp, path)
	return nil
}

// PrintAndSaveFinalResp prints and saves the CustomResponse
func PrintAndSaveFinalResp(resp CustomResponse, path string) {
	jsonData, err := json.MarshalIndent(resp, "", "  ")

	var strBody string
	if err == nil {
		strBody = string(jsonData)
		err = utils.PrintJSONColored(jsonData)
		if err != nil {
			utils.PrintRed(string(err.Error()))
		}
	} else {
		utils.PrintWarning("error serializing response: " + err.Error())
		strBody = fmt.Sprintf("%+v", resp) // Fallback to raw struct
		fmt.Println(strBody)
	}

	if err := evalAndWriteRes(strBody, path); err != nil {
		utils.PrintRed("PrintAndSaveFinalResp " + err.Error())
	}
}
