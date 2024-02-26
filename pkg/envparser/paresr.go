package envparser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envVars map[string]string

// if the .env secrets have " " or ” around them, remove it
func trimQuotes(str string) string {
	if len(str) >= 2 {
		if str[0] == str[len(str)-1] && (str[0] == '"' || str[0] == '\'') {
			return str[1 : len(str)-1]
		}
	}
	return str
}

// LoadEnv loads environment variables from the given file path into envVars
func ParsingEnv(filePath string) error {
	envVars = make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// skip empty line and comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		// remove the empty lines
		splitStr := strings.Split(line, "\n")
		secret := strings.Split(splitStr[0], "=")
		if len(secret) < 2 {
			// if there is no =
			continue
		}
		key := secret[0]
		val := secret[1]
		val = trimQuotes(val)
		envVars[key] = val
	}
	// also load global file path by default
	// if the global does not exist, no need to panic. Just exit.
	// Same with the collection level .env files
	return nil
}

// Get secret value from envVars
func GetEnvVar(key string) (string, bool) {
	value, ok := envVars[key]
	return value, ok
}

// look for the string in the env map && substitue the actual value in place of {{...}}
func SubstitueVariables(input string) (string, error) {
	if len(input) == 0 {
		return "", errors.New("input string can't be empty")
	}
	regex := regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)
	matches := regex.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		// match[0] is the full match, match[1] is the first group
		// thisisa/{{test}}/ofmywork/{{work}} => [["{{test}}" "test"] ["{{work}}" "work"]]
		envKey := match[1]
		if envVal, ok := GetEnvVar(envKey); ok {
			input = strings.Replace(input, match[0], envVal, 1)
		} else {
			return "", fmt.Errorf("unresolved variable: %s", envKey)
		}
	}

	return input, nil
}

/*
Handle boolean
Handle upperCase and lowerCase
Tests
Check SubstitueVariables and make sure the substitution is working as expected
Make sure no two items have the same key or is replaced by the later key/value pair


*/

// be able to set a .env file as main so that,  {{}} is read as a variable
/*
- Find the name in the default current environment.
- Default is global only if user does not have a defined environment.
- Global > defined > Collection
// something like this but with user's folder
func ActiveEnv(envName string) error {
	var filePath string
	switch envName {
	case "global":
		filePath = "path/to/global.env"
	case "test":
		filePath = "path/to/test.env"
	case "prod":
		filePath = "path/to/prod.env"
	default:
		return errors.New("unknown environment")
	}
	return LoadEnv(filePath)
}
*/
