package yamlParser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

// handle situation when a necessary component url, method is missing from yaml

// reads the yaml for http request.
// right now, the yaml is only meant to hold http request
func ReadYamlForHttpRequest() {
	file, err := os.Open("test_collection/user.yaml")
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	var user User
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&user); err != nil {
		log.Fatalf("error decoding data: %v", err)
	}

	// uppercase and type conversion
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(user.Method)))
	user.Method = upperCasedMethod

	// method is required for any http request
	if !user.Method.IsValid() {
		log.Fatalf("invalid HTTP method: %s", user.Method)
	}

	// url is required for any http request
	if !user.Url.IsValidURL() {
		log.Fatalf("invalid URL: %s", user.Url)
	}

	// check body is valid
	if !user.Body.IsValid() {
		log.Fatalf("invalid body: %v", *user.Body)
	}

	val, _ := json.MarshalIndent(user, "", "  ")

	fmt.Println(string(val))
	// fmt.Printf("Name: %s, Age: %v, Email: %s", user.Name, user.Age, user.Email)
}

func ReadingYamlWithoutStruct() {
	file, err := os.Open("test_collection/test.yml")
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	var data map[string]interface{}
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		log.Fatalf("error decoding data: %v", err)
	}

	val, _ := json.MarshalIndent(data, "", "  ")
	// log prints time, which I don't need
	fmt.Println(string(val))

	// for key, value := range data {
	// 	fmt.Printf("%s: %v\n", key, value)
	// }
}

/*
Example Usage:
import (
ymlReader "github.com/xaaha/hulak/pkg/hulak_yaml_reader"
)
// use it like
ymlReader.ReadingYamlWithStruct()
ymlReader.ReadingYamlWithoutStruct()
*/
