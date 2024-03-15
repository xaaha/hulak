package envparser

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

var hulakEnvironmentVariables map[string]string

const (
	envKey        = "hulakEnv"
	defaultEnvVal = "global"
)

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
Sets default environment for the user.
Global is default if -env flagName is not provided.
Also, asks the user if they want to create the file in env folder
*/
func SetEnvironment() error {
	// set default value if the env is empty
	if os.Getenv(envKey) == "" {
		err := os.Setenv(envKey, defaultEnvVal)
		if err != nil {
			return fmt.Errorf("error setting environment variable: %v", err)
		}
	}
	// get a list of env files and get their file name
	environmentFiles, err := GetEnvFiles()
	if err != nil {
		return err
	}
	var envFromFiles []string
	for _, file := range environmentFiles {
		file = strings.ToLower(file)
		fileName := strings.ReplaceAll(file, ".env", "")
		envFromFiles = append(envFromFiles, fileName)
	}

	// get user's provided value
	envFromFlag := flag.String("env", defaultEnvVal, "environment files")
	flag.Parse()
	*envFromFlag = strings.ToLower(*envFromFlag)

	// compare both values
	if !slices.Contains(envFromFiles, *envFromFlag) {
		fmt.Printf(
			"%v does not exist in the env directory. Current Active Environment: %v.",
			*envFromFlag, os.Getenv(envKey),
		)
		// ask if the file does not exist
		fmt.Printf("Would you like to create the file %v? (y/n)", *envFromFlag)
		reader := bufio.NewReader(os.Stdin)
		responses, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read responses: %v", err)
		}
		if strings.TrimSpace(responses) == "y" || strings.TrimSpace(responses) == "Y" {
			err := CreateDefaultEnvs(envFromFlag)
			if err != nil {
				return fmt.Errorf("failed to create environment file: %v", err)
			}
			fmt.Println("File Successfully Created")
		} else {
			fmt.Println("Skipping file Creation")
		}
	}

	err = os.Setenv(envKey, *envFromFlag)
	if err != nil {
		return err
	}
	return nil
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

// Given .env file path this func loads environment variables to a hulakEnvironmentVariables map
func LoadEnvVars(filePath string) error {
	hulakEnvironmentVariables = make(map[string]string)
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
		hulakEnvironmentVariables[key] = val
	}
	return nil
}

/*
Generate FinalHulakEnvironment FinalEnvMap
Users Choice >  Global
*/
func FinalEnvMap() error {
	err := SetEnvironment()
	if err != nil {
		return fmt.Errorf("error while setting environment: %v", err)
	}
	envVal, ok := os.LookupEnv(envKey)
	if !ok {
		return fmt.Errorf("error while looking up environment variable")
	}

	envFileName := envVal + ".env"

	// if envFileName is not global, then create a map first with the val provided
	// then merge the maps replacing the global with defined
	// main.go has the sample code
	// make sure the LoadEnv can actually merge the two maps, while replacing the duplicate values

	completeFilePath, err := utils.CreateFilePath(envFileName)
	if err != nil {
		return fmt.Errorf("error during creating  %v: %v", envFileName, err)
	}

	err = LoadEnvVars(completeFilePath)
	if err != nil {
		return fmt.Errorf("error while loading, %v, : %v", completeFilePath, err)
	}

	return nil
}

// using the os.GetEnv grab the default env set by SetDefaultEnv
// Then construct the file name and use
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
  - When user passes -env staging or something similar in the shell
  - There should be a terminal ui to change the envrionment for now
  - hulak -env staging should do it whether itself or with other command
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
