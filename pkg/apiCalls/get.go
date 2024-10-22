package apicalls

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

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

	// Prepare query parameters
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
