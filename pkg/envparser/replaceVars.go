package envparser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/xaaha/hulak/pkg/utils"
)

// gets the value of key from the file
// looks for the json _response file, if the file does not exist, it makes a new call, writes the file and then reads it
func getValueOf(key, fileName string) string {
	yamlPathList, err := utils.ListMatchingFiles(fileName)
	if err != nil {
		utils.PrintRed(
			"replaceVars.go: error occured while grabbing matchingPath for " + fileName + " \n" +
				err.Error(),
		)
	}

	singlePath := yamlPathList[0] // there could be multiple yaml file matches only take the first one
	if len(yamlPathList) > 1 {
		utils.PrintWarning(
			"Multiple matches for the file " + fileName + " found. Using the first one",
		)
	}

	dirPath := filepath.Dir(singlePath)
	jsonBaseName := utils.FileNameWithoutExtension(singlePath) + "_response.json"
	jsonResFilePath := filepath.Join(dirPath, jsonBaseName)

	if _, err := os.Stat(jsonResFilePath); os.IsNotExist(err) {
		fmt.Println(jsonResFilePath) // just a place holder for now
		// call the response
	}

	// with fileName find all the paths and use the first one
	// if the _response.json file does not exit, call the api, write the file
	// only in json file. Not sure how would I do this in xml or html yet
	// if the _response.json file exists, use recursion, find the vlaue and return the value
	// if the value does not exist, return an empty string and print the message in Red

	return key + " && " + fileName
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

// we can also use recursion and regex to solve this issue
// trying to use template as much as possible

// func SubstituteVariables(strToChange string, mapWithVars map[string]string) (string, error) {
// 	if len(strToChange) == 0 {
// 		return "", utils.ColorError(utils.EmptyVariables)
// 	}
// 	regex := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`) // matches {{key}}
// 	matches := regex.FindAllStringSubmatch(strToChange, -1)
// 	if len(matches) == 0 {
// 		return strToChange, nil
// 	}
// 	for _, match := range matches {
// 		/*
// 			match[0] is the full match, match[1] is the first group
// 			thisisa/{{test}}/ofmywork/{{work}} => [["{{test}}" "test"] ["{{work}}" "work"]]
// 		*/
// 		envKey := match[1]
// 		envVal := mapWithVars[envKey]
// 		if len(envVal) == 0 {
// 			message := utils.UnResolvedVariable + envKey
// 			return "", utils.ColorError(message)
// 		}
// 		strToChange = strings.Replace(strToChange, match[0], envVal, 1)
// 		matches = regex.FindAllStringSubmatch(strToChange, -1)
// 		if len(matches) > 0 {
// 			return SubstituteVariables(strToChange, mapWithVars)
// 		}
// 	}
// 	return strToChange, nil
// }
