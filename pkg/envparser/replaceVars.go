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
