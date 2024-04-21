package envparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// Provide a key name as as tring to get get secret value from envVars map
func GetEnvVarValue(key string) (string, bool) {
	envMap, err := GenerateFinalEnvMap()
	if err != nil {
		panic(err)
	}
	value, ok := envMap[key]
	return value, ok
}

/*
* looks for the secret in the envMap && substitue the actual value in place of {{...}}
* argument: string {{keyName}}
 */
func SubstitueVariables(input string) (string, error) {
	if len(input) == 0 {
		return "", utils.ColorError("input string can't be empty")
	}
	// matches string with: {{key}}
	regex := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
	matches := regex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return "", utils.ColorError("ensure you've key name inside {{}}")
	}
	for _, match := range matches {
		/*
			match[0] is the full match, match[1] is the first group
			thisisa/{{test}}/ofmywork/{{work}} => [["{{test}}" "test"] ["{{work}}" "work"]]
		*/
		envKey := match[1]

		fmt.Println("Env Key:", envKey)
		if envVal, ok := GetEnvVarValue(envKey); ok {
			input = strings.Replace(input, match[0], envVal, 1)
		} else {
			errorMessge := "unresolved variable " + envKey
			return "", utils.ColorError(errorMessge)
		}
	}
	return input, nil
}
