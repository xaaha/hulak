package yamlParser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
)

// handle situation when a necessary component url, method is missing from yaml

// reads the yaml for http request.
// right now, the yaml is only meant to hold http request as defined in the body struct in "./yamlTypes.go"
func ReadYamlForHttpRequest() {
	file, err := os.Open("test_collection/user.yaml")
	if err != nil {
		utils.PanicRedAndExit("error opening file: %v", err)
	}
	defer file.Close()

	var user User
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&user); err != nil {
		utils.PanicRedAndExit("error decoding data: %v", err)
	}

	// uppercase and type conversion
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(user.Method)))
	user.Method = upperCasedMethod

	// method is required for any http request
	if !user.Method.IsValid() {
		utils.PanicRedAndExit("invalid HTTP method: %s", user.Method)
	}

	// url is required for any http request
	if !user.Url.IsValidURL() {
		utils.PanicRedAndExit("invalid URL: %s", user.Url)
	}

	// check body is valid
	if !user.Body.IsValid() {
		utils.PanicRedAndExit(
			"Invalid Body. Make sure body contains only one valid arguments \n %v",
			*user.Body,
		)
	}

	val, _ := json.MarshalIndent(user, "", "  ")

	fmt.Println(string(val))
	// fmt.Printf("Name: %s, Age: %v, Email: %s", user.Name, user.Age, user.Email)
}

func ReadingYamlWithoutStruct() {
	file, err := os.Open("test_collection/test.yml")
	if err != nil {
		utils.PanicRedAndExit("Error opening file: %v", err)
	}
	defer file.Close()

	var data map[string]interface{}
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		utils.PanicRedAndExit("error decoding data: %v", err)
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
