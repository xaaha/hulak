package envparser

import (
	"bytes"
	"text/template"

	"github.com/xaaha/hulak/pkg/utils"
)

// gets the value of key from the json path provided
func getValueOf(key, path string) string {
	return key + " && " + path
}

func replaceVariables(
	strToChange string,
	secretsMap map[string]string,
) (string, error) {
	if len(strToChange) == 0 {
		return "", utils.ColorError("input string is empty")
	}

	funcMap := template.FuncMap{
		"getValueOf": func(key, path string) string {
			return getValueOf(key, path)
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
