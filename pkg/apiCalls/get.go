package apicalls

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Headers struct {
	key   string
	value string
}

func StandardCall(method, url string, body io.Reader, headers ...Headers) {
	errMessage := "error occured during" + method + "call"

	// Function to handle urls (base and params)
	// when the method has x-www-form-urlencoded, body is the  strings.NewReader(formData.Encode())
	// body should be string. If multiple lines use `` otherwise ""
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Fatalln(errMessage, err)
	}

	if len(headers) > 0 {
		for _, header := range headers {
			req.Header.Add(header.key, header.value)
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

	fmt.Println(string(jsonBody))
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
