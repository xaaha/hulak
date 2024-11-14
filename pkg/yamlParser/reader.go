package yamlParser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
)

// Recursively change the value
// func substituteEnvVarsInMapValues(dict map[string]string) map[string]interface{} {
// 	finalDict := make(map[string]interface{})
// 	envMap, err := envparser.GenerateSecretsMap()
// 	if err != nil {
// 		panic(err)
// 	}
// 	for key, val := range finalDict {
// 		switch changedStringVal := val.(type) {
// 		case map[string]interface{}:
// 			fmt.Println("handle interface")
// 		case string:
// 			finalStr, err := envparser.SubstitueVariables(val, envMap)
// 			if err != nil {
// 				fmt.Println(err)
// 			}
// 		}
// 	}
//
// 	// print entire json
// 	niceJson, _ := json.MarshalIndent(envMap, "", "  ")
// 	fmt.Println(string(niceJson))
//
// 	// how to substitute variable
// 	finalAns, err := envparser.SubstitueVariables("env{{PORT}}", envMap)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	fmt.Println(finalAns)
// }

// TODO: Complete this, replace actual values from envmap intead of printing them
// Check Replaces all values for all the values provided in the map, this function rep
func ReplaceValues(dict map[string]interface{}) map[string]interface{} {
	// Initialize a new map to store modified values
	changedMap := make(map[string]interface{})

	// Iterate over each key-value pair in the input map
	for key, val := range dict {
		switch valTyped := val.(type) {
		// If the value is another map, recursively call ReplaceValues
		case map[string]interface{}:
			changedMap[key] = ReplaceValues(valTyped)
		// If the value is a string, print and add it to the changed map
		case string:
			fmt.Println("This is a string:", valTyped)
			changedMap[key] = valTyped // Add to changedMap
		// If the value is a map of strings, iterate over it and print values
		case map[string]string:
			innerMap := make(map[string]interface{})
			for k, v := range valTyped {
				fmt.Println("This is a string within map[string]string:", v)
				innerMap[k] = v // Add string values from map[string]string to a new map
			}
			changedMap[key] = innerMap
		// Default case for unexpected types
		default:
			fmt.Println("Unexpected type:", val)
			changedMap[key] = val // Add unmodified
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

// reads the yaml for http request.
// right now, the yaml is only meant to hold http request as defined in the body struct in "./yamlTypes.go"
func handleYamlFile(filepath string) (*bytes.Buffer, error) {
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

	// TODO: after lowering the key, change all the values as well {{}}
	data = utils.ConvertKeysToLowerCase(data)

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

// replace the user's map's value's {{ }} with proper variable
func GenerateFinalYamlMap(jsonString string) string {
	// TODO: Replace all the {{}} right before you validate them. URL needs the right variables to work.

	// envMap, err := envparser.GenerateSecretsMap()
	// if err != nil {
	// 	utils.PanicRedAndExit("creating environment map: %v", err)
	// }

	// finalUrl, err := envparser.SubstitueVariables(string(user.Url), envMap)
	// if err != nil {
	// 	utils.PanicRedAndExit("creating environment map: %v", err)
	// }
	// user.Url = URL(finalUrl)
	return ""
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
