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
func StandardCall(
	ctx context.Context,
	apiInfo yamlparser.APIInfo,
	debug bool,
) (CustomResponse, error) {
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

// SendAndSaveAPIRequest builds the API request from the file at opts.Path,
// then either executes it or (when opts.DryRun) prints the built request.
//
// Returns the HTTP status string (e.g. "200 OK") so the runner can render
// a per-file outcome line. On pre-flight failures (config parse, request
// build) the status is empty. On transport failures (network down, DNS) the
// status is also empty — the err carries the detail.
//
// When opts.DryRun is true, the request is built and printed to stdout but
// never sent. No response file is written. opts.Show controls whether
// sensitive headers are revealed in the printed output.
func SendAndSaveAPIRequest(ctx context.Context, opts RequestOptions) ([]byte, string, error) {
	apiConfig, _, err := yamlparser.FinalStructForAPI(opts.Path, opts.Secrets)
	if err != nil {
		return nil, "", err
	}

	apiInfo, err := apiConfig.PrepareStruct()
	if err != nil {
		return nil, "", err
	}

	if opts.DryRun {
		if err := PrintDryRun(&apiInfo, opts.Show); err != nil {
			return nil, "", err
		}
		return nil, "", nil
	}

	resp, err := StandardCall(ctx, apiInfo, opts.Debug)
	if err != nil {
		return nil, "", err
	}

	status := ""
	if resp.Response != nil {
		status = resp.Response.Status
	}

	if opts.NoSave {
		return SerializeResp(&resp), status, nil
	}

	respBytes, saveErr := SerializeAndSaveResp(&resp, opts.Path)
	return respBytes, status, saveErr
}

// PrintAndSaveFinalResp prints the CustomResponse to stdout and saves it to
// disk next to the request file. Returns an error if the disk write fails so
// the caller can fail the task. A successful HTTP request with a missing
// response file should not look like a success to the user.
func PrintAndSaveFinalResp(resp *CustomResponse, path string) error {
	respBytes, saveErr := SerializeAndSaveResp(resp, path)
	if respBytes != nil {
		PrintRespBytes(respBytes)
	}
	return saveErr
}

// SerializeResp returns the response bytes used for output and saving,
// without touching disk. Callers that only want the response in-hand (e.g.
// the MCP call_request tool with NoSave) use this directly.
//
// Default: raw response body (JSON pretty-printed, others byte-perfect).
// --debug: full CustomResponse marshaled as JSON, falling back to a %+v dump
// if marshaling fails so the caller still gets something useful.
func SerializeResp(resp *CustomResponse) []byte {
	if resp.isDebug() {
		jsonData, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			utils.PrintWarningStderr("serializing response: " + err.Error())
			return []byte(fmt.Sprintf("%+v", resp))
		}
		return jsonData
	}
	return defaultBodyForOutput(resp.rawBody)
}

// SerializeAndSaveResp writes the response to disk next to the request
// file and returns the bytes. Does not print.
//
// Default: raw response body (JSON pretty-printed, others byte-perfect).
// --debug: full CustomResponse (request, response, http_info, duration).
func SerializeAndSaveResp(resp *CustomResponse, path string) ([]byte, error) {
	body := SerializeResp(resp)
	if len(body) == 0 {
		// 204 No Content and friends: nothing to print or save.
		return body, nil
	}
	return body, evalAndWriteRes(string(body), resp.contentType, path)
}

// defaultBodyForOutput returns the bytes used for default-mode save and
// print: raw bytes verbatim, but pretty-printed when the payload is valid
// JSON so the on-disk file is readable. Falls back to the original bytes
// when json.Indent fails so non-JSON content (HTML, XML, plain text, binary)
// is preserved exactly.
func defaultBodyForOutput(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	if !IsJSON(string(raw)) {
		return raw
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return raw
	}
	return pretty.Bytes()
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
