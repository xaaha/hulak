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

// example
func ReadingYamlWithStruct() {
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

	// type conversion
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(user.Method)))
	user.Method = upperCasedMethod

	if !user.Method.IsValid() {
		log.Fatalf("invalid HTTP method: %s", user.Method)
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
