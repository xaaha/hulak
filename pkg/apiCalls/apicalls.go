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

	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return CustomResponse{}, err
	}

	newBodyReader := bytes.NewReader(bodyBytes)
	headers := apiInfo.Headers
	urlParams := map[string]string{}
	preparedURL := PrepareUrl(urlStr, urlParams)

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
	return processResponse(response), nil
}

// Using the provided envMap, this function calls the PrepareStruct,
// and Makes the Api Call with StandardCall and prints the response in console
func SendAndSaveApiRequest(secretsMap map[string]any, path string) error {
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

// Prints and Save the custom response
func PrintAndSaveFinalResp(resp CustomResponse, path string) {
	// Create a combined structure to store both body and status
	combined := struct {
		Body   any    `json:"body"`
		Status string `json:"status"`
	}{
		Body:   resp.Body,
		Status: resp.ResponseStatus,
	}

	var strBody string
	// Marshal the combined structure
	if jsonData, err := json.MarshalIndent(combined, "", "  "); err == nil {
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
