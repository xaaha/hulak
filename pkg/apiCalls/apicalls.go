// Package apicalls has all things related to api call
package apicalls

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// HTTPClient interface allows mocking HTTP calls in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultClient is the production HTTP client
var DefaultClient HTTPClient = &http.Client{}

// StandardCall calls the api and returns the json body string
// Uses the DefaultClient for HTTP calls
func StandardCall(ctx context.Context, apiInfo yamlparser.APIInfo, debug bool) (CustomResponse, error) {
	return StandardCallWithClient(ctx, apiInfo, debug, DefaultClient)
}

// StandardCallWithClient calls the api with a custom HTTP client and returns the json body string
// This function allows dependency injection for testing purposes
func StandardCallWithClient(
	ctx context.Context,
	apiInfo yamlparser.APIInfo,
	debug bool,
	client HTTPClient,
) (CustomResponse, error) {
	if apiInfo.Headers == nil {
		apiInfo.Headers = map[string]string{}
	}
	method := apiInfo.Method
	urlStr := apiInfo.URL
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
	preparedURL := PrepareURL(urlStr, apiInfo.URLParams)

	reqBodyForDebug := make([]byte, len(bodyBytes))
	copy(reqBodyForDebug, bodyBytes)

	req, err := http.NewRequestWithContext(ctx, method, preparedURL, newBodyReader)
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

	return processResponse(req, response, duration, debug, reqBodyForDebug)
}

// SendAndSaveAPIRequest calls the PrepareStruct using the provided envMap
// and makes the Api Call with StandardCall and prints the response in console.
//
// Returns the HTTP status string (e.g. "200 OK") so the runner can render
// a per-file outcome line. On pre-flight failures (config parse, request
// build) the status is empty. On transport failures (network down, DNS) the
// status is also empty — the err carries the detail.
func SendAndSaveAPIRequest(ctx context.Context, secretsMap map[string]any, path string, debug bool) (string, error) {
	apiConfig, _, err := yamlparser.FinalStructForAPI(
		path,
		secretsMap,
	)
	if err != nil {
		return "", err
	}

	apiInfo, err := apiConfig.PrepareStruct()
	if err != nil {
		return "", err
	}

	resp, err := StandardCall(ctx, apiInfo, debug)
	if err != nil {
		return "", err
	}

	saveErr := PrintAndSaveFinalResp(resp, path)
	status := ""
	if resp.Response != nil {
		status = resp.Response.Status
	}
	return status, saveErr
}

// PrintAndSaveFinalResp prints the CustomResponse to stdout and saves it to
// disk next to the request file. Returns an error if the disk write fails so
// the caller can fail the task — a successful HTTP request with a missing
// response file should not look like a success to the user.
func PrintAndSaveFinalResp(resp CustomResponse, path string) error {
	jsonData, err := json.MarshalIndent(resp, "", "  ")

	var strBody string
	if err == nil {
		strBody = string(jsonData)
		err = utils.PrintJSONColored(jsonData)
		if err != nil {
			utils.PrintErrorStderr(err.Error())
		}
	} else {
		utils.PrintWarningStderr("serializing response: " + err.Error())
		strBody = fmt.Sprintf("%+v", resp) // Fallback to raw struct
		fmt.Println(strBody)
	}

	return evalAndWriteRes(strBody, resp.contentType, path)
}
