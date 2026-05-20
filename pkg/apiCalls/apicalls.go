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

	"github.com/xaaha/hulak/pkg/httpclient"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// DefaultClient is the production HTTP client.
// Uses httpclient.New() for shared redirect policy and TLS defaults.
var DefaultClient httpclient.HTTPClient = httpclient.New()

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
	client httpclient.HTTPClient,
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
		return CustomResponse{}, fmt.Errorf("error occurred on '%s': %w", method, err)
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
func SendAndSaveAPIRequest(ctx context.Context, secretsMap map[string]any, path string, debug bool) ([]byte, string, error) {
	apiConfig, _, err := yamlparser.FinalStructForAPI(
		path,
		secretsMap,
	)
	if err != nil {
		return nil, "", err
	}

	apiInfo, err := apiConfig.PrepareStruct()
	if err != nil {
		return nil, "", err
	}

	resp, err := StandardCall(ctx, apiInfo, debug)
	if err != nil {
		return nil, "", err
	}

	respBytes, saveErr := SerializeAndSaveResp(resp, path)
	status := ""
	if resp.Response != nil {
		status = resp.Response.Status
	}
	return respBytes, status, saveErr
}

// PrintAndSaveFinalResp prints the CustomResponse to stdout and saves it to
// disk next to the request file. Returns an error if the disk write fails so
// the caller can fail the task. A successful HTTP request with a missing
// response file should not look like a success to the user.
func PrintAndSaveFinalResp(resp CustomResponse, path string) error {
	respBytes, saveErr := SerializeAndSaveResp(resp, path)
	if respBytes != nil {
		PrintRespBytes(respBytes)
	}
	return saveErr
}

// SerializeAndSaveResp serializes the response, saves it to disk next to the
// request file, and returns the serialized bytes. It does NOT print to
// stdout. Use this when the caller needs to defer printing (e.g. when the
// response will be printed after a stderr spinner clears).
func SerializeAndSaveResp(resp CustomResponse, path string) ([]byte, error) {
	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		utils.PrintWarningStderr("serializing response: " + err.Error())
		// Fallback so the disk write still has something useful.
		strBody := fmt.Sprintf("%+v", resp)
		return []byte(strBody), evalAndWriteRes(strBody, resp.contentType, path)
	}
	strBody := string(jsonData)
	return jsonData, evalAndWriteRes(strBody, resp.contentType, path)
}

// PrintRespBytes prints serialized response JSON to stdout, colored when
// terminal supports it. Plain print on color failure so the user still sees
// the response body.
func PrintRespBytes(b []byte) {
	if err := utils.PrintJSONColored(b); err != nil {
		utils.PrintErrorStderr(err.Error())
		fmt.Println(string(b))
	}
}
