package envparser

import (
	"regexp"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

/*
* looks for the secret in the envMap && substitue the actual value in place of {{...}}
* argument: string {{keyName}}
 */
func SubstitueVariables(strToChange string, mapWithVars map[string]string) (string, error) {
	if len(strToChange) == 0 {
		return "", utils.ColorError("variable string can't be empty")
	}
	// matches string with: {{key}}
	regex := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
	matches := regex.FindAllStringSubmatch(strToChange, -1)
	if len(matches) == 0 {
		return "", utils.ColorError("ensure you've proper key name inside {{}}")
	}
	for _, match := range matches {
		/*
			match[0] is the full match, match[1] is the first group
			thisisa/{{test}}/ofmywork/{{work}} => [["{{test}}" "test"] ["{{work}}" "work"]]
		*/
		envKey := match[1]
		envVal := mapWithVars[envKey]
		if len(envVal) == 0 {
			message := "unresolved variable " + envKey
			return "", utils.ColorError(message)
		}
		strToChange = strings.Replace(strToChange, match[0], envVal, 1)
	}
	return strToChange, nil
}
