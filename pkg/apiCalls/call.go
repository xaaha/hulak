package apicalls

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

// reads the json file based on the user's flag
// if the flag is absent, it panics.
// Finally, it uses the json from yaml with StandardCall
func CombineAndCall(jsonString string) ApiInfo {
	var user yamlParser.User
	err := json.Unmarshal([]byte(jsonString), &user)
	if err != nil {
		message := "Error unmarshalling jsonString " + err.Error()
		utils.PrintRed(message)
	}
	// prepare the user's body
	var body io.Reader
	if user.Body == nil {
		body = nil
	}

	if user.Body != nil && user.Body.Graphql != nil && user.Body.Graphql.Query != "" {
		body, err = EncodeGraphQlBody(user.Body.Graphql.Query, user.Body.Graphql.Variables)
		if err != nil {
			utils.ColorError("Call.go: Error while encoding graphql body", err)
		}
	}

	// var formDatacontentType string. // use this to get the content-type for form data

	if user.Headers != nil && len(user.Headers) > 0 {
		headerMap := user.Headers
		for key, value := range headerMap {
			if strings.ToLower(key) == "content-type" {
				if value == "multipart/form-data" {
					if user.Body != nil && user.Body.FormData != nil &&
						len(user.Body.FormData) > 0 {
						body, value, err = EncodeFormData(user.Body.FormData)
						if err != nil {
							utils.ColorError("Check your header type and body of FormData", err)
						}
					}
				}
				if value == "application/x-www-form-urlencoded" {
					if user.Body != nil {
						EncodeXwwwFormUrlBody(user.Body.UrlEncodedFormData)
					}
				}

			}
		}
	}

	// if user.Body != nil && user.Headers
	// if the user has header of form-data, then formData otherwise it's x-form-urlencoded
	// TODO handle the rest of the situation for body.... Raw could be xml, json, html.
	// if the body is of type string, or urlEncoded, or x-form-urlencoded
	// Does string handles everthing

	data := ApiInfo{
		Method: string(user.Method),
		Url:    string(user.Url),
		Body:   body,
	}
	return data
}

// need to Unmarshal json string
/*
package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	jsonString := `{"name": "John", "age": 30, "city": "New York"}`

	// Create a struct to hold the JSON data
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
		City string `json:"city"`
	}

	// Unmarshal the JSON string into the struct
	var person Person
	err := json.Unmarshal([]byte(jsonString), &person)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// Access the parsed data
	fmt.Println("Name:", person.Name)
	fmt.Println("Age:", person.Age)
	fmt.Println("City:", person.City)
}
*/
