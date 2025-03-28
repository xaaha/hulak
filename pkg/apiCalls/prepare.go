// Package apicalls has all things related to api call
package apicalls

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// PrepareURL perpares and returns the full url.
// If the url has parameters, then the function returns the provided baseUrl
func PrepareURL(baseURL string, urlParams map[string]string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		// If parsing fails, return the base URL as is
		return baseURL
	}
	// Prepare URL query parameters if params are provided
	if urlParams != nil {
		queryParams := url.Values{}
		for key, val := range urlParams {
			queryParams.Add(key, val)
		}
		u.RawQuery = queryParams.Encode()
	}
	return u.String()
}

// Takes in http response, adds response status code,
// and returns CustomResponse type string for the StandardCall function below
func processResponse(response *http.Response) CustomResponse {
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("prepare.go: Error while reading response: %v", err)
	}

	responseData := CustomResponse{
		ResponseStatus: fmt.Sprintf(
			"%d %s",
			response.StatusCode,
			http.StatusText(response.StatusCode),
		),
	}
	var parsedBody any
	if err := json.Unmarshal(respBody, &parsedBody); err == nil {
		responseData.Body = parsedBody
	} else {
		// If the body isn't valid JSON, include it as a string
		responseData.Body = string(respBody)
	}

	return responseData
}
