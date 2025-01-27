package envparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/xaaha/hulak/pkg/utils"
)

// gets the value of key from a json file
// looks for the json _response file, if the file does not exist, it makes a new call, writes the file and then reads it
func GetValueOf(key, fileName string) interface{} {
	if key == "" && fileName == "" {
		utils.PanicRedAndExit("replaceVars.go: key and fileName can't be empty")
	}

	yamlPathList, err := utils.ListMatchingFiles(fileName)
	if err != nil {
		utils.PrintRed(fmt.Sprintf(
			"replaceVars.go: error occurred while grabbing matching paths for '%s': %s",
			fileName, err.Error(),
		))
		return ""
	}

	var singlePath string
	if len(yamlPathList) > 0 {
		// only take the first one
		singlePath = yamlPathList[0]
	} else {
		utils.PrintRed("could not find matching files " + fileName)
		return ""
	}

	if len(yamlPathList) > 1 {
		utils.PrintWarning(
			"Multiple matches for the file " + fileName + " found. Using \n" + singlePath,
		)
	}

	dirPath := filepath.Dir(singlePath)

	// TODO 1: if the fileName contains .json then there is no need to  do following two steps
	// here I am exclusively looking for test_response.json where test is the original yal file
	// but, since I have json file in the response, we should check whether the file contains the .json,
	// if the singlePath contains, .json, then use that instead
	jsonBaseName := utils.FileNameWithoutExtension(singlePath) + utils.ResponseFileName
	jsonResFilePath := filepath.Join(dirPath, jsonBaseName)

	// If the file does not exist
	if _, err := os.Stat(jsonResFilePath); os.IsNotExist(err) {
		utils.PrintRed(fmt.Sprintf(
			"%s file does not exist. Either fetch the API response for '%s', or make sure the '%s' exists with '%s'. \n",
			jsonResFilePath,
			fileName,
			jsonResFilePath,
			key,
		))
		return ""
	}

	file, err := os.Open(jsonResFilePath)
	if err != nil {
		utils.PrintRed(
			fmt.Sprintf(
				"replaceVars.go: error occured while opening the file '%s': %s",
				jsonBaseName,
				err.Error(),
			),
		)
	}
	defer file.Close()

	var fileContent map[string]interface{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&fileContent)
	if err != nil {
		utils.PrintRed(
			"replaceVars.go: make sure " + jsonBaseName + " has proper json content" + err.Error(),
		)
	}

	result, err := utils.LookupValue(key, fileContent)
	if err != nil {
		utils.PanicRedAndExit(
			"replaceVars.go: error while looking up the value: '%s'. \nMake sure '%s' exists and has key '%s'",
			key,
			filepath.Join("...", utils.FileNameWithoutExtension(dirPath), jsonBaseName),
			key,
		)
	}

	return result
}

// Processes a given string, strToChange, by substituting template variables with values from the secretsMap.
// It uses Go’s template package to parse the string, dynamically.
// Returns the updated string or an error if parsing or execution fails.
func replaceVariables(
	strToChange string,
	secretsMap map[string]interface{},
) (string, error) {
	if len(strToChange) == 0 {
		return "", utils.ColorError("input string is empty")
	}

	getValueOf := template.FuncMap{
		"getValueOf": func(key, fileName string) interface{} {
			return GetValueOf(key, fileName)
		},
	}

	tmpl, err := template.New("template").
		Funcs(getValueOf).
		Option("missingkey=error").
		Parse(strToChange)
	if err != nil {
		return "", utils.ColorError("template parsing error: %w", err)
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, secretsMap)
	if err != nil {
		return "", utils.ColorError("%v", err)
	}
	return result.String(), nil
}

// Iterates over a map of secret values (secretsMap), resolving any string values
// containing template variables using the replaceVariables function.
// It ensures that non-string values (e.g., booleans, integers) are preserved and validates against unsupported types.
// Returns a new map with resolved values or an error if any resolution fails.
func prepareMap(secretsMap map[string]interface{}) (map[string]interface{}, error) {
	updatedMap := make(map[string]interface{})
	for key, val := range secretsMap {
		switch v := val.(type) {
		case string:
			changedValue, err := replaceVariables(v, secretsMap)
			if err != nil {
				return nil, utils.ColorError("error while preparing variables in map: %v", err)
			}
			updatedMap[key] = changedValue
		case bool, int, float64, nil:
			updatedMap[key] = v
		default:
			return nil, utils.ColorError(
				fmt.Sprintf("unsupported type for key '%s': %T", key, val),
			)
		}
	}
	return updatedMap, nil
}

// Substitutes template variables in a given string strToChange using the secretsMap.
// It first prepares the map by resolving all nested variables using prepareMap
// and then applies replaceVariables to the input string.
// Returns the final substituted string or an error if any step fails.
func SubstituteVariables(
	strToChange string,
	secretsMap map[string]interface{},
) (interface{}, error) {
	finalMap, err := prepareMap(secretsMap)
	if err != nil {
		return nil, err
	}

	result, err := replaceVariables(strToChange, finalMap)
	if err != nil {
		return nil, err
	}

	return result, nil
}
