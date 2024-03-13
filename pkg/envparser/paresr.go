package envparser

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// At this point. Secrets only support strings
var envVars map[string]string

/*
TODO: parse the secrets dynamically to it's respective types
GetEnvVarGeneric attempts to retrieve an environment variable and guess its type.
Currently it's not being used
*/
func GetEnvVarGeneric(key string) (interface{}, error) {
	valueStr, ok := envVars[key]
	if !ok {
		return nil, fmt.Errorf("environment variable not found: %s", key)
	}

	// Attempt to parse as bool
	if valueBool, err := strconv.ParseBool(valueStr); err == nil {
		return valueBool, nil
	}

	// Attempt to parse as int
	if valueInt, err := strconv.Atoi(valueStr); err == nil {
		return valueInt, nil
	}

	// Attempt to parse as float
	if valueFloat, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return valueFloat, nil
	}

	// Default to string if no other types match
	return valueStr, nil
}

// Removes doube quotes " " or single quotes ' from env secrets
func trimQuotes(str string) string {
	if len(str) >= 2 {
		if str[0] == str[len(str)-1] && (str[0] == '"' || str[0] == '\'') {
			return str[1 : len(str)-1]
		}
	}
	return str
}

// LoadEnv loads environment variables from the given file path into envVars map
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

		// trim all empty spaces around the secret line and around =
		var trimedStr string
		for _, eachLine := range splitStr {
			trimedStr = strings.ReplaceAll(eachLine, " ", "")
		}
		// trim quotes around the =, and before and after the string
		fmt.Println("String that is trimed of spaces", trimedStr)
		secret := strings.Split(trimedStr, "=")
		if len(secret) < 2 {
			// if there is no =
			continue
		}
		key := secret[0]
		val := secret[1]
		val = trimQuotes(val)
		envVars[key] = val
	}
	return nil
}

// Get secret value from envVars map
func getEnvVar(key string) (string, bool) {
	value, ok := envVars[key]
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
		if envVal, ok := getEnvVar(envKey); ok {
			input = strings.Replace(input, match[0], envVal, 1)
		} else {
			return "", fmt.Errorf("unresolved variable: %s", envKey)
		}
	}

	return input, nil
}

// Get a list of environment file names from the env folder
func GetEnvFiles() ([]string, error) {
	var environmentFiles []string
	dir, err := os.Getwd()
	if err != nil {
		return environmentFiles, err
	}
	// get a list of envFileName
	contents, err := os.ReadDir(dir + "/env")
	if err != nil {
		panic(err)
	}

	// discard any folder in the env directory
	for _, fileOrDir := range contents {
		if !fileOrDir.IsDir() {
			environmentFiles = append(environmentFiles, fileOrDir.Name())
		}
	}
	fmt.Println("Env Files", environmentFiles)
	return environmentFiles, nil
}

/*
Sets global as default env if -env flag is not provided
*/
func SetDefaultEnv() error {
	// set default hulakEnv
	err := os.Setenv("hulakEnv", "global")
	if err != nil {
		return fmt.Errorf("error setting environment variable: %v", err)
	}
	// get a list of env files and get their file name
	environmentFiles, err := GetEnvFiles()
	if err != nil {
		return err
	}
	var environments []string
	for _, file := range environmentFiles {
		file = strings.ToLower(file)
		fileName := strings.ReplaceAll(file, ".env", "")
		environments = append(environments, fileName)
	}
	envFromFlag := flag.String("env", "global", "environment files")
	flag.Parse()
	*envFromFlag = strings.ToLower(*envFromFlag)

	if !slices.Contains(environments, *envFromFlag) {
		fmt.Printf(
			"%v does not exist in the env folder. Current Environment: %v.",
			*envFromFlag, os.Getenv("hulakEnv"),
		)
		return fmt.Errorf("create %v.env file in the env folder", *envFromFlag)
	}
	err = os.Setenv("hulakEnv", *envFromFlag)
	if err != nil {
		return err
	}
	/*
		- If the user has provided the flag during run.
			-  Get the flag's value.
				- handle the error if the flag is set but the value is not provided.
			- Check the flag's value against a list of available env files.
				- If the flag's value does not match what's available. Ask user if they want to create the file.
					- If Yes ~ create a file and set the env as the name.
					- If No ~ let the user know that the default value is
						- Get env from global.
				- If the flag's value
		- If the user has does not have the  flag
			- Then set global as default.
		- User should be able to set the variable from terminal.
	*/
	return nil
}

/*
Write a function that is triggered if the user has provided flag... Figure this out.
*/

/*
// using the function above
func howToUse() {
	// get cwd
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	filePath := filepath.Join(cwd, ".env.global")
	err = ParsingEnv(filePath)
	if err != nil {
		panic(err)
	}
	resolved, err := SubstitueVariables("myNameIs={{NAME}}")
	if err != nil {
		panic(err)
	}
	fmt.Println(resolved)
}
*/

/*
- When do we set up the env?
  - When user passes -env staging or something similar in the shell
  - There should be a terminal ui to change the envrionment for now
  - hulak -env staging should do it whether itself or with other command
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
