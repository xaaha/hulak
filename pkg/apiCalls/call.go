package apicalls

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// Makes an api call and returns the json body string
func StandardCall(apiInfo ApiInfo) CustomResponse {
	if apiInfo.Headers == nil {
		apiInfo.Headers = map[string]string{}
	}
	method := apiInfo.Method
	url := apiInfo.Url
	body := apiInfo.Body
	headers := apiInfo.Headers
	urlParams := map[string]string{}
	errMessage := "error occured on " + method

	preparedUrl := PrepareUrl(url, urlParams)

	req, err := http.NewRequest(method, preparedUrl, body)
	if err != nil {
		log.Fatalln(errMessage, err)
	}

	if len(headers) > 0 {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalln(errMessage, err)
	}
	return processResponse(response)
}

// Using the provided envMap, this function calls the PrepareStruct,
// and Makes the Api Call with StandardCall and prints the response in console
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
	PrintAndSaveFinalResp(resp, path)
}

// Prints and Save the custom response
// TODO: Flag to disable and silence the std output and file save
func PrintAndSaveFinalResp(resp CustomResponse, path string) {
	var strBody string
	switch body := resp.Body.(type) {
	case string:
		strBody = body
	case map[string]interface{}, []interface{}:
		if jsonData, err := json.MarshalIndent(body, "", "  "); err == nil {
			strBody = string(jsonData)
		} else {
			utils.PrintWarning("call.go: error serializing response body: " + err.Error())
			strBody = fmt.Sprintf("%+v", resp) // Fallback to entire response
		}
	default:
		strBody = fmt.Sprintf("%+v", resp) // Fallback for unsupported types
	}

	err := evalAndWriteRes(strBody, path)
	if err != nil {
		utils.PrintRed("call.go: " + err.Error())
	}

	fmt.Println(strBody)
}
