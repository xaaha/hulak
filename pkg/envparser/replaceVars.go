package envparser

import (
	"fmt"
	"regexp"
	"strings"
)

// Get secret value from envVars map
func GetEnvVarValue(key string) (string, bool) {
	envMap, err := GenerateFinalEnvMap()
	if err != nil {
		panic(err)
	}
	value, ok := envMap[key]
	return value, ok
}

// looks for the secret in the envMap && substitue the actual value in place of {{...}}
func SubstitueVariables(input string) (string, error) {
	if len(input) == 0 {
		return "", fmt.Errorf("input string can't be empty")
	}
	// matches string with: {{key}}
	regex := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
	matches := regex.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		/*
			match[0] is the full match, match[1] is the first group
			thisisa/{{test}}/ofmywork/{{work}} => [["{{test}}" "test"] ["{{work}}" "work"]]
		*/
		envKey := match[1]
		if envVal, ok := GetEnvVarValue(envKey); ok {
			input = strings.Replace(input, match[0], envVal, 1)
		} else {
			return "", fmt.Errorf("unresolved variable: %s", envKey)
		}
	}

	return input, nil
}
