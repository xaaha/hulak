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
func StandardCall(apiInfo ApiInfo) string {
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
