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
// So, we need to, read the file, make sure those values are handled, then return the proper map

// Parses the user's input yaml file to a json interface.
// Then, this function recursively replaces all variables {{.value}} specified in user's yaml values, with values from environment map
// This is necessary, as the some variables, like URL needs correct string
func replaceVarsWithValues(
	dict map[string]interface{},
	secretsMap map[string]interface{},
) map[string]interface{} {
	changedMap := make(map[string]interface{})

	for key, val := range dict {
		switch valTyped := val.(type) {
		case map[string]interface{}:
			changedMap[key] = replaceVarsWithValues(valTyped, secretsMap)
		case string:
			finalChangedValue, err := envparser.SubstituteVariables(valTyped, secretsMap)
			if err != nil {
				utils.PrintRed(err.Error())
			}
			if replacedValue, ok := secretsMap[valTyped]; ok {
				changedMap[key] = replacedValue
			} else {
				changedMap[key] = finalChangedValue
			}
		case map[string]string:
			innerMap := make(map[string]interface{})
			for k, v := range valTyped {
				finalChangedValue, err := envparser.SubstituteVariables(v, secretsMap)
				if err != nil {
					utils.PrintRed(err.Error())
				}
				if replacedValue, ok := secretsMap[v]; ok {
					innerMap[k] = replacedValue
				} else {
					innerMap[k] = finalChangedValue
				}
			}
			changedMap[key] = innerMap
		default:
			changedMap[key] = val
		}
	}
	return changedMap
}

// checks whether string has "{{value}}"
func stringHasDelimiter(s string) bool {
	if len(s) < 4 || !strings.HasPrefix(s, "{{") || !strings.HasSuffix(s, "}}") {
		return false
	}
	if strings.Count(s[:3], "{") > 2 || strings.Count(s[len(s)-3:], "}") > 2 {
		return false
	}
	// Extract the content inside the curly braces
	content := s[2 : len(s)-2]
	// Check if the content is non-empty and doesn't consist solely of whitespace characters
	return len(strings.TrimSpace(content)) > 0
}

func CompareAndConvert(
	dataBefore, dataAfter, secretsMap map[string]interface{},
) map[string]interface{} {
	var result map[string]interface{}
	// range over on dataBefore, (key, value) and find all the values, with valid delimiters "{{}}" -- keep track of the key and value we are concerned with `"myAwesomeNumber": "{{.myAwesomeNumber}}"`
	// here, myAwesomeNumber value is of type string
	// then range over the secretsMap, map[string]interface{}, -- find the key `myAwesomeNumber` and determine it's type -- `int` in this case 22
	// and find the type, (string, int, float, bool or null), and compare  the type with the value. -- compare the type int from secretsMap to the
	// here, myAwesomeNumber value's type is int
	// if there is a mismatch, find the exact key, in the dataAfter map[string] and convert it's value to the the type shown by secretsMap
	// so, finally, in dataAfter, the map would be `"myAwesomeNumber": 22` from ``"myAwesomeNumber": "22"``
	return result
}

// Reads YAML, validates if the file exists, is not empty, and changes keys to lowercase for http request.
// Right now, the yaml file is only meant to hold http request as defined in the body struct in "./yamlTypes.go"
func checkYamlFile(filepath string, secretsMap map[string]interface{}) (*bytes.Buffer, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		utils.PanicRedAndExit("File does not exist, %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		utils.PanicRedAndExit("Error opening file: %v", err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		utils.PanicRedAndExit("Empty yaml file")
	}

	var data map[string]interface{}
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		utils.PanicRedAndExit("1. error decoding data: %v", err)
	}

	// case sensitivity keys in yaml file is ignored.
	// method or Method or METHOD should all be the same
	data = utils.ConvertKeysToLowerCase(data)

	// TODO:
	// if data has key whose value is a template,
	// && the replacement value's type is either false, float64, int, nil/null
	// convert these to original values again
	// or if the key is the same,
	// and the value are of different type, convert them from string to the one of secretsMap

	// parse all the values to with {{.key}} from .env folder
	parsedMap := replaceVarsWithValues(data, secretsMap)

	// dataFmt, _ := utils.MarshalToJSON(data)
	// fmt.Println("this is data", dataFmt)
	// printPm, _ := utils.MarshalToJSON(parsedMap)
	// fmt.Println("this is parsed map", printPm)

	// TODO:
	// parsedMap is always string

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(parsedMap); err != nil {
		utils.PanicRedAndExit("error encoding data: %v", err)
	}
	enc.Close()

	return &buf, nil
}

// checks the validity of all the fields in the yaml file
// and returns the json string of the yaml file
func ReadYamlForHttpRequest(filePath string, secretsMap map[string]interface{}) string {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		utils.PanicRedAndExit("Error occured after reading yaml file: %v", err)
	}

	var user User
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&user); err != nil {
		utils.PanicRedAndExit("2. error decoding data: %v", err)
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
	// if the body is not present in the body, then the body is nil
	if user.Body != nil && !user.Body.IsValid() {
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
		utils.PanicRedAndExit("3. error decoding data: %v", err)
	}

	val, _ := json.MarshalIndent(data, "", "  ")
	// log prints time, which I don't need
	fmt.Println(string(val))

	// for key, value := range data {
	// 	fmt.Printf("%s: %v\n", key, value)
	// }
}
