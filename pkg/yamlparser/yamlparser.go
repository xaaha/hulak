// Package yamlparser does everything related to yaml file for hulak, including type translation
package yamlparser

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/actions"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// From the yaml file, create a json file. But the json could have {{}} on it
// So, we need to, read the file, make sure those values are handled, then return the proper map

// Parses the user's input yaml file to a json interface.
// Then, this function recursively replaces all variables {{.value}} specified in user's yaml values, with values from environment map
// This is necessary, as the some variables, like URL needs correct string
func replaceVarsWithValues(
	dict map[string]any,
	secretsMap map[string]any,
) (map[string]any, error) {
	changedMap := make(map[string]any)

	for key, val := range dict {
		switch valTyped := val.(type) {
		case map[string]any:
			// Recursively process nested maps
			nestedMap, err := replaceVarsWithValues(valTyped, secretsMap)
			if err != nil {
				return nil, fmt.Errorf("error in %s: %w", key, err)
			}
			changedMap[key] = nestedMap

		case string:
			// OPTIMIZATION: Skip template parsing if no template syntax present
			if !strings.Contains(valTyped, "{{") {
				changedMap[key] = valTyped
				continue
			}

			// Process template variables
			finalChangedValue, err := envparser.SubstituteVariables(valTyped, secretsMap)
			if err != nil {
				errMsg := fmt.Sprintf("error substituting variable in '%s': %v", key, err)
				return nil, utils.ColorError(errMsg)
			}

			// Only store the processed template value
			changedMap[key] = finalChangedValue

		case map[string]string:
			innerMap := make(map[string]any)
			for k, v := range valTyped {
				// OPTIMIZATION: Skip template parsing if no template syntax present
				if !strings.Contains(v, "{{") {
					innerMap[k] = v
					continue
				}

				// Process template variables
				finalChangedValue, err := envparser.SubstituteVariables(v, secretsMap)
				if err != nil {
					return nil, fmt.Errorf("error substituting variable in '%s.%s': %w", key, k, err)
				}

				// Only store the processed template value
				innerMap[k] = finalChangedValue
			}
			changedMap[key] = innerMap

		default:
			changedMap[key] = val
		}
	}
	return changedMap, nil
}

// Reads YAML, validates if the file exists, is not empty, and changes keys to lowercase
func checkYamlFile(filepath string, secretsMap map[string]any) (*bytes.Buffer, error) {
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

	var data map[string]any
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		utils.PanicRedAndExit("1. error decoding data: %v", err)
	}

	// make yaml keys  case insensitive. method or Method or METHOD should all be the same
	data = utils.ConvertKeysToLowerCase(data)

	// parse all the values to with {{.key}} from .env folder
	parsedMap, err := replaceVarsWithValues(data, secretsMap)
	if err != nil {
		return nil, err
	}

	// translate the types, if acceptable
	parsedMap, err = translateType(data, parsedMap, secretsMap, actions.GetValueOf)
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

// FinalStructForAPI builds a final struct for the api call.
// Returns ApiCallFile struct, true if file is valid, and error
// It  checks the validity of all the fields in the yaml file meant for regular api call
func FinalStructForAPI(filePath string, secretsMap map[string]any) (ApiCallFile, bool, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return ApiCallFile{}, false, err
	}

	var file ApiCallFile
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&file); err != nil {
		return ApiCallFile{}, false, err
	}

	if valid, err := file.IsValid(filePath); !valid {
		return ApiCallFile{}, false, err
	}

	return file, true, nil
}

// FinalStructForOAuth2 checks the validity of all the fields in the yaml file meant for OAuth2.0.
// It returns AuthRequestBody struct
func FinalStructForOAuth2(
	filePath string,
	secretsMap map[string]any,
) (AuthRequestFile, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return AuthRequestFile{}, utils.ColorError("Error after reading yaml file: %v", err)
	}

	var auth2Config AuthRequestFile
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&auth2Config); err != nil {
		return AuthRequestFile{}, utils.ColorError("Error decoding data: %v", err)
	}

	if valid, err := auth2Config.IsValid(); !valid {
		return AuthRequestFile{}, utils.ColorError("Error on Auth2 Request Body %v", err)
	}
	return auth2Config, nil
}
