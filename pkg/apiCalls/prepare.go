package apicalls

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// if the url has parameters, the function perpares and returns the full url otherwise,
// the function returns the provided baseUrl
func PrepareUrl(baseUrl string, urlParams map[string]string) string {
	u, err := url.Parse(baseUrl)
	if err != nil {
		// If parsing fails, return the base URL as is
		return baseUrl
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
	var parsedBody interface{}
	if err := json.Unmarshal(respBody, &parsedBody); err == nil {
		responseData.Body = parsedBody
	} else {
		// If the body isn't valid JSON, include it as a string
		responseData.Body = string(respBody)
	}

	return responseData
}
