package yamlParser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// From the yaml file, create a json file. But the json could have {{}} on it
// But we need to make sure those values are replaced. Then
//

// First, this generates the "envMap" from the .env files.
// Parses the user's input yaml file in json interface.
// Then, this function recursively replaces all variables {{.value}} specified in user's yaml values, with values from environment map
// This is necessary, as the all variables, like URL needs correct string
func replaceVarsWithValues(dict map[string]interface{}) map[string]interface{} {
	changedMap := make(map[string]interface{})

	envMap, err := envparser.GenerateSecretsMap()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	for key, val := range dict {
		switch valTyped := val.(type) {
		case map[string]interface{}:
			changedMap[key] = replaceVarsWithValues(valTyped)
		case string:
			finalChangedString, err := envparser.SubstituteVariables(valTyped, envMap)
			if err != nil {
				fmt.Println(err)
			}
			changedMap[key] = finalChangedString
		case map[string]string:
			innerMap := make(map[string]interface{})
			for k, v := range valTyped {
				finalChangedString, err := envparser.SubstituteVariables(v, envMap)
				if err != nil {
					fmt.Println(err)
				}
				innerMap[k] = finalChangedString
			}
			changedMap[key] = innerMap
		default:
			fmt.Println("Unexpected type:", val)
			changedMap[key] = val
		}
	}
	return changedMap
	/*
		// test this with
				test := map[string]interface{}{
						"first": "Pratik",
						"last":  "Thapa",
						"work":  map[string]interface{}{"position": "engineer"},
						"roles": map[string]string{"primary": "developer", "secondary": "designer"},
					}

					// Call ReplaceValues to process and print each value
					updatedTest := ReplaceValues(test)
					fmt.Println("Updated map:", updatedTest)
	*/
}

// Reads the yaml for http request.
// Right now, the yaml is only meant to hold http request as defined in the body struct in "./yamlTypes.go"
func handleYamlFile(filepath string) (*bytes.Buffer, error) {
	_, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		// utils.PanicRedAndExit("File does not exist, %s", filepath)
		panic("File does not exist " + filepath)
	}

	file, err := os.Open(filepath)

	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		utils.PanicRedAndExit("Empty yaml file")
	}

	if err != nil {
		utils.PanicRedAndExit("Error opening file: %v", err)
	}
	defer file.Close()
	var data map[string]interface{}
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		utils.PanicRedAndExit("error decoding data: %v", err)
	}

	data = utils.ConvertKeysToLowerCase(data)
	// TODO: parse all the values to with {{.key}}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		utils.PanicRedAndExit("error encoding data: %v", err)
	}
	enc.Close()

	return &buf, nil
}

// checks the validity of all the fields in the yaml file
// and returns the json string of the yaml file
func ReadYamlForHttpRequest(filePath string) string {
	buf, err := handleYamlFile(filePath)
	if err != nil {
		utils.ColorError("Error occured after reading yaml file", err)
	}

	var user User
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&user); err != nil {
		utils.PanicRedAndExit("error decoding data: %v", err)
	}

	// uppercase and type conversion
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(user.Method)))
	user.Method = upperCasedMethod

	// method is required for any http request
	if !user.Method.IsValid() {
		utils.PanicRedAndExit("missing or invalid HTTP method: %s", user.Method)
	}

	// url is required for any http request
	if !user.Url.IsValidURL() {
		utils.PanicRedAndExit("missing or invalid URL: %s", user.Url)
	}

	// check if body is valid
	if user.Body == nil {
		utils.PanicRedAndExit("Body is missing in the YAML file. Please add a valid Body.")
	} else if !user.Body.IsValid() {
		utils.PanicRedAndExit(
			"Invalid Body. Make sure body contains only one valid argument.\n %v",
			user.Body,
		)
	}
	val, _ := json.MarshalIndent(user, "", "  ")
	jsonString := string(val)
	return jsonString
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
