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
But it's not being used
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
	// also load global file path by default
	// if the global does not exist, no need to panic. Just create one and exit.
	// Same with the collection level .env files
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

// Set SetDefault environment as global.
func SetDefaultEnv() error {
	err := os.Setenv("hulakEnv", "global")
	if err != nil {
		return fmt.Errorf("error setting environment variable: %v", err)
	}
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
	userDefinedEnv := *envFromFlag
	userDefinedEnv = strings.ToLower(userDefinedEnv)

	if !slices.Contains(environments, userDefinedEnv) {
		fmt.Println(
			"Environment file does not exist in the env folder. Default Environment value",
			os.Getenv("hulakEnv"),
		)
		return fmt.Errorf("create %v.env file", userDefinedEnv)
	}
	err = os.Setenv("hulakEnv", userDefinedEnv)
	if err != nil {
		return err
	}
	// if the flag user provides does not exist in the env folder
	// ask if the user want's to create the file in the folder
	return nil
}

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
