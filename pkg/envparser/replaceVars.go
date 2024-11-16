package envparser

import (
	"bytes"
	"html/template"

	"github.com/xaaha/hulak/pkg/utils"
)

func replaceVariables(
	strToChange string,
	mapWithVars map[string]string,
) (string, error) {
	if len(strToChange) == 0 {
		return "", utils.ColorError("input string is empty")
	}
	tmpl, err := template.New("template").Option("missingkey=error").Parse(strToChange)
	if err != nil {
		return "", utils.ColorError("template parsing error: %w", err)
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, mapWithVars)
	if err != nil {
		return "", utils.ColorError("%v", err)
	}
	return result.String(), nil
}

// Replace the template {{ }} in the variable map itself.
// Sometimes, we have a variables map that references some other variable in itself.
func prepareMap(varsMap map[string]string) (map[string]string, error) {
	for key, val := range varsMap {
		changedStr, err := replaceVariables(val, varsMap)
		if err != nil {
			return nil, utils.ColorError("error while preparing variables in map: %v", err)
		}
		varsMap[key] = changedStr
	}
	return varsMap, nil
}

func SubstituteVariables(
	strToChange string,
	mapWithVars map[string]string,
) (string, error) {
	finalMap, err := prepareMap(mapWithVars)
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
