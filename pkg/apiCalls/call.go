package apicalls

import (
	"encoding/json"
	"io"

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
	// handle the rest of the situation

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
