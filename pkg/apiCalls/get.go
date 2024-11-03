package apicalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
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

// Encodes key-value pairs as "application/x-www-form-urlencoded" data.
// Returns an io.Reader containing the encoded data, or an error if the input is empty.
func EncodeXwwwFormUrlBody(keyValue map[string]string) (io.Reader, error) {
	// Initialize form data
	formData := url.Values{}

	// Populate form data, using Set to overwrite duplicate keys if any
	for key, val := range keyValue {
		if key != "" && val != "" {
			formData.Set(key, val)
		}
	}

	// Return an error if no valid key-value pairs were found
	if len(formData) == 0 {
		return nil, fmt.Errorf("no valid key-value pairs to encode")
	}

	// Encode form data to "x-www-form-urlencoded" format
	return strings.NewReader(formData.Encode()), nil
}

// Encodes form data other than x-www-form-urlencoded,
// Returns the payload, Content-Type for the headers and error
func EncodeFormData(keyValue map[string]string) (io.Reader, string, error) {
	if len(keyValue) == 0 {
		return nil, "", fmt.Errorf("no key-value pairs to encode")
	}

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	defer writer.Close() // Ensure writer is closed

	for key, val := range keyValue {
		if key != "" && val != "" {
			if err := writer.WriteField(key, val); err != nil {
				return nil, "", err
			}
		}
	}

	// Return the payload and the content type for the header
	return payload, writer.FormDataContentType(), nil
}

// accepts query string and variables map[string]interface, then returns the payload
func EncodeGraphQlBody(query string, variables map[string]interface{}) (io.Reader, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(jsonData), nil
}

// struct for StandardCall
type ApiInfo struct {
	Body    io.Reader
	Headers map[string]string
	Method  string
	Url     string
}

// Makes an api call and resturns the json body string
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
	defer response.Body.Close()

	jsonBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return string(jsonBody)
}
