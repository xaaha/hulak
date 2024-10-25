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

// used on url paramaters and request body's form data
type KeyValuePair struct {
	Key   string
	Value string
}

// if the url has parameters, the function perpares and returns the full url otherwise,
// the function returns the provided baseUrl
func PrepareUrl(baseUrl string, params ...KeyValuePair) string {
	u, err := url.Parse(baseUrl)
	if err != nil {
		// If parsing fails, return the base URL as is
		return baseUrl
	}

	// Prepare URL query parameters
	queryParams := url.Values{}
	for _, param := range params {
		queryParams.Add(param.Key, param.Value)
	}
	// If there are parameters, encode them and append to the base URL
	if len(params) > 0 {
		u.RawQuery = queryParams.Encode()
	}
	return u.String()
}

// types of body that could be passed
type Body struct {
	RawString          string
	FormData           []KeyValuePair
	UrlEncodedFormData []KeyValuePair
	// Graphql
	// binary
}

// Encodes key-value pairs as "application/x-www-form-urlencoded" data.
// Returns an io.Reader containing the encoded data, or an error if the input is empty.
func EncodeXwwwFormUrlBody(keyValue []KeyValuePair) (io.Reader, error) {
	// Initialize form data
	formData := url.Values{}

	// Populate form data, using Set to overwrite duplicate keys if any
	for _, kv := range keyValue {
		if kv.Key != "" && kv.Value != "" {
			formData.Set(kv.Key, kv.Value)
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
func EncodeFormData(keyValue []KeyValuePair) (io.Reader, string, error) {
	if len(keyValue) == 0 {
		return nil, "", fmt.Errorf("no key-value pairs to encode")
	}

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	defer writer.Close() // Ensure writer is closed

	for _, kv := range keyValue {
		if kv.Key != "" && kv.Value != "" {
			if err := writer.WriteField(kv.Key, kv.Value); err != nil {
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

type ApiInfo struct {
	Method  string
	Url     string
	Body    io.Reader
	Headers []KeyValuePair
}

func StandardCall(apiInfo ApiInfo) string {
	if apiInfo.Headers == nil {
		apiInfo.Headers = []KeyValuePair{}
	}
	method := apiInfo.Method
	url := apiInfo.Url
	body := apiInfo.Body
	headers := apiInfo.Headers
	errMessage := "error occured on " + method

	preparedUrl := PrepareUrl(url)

	// handle different case for body. EncodeBodyFormData when x-www-form-urlencoded
	// multiple lines.
	// single line raw body
	// graphql query and others
	// json and other
	// always check StandardCall() and close barnch

	req, err := http.NewRequest(method, preparedUrl, body)
	if err != nil {
		log.Fatalln(errMessage, err)
	}

	if len(headers) > 0 {
		for _, header := range headers {
			req.Header.Add(header.Key, header.Value)
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

// sample get call from https://blog.logrocket.com/making-http-requests-in-go/
// when the request is just with a url
func Get() {
	resp, err := http.Get("https://jsonplaceholder.typicode.com/posts")
	if err != nil {
		log.Fatalln(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// Convert the body to type string
	sb := string(body)
	log.Print(sb)
}

func CallUrlEncodedForm() {
	method := "POST"
	// first prepare the baseurl
	baseurl := ""

	// Prepare query parameters (x-www-form-urlencoded)
	formData := url.Values{}
	formData.Set("grant_type", "http://auth0.com/oauth/grant-type/password-realm")
	formData.Set("username", "")
	formData.Set("password", "")
	formData.Set("audience", "")
	formData.Set("scope", "openid")
	formData.Set("client_id", "")
	formData.Set("realm", "")

	// strings.NewReader(data.Encode())
	req, err := http.NewRequest(method, baseurl, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Fatalln(err)
	}

	// Add the Authorization token and other headers as needed
	// req.Header.Add("Authorization", "Bearer YOUR_AUTH_TOKEN")

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// Convert the body to type string
	fmt.Println(string(body))
}
