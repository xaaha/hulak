package envparser

import (
	"bufio"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// var HulakEnvironmentVariable map[string]string

const (
	envKey            = "hulakEnv"
	defaultEnvVal     = "global"
	defaultFileSuffix = ".env"
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
func setEnvironment() error {
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
		fileName := strings.ReplaceAll(file, defaultFileSuffix, "")
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

// Given .env file path this func returns map of the key-value pair of the content
func LoadEnvVars(filePath string) (map[string]string, error) {
	hulakEnvironmentVariable := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
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
		hulakEnvironmentVariable[key] = val
	}
	return hulakEnvironmentVariable, nil
}

/*
Generate final HulakEnvironment map.
User's Choice > Global.
When user has custom env they want to use, it merges custom with global env.
Replaces global key with custom when keys repeat
*/
func GenerateFinalEnvMap() (map[string]string, error) {
	err := setEnvironment()
	if err != nil {
		return nil, fmt.Errorf("error while setting environment: %v", err)
	}
	envVal, ok := os.LookupEnv(envKey)
	if !ok {
		return nil, fmt.Errorf("error while looking up environment variable")
	}
	if envVal == "" {
		envVal = defaultEnvVal
	}

	envFileName := envVal + defaultFileSuffix
	completeFilePath, err := utils.CreateFilePath(envFileName)
	if err != nil {
		return nil, fmt.Errorf("error while creating  %v: %v", envFileName, err)
	}

	customMap, err := LoadEnvVars(completeFilePath)
	if err != nil {
		return nil, fmt.Errorf("error while loading, %v, : %v", completeFilePath, err)
	}

	//	when user has custom env
	if envFileName != defaultEnvVal {
		globalEnv := "global.env"
		globalPath, err := utils.CreateFilePath(globalEnv)
		if err != nil {
			return nil, fmt.Errorf("error while creating  %v: %v", globalEnv, err)
		}
		globalMap, err := LoadEnvVars(globalPath)
		if err != nil {
			return nil, fmt.Errorf("error while loading, %v, : %v", globalPath, err)
		}
		maps.Copy(globalMap, customMap)
	}
	return customMap, nil
}

/*
  - When user passes -env staging or something similar in the shell
  - There should be a terminal ui to change the envrionment for now
  - hulak -env staging should do it whether itself or with other command
- Global > defined > Collection
*/
