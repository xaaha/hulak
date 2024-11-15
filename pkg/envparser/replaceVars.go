package envparser

import (
	"regexp"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// looks for the secret in the envMap && substitue the actual value in place of {{...}}
// argument: string {{keyName}}
func SubstituteVariables(strToChange string, mapWithVars map[string]string) (string, error) {
	if len(strToChange) == 0 {
		return "", utils.ColorError(utils.EmptyVariables)
	}
	regex := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`) // matches {{key}}
	matches := regex.FindAllStringSubmatch(strToChange, -1)
	if len(matches) == 0 {
		return strToChange, nil
	}
	for _, match := range matches {
		/*
			match[0] is the full match, match[1] is the first group
			thisisa/{{test}}/ofmywork/{{work}} => [["{{test}}" "test"] ["{{work}}" "work"]]
		*/
		envKey := match[1]
		envVal := mapWithVars[envKey]
		if len(envVal) == 0 {
			message := utils.UnResolvedVariable + envKey
			return "", utils.ColorError(message)
		}
		strToChange = strings.Replace(strToChange, match[0], envVal, 1)
		matches = regex.FindAllStringSubmatch(strToChange, -1)
		if len(matches) > 0 {
			return SubstituteVariables(strToChange, mapWithVars)
		}
	}
	return strToChange, nil
}

//
// func SubstituteVariables(strToChange string, mapWithVars map[string]string) (string, error) {
// 	if len(strToChange) == 0 {
// 		return "", errors.New("Input string is empty")
// 	}
// 	tmpl, err := template.New("template").Parse(strToChange)
// 	if err != nil {
// 		return "", fmt.Errorf("Failed to parse template: %w", err)
// 	}
// 	var result bytes.Buffer
// 	err = tmpl.Execute(&result, mapWithVars)
// 	if err != nil {
// 		return "", fmt.Errorf("Failed to execute template: %w", err)
// 	}
// 	return result.String(), nil
// }
//
// func main() {
// 	str := "Hello, {{.name}}! Welcome to {{.place}}. This is a {{.project}}. And a comment {{/* a comment */}}. Finally, an {{.error}}"
// 	vars := map[string]string{
// 		"name":    "John",
// 		"place":   "Gopherland",
// 		"project": "{{.hulak}}",
// 		"hulak":   "Hulak v1",
// 	}
// 	// in case the map has nested vals
// 	for key, val := range vars {
// 		result, err := SubstituteVariables(val, vars)
// 		vars[key] = result
// 		if err != nil {
// 			fmt.Println("Error:", err)
// 		} else {
// 			fmt.Println(result)
// 		}
// 	}
//
// 	result, err := SubstituteVariables(str, vars)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	} else {
// 		fmt.Println(result)
// 	}
// }
