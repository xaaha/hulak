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
func getValueOf(key, fileName string) string {
	if key == "" && fileName == "" {
		utils.PanicRedAndExit("replaceVars.go: key and fileName can't be empty")
	}

	yamlPathList, err := utils.ListMatchingFiles(fileName)
	if err != nil {
		utils.PrintRed(
			"replaceVars.go: error occured while grabbing matchingPath for " + fileName + " \n" +
				err.Error(),
		)
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

	file, err := os.Open(jsonBaseName)
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
		utils.PanicRedAndExit("replaceVars.go: error while looking up the value")
	}

	return result
}

func replaceVariables(
	strToChange string,
	secretsMap map[string]string,
) (string, error) {
	if len(strToChange) == 0 {
		return "", utils.ColorError("input string is empty")
	}

	funcMap := template.FuncMap{
		"getValueOf": func(key, fileName string) string {
			return getValueOf(key, fileName)
		},
	}

	tmpl, err := template.New("template").
		Funcs(funcMap).
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

// Replace the template {{ }} in the variable map itself.
// Sometimes, we have a variables map that references some other variable in itself.
func prepareMap(secretsMap map[string]string) (map[string]string, error) {
	for key, val := range secretsMap {
		changedStr, err := replaceVariables(val, secretsMap)
		if err != nil {
			return nil, utils.ColorError("error while preparing variables in map: %v", err)
		}
		secretsMap[key] = changedStr
	}
	return secretsMap, nil
}

// from the mapWithVars, parse the string and replace {{}}
func SubstituteVariables(
	strToChange string,
	secretsMap map[string]string,
) (string, error) {
	finalMap, err := prepareMap(secretsMap)
	if err != nil {
		return "", err
	}

	result, err := replaceVariables(strToChange, finalMap)
	if err != nil {
		return "", err
	}
	return result, nil
}
