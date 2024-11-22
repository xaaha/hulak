package apicalls

// reads the json file based on the user's flag
// if the flag is absent, it panics.
// Finally, it uses json from yaml with StandardCall
func combineAndCall(jsonString string) {
	// unmarhall the json
	// need to return struct ApiInfo for StandardCall
	// i need to think this through
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
