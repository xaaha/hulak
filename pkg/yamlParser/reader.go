package yamlParser

import (
	"bytes"
	"encoding/json"
	"os"

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

// Reads YAML, validates if the file exists, is not empty, and changes keys to lowercase
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

	// make yaml keys  case insensitive. method or Method or METHOD should all be the same
	data = utils.ConvertKeysToLowerCase(data)

	// parse all the values to with {{.key}} from .env folder
	parsedMap := replaceVarsWithValues(data, secretsMap)

	// translate the types, if acceptable
	parsedMap, err = translateType(data, parsedMap, secretsMap, envparser.GetValueOf)
	if err != nil {
		return nil, utils.ColorError("#reader", err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(parsedMap); err != nil {
		utils.PanicRedAndExit("error encoding data: %v", err)
	}
	enc.Close()

	return &buf, nil
}

// checks the validity of all the fields in the yaml file meant for regular api call
// and returns the json string of the yaml file
func FinalJsonForHttpRequest(filePath string, secretsMap map[string]interface{}) string {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		utils.PanicRedAndExit("Error occured after reading yaml file: %v", err)
	}

	var user User
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&user); err != nil {
		utils.PanicRedAndExit("2. error decoding data: %v", err)
	}

	user.Method.ToUpperCase()

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

// checks the validity of all the fields in the yaml file meant for OAuth2.0.
// It returns AuthRequestBody struct
func FinalStructForOAuth2(filePath string, secretsMap map[string]interface{}) AuthRequestBody {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		utils.PanicRedAndExit("authTypes.go: Error occured after reading yaml file: %v", err)
	}

	var user AuthRequestBody
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&user); err != nil {
		utils.PanicRedAndExit("reader.go: error decoding data: %v", err)
	}
	// uppercase the method
	user.Method.ToUpperCase()

	// method is required as each implementation of  Auth2.0 is different
	if !user.Method.IsValid() {
		utils.PanicRedAndExit("missing or invalid HTTP method: %s", user.Method)
	}

	if user.Auth == nil || !user.Auth.IsValid() {
		utils.PanicRedAndExit(
			"Invalid 'auth' section. Make sure the file '%s' contains valid auth section.\n %v",
			filePath,
			user.Auth,
		)
	}

	// make sure authorize url, that opens up in browser, is required for auth2.0
	if !user.Url.IsValidURL() {
		utils.PanicRedAndExit("missing or invalid URL: %s", user.Url)
	}

	return user
}
