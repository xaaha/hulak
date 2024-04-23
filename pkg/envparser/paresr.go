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

/*
Sets default environment for the user.
Global is default if -env flagName is not provided.
Also, asks the user if they want to create the file in env folder
*/
func setEnvironment(utility utils.Utilities) (bool, error) {
	fileCreationSkipped := false
	// set default value if the env is empty
	if os.Getenv(utils.EnvKey) == "" {
		err := os.Setenv(utils.EnvKey, utils.DefaultEnvVal)
		if err != nil {
			return fileCreationSkipped, fmt.Errorf("error setting environment variable: %v", err)
		}
	}
	// get a list of env files and get their file name
	environmentFiles, err := utility.GetEnvFiles()
	if err != nil {
		return fileCreationSkipped, err
	}
	var envFromFiles []string
	for _, file := range environmentFiles {
		file = strings.ToLower(file)
		fileName := strings.ReplaceAll(file, utils.DefaultEnvFileSuffix, "")
		envFromFiles = append(envFromFiles, fileName)
	}

	// get user's provided value
	envFromFlag := flag.String(
		"env",
		utils.DefaultEnvVal,
		"environment file to use during the call",
	)
	flag.Parse()

	// Only take the first argument after -env flag, ignore the rest
	arguments := strings.Fields(*envFromFlag)
	if len(arguments) > 0 {
		*envFromFlag = strings.ToLower(arguments[0])
	} else {
		*envFromFlag = utils.DefaultEnvVal
	}

	// compare both values
	if !slices.Contains(envFromFiles, *envFromFlag) {
		fmt.Printf("'%v.env' not found in the env directory\n", *envFromFlag)
		// ask to create the file, if the file does not exist
		fmt.Printf("Create '%v.env'? (y/n)", *envFromFlag)
		reader := bufio.NewReader(os.Stdin)
		responses, err := reader.ReadString('\n')
		if err != nil {
			return fileCreationSkipped, fmt.Errorf("failed to read responses: %v", err)
		}
		if strings.TrimSpace(responses) == "y" || strings.TrimSpace(responses) == "Y" {
			err := CreateDefaultEnvs(envFromFlag)
			if err != nil {
				return fileCreationSkipped, fmt.Errorf("failed to create environment file: %v", err)
			}
		} else {
			fileCreationSkipped = true
			*envFromFlag = utils.DefaultEnvVal
			utils.PrintGreen("Skipping file Creation")
		}
	}

	err = os.Setenv(utils.EnvKey, *envFromFlag)
	fmt.Println("Current active environment:", os.Getenv(utils.EnvKey))
	if err != nil {
		return fileCreationSkipped, err
	}
	return fileCreationSkipped, nil
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
			trimedStr = strings.Trim(eachLine, " ")
		}
		// trim quotes around the =, and before and after the string
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
Generate final map of environemnt variables and it's values
User's Choice > Global.
When user has custom env they want to use, it merges custom with global env.
Replaces global key with custom when keys repeat
*/
func GenerateSecretsMap() (map[string]string, error) {
	skipped, err := setEnvironment(utils.Utilities{})
	if err != nil {
		return nil, fmt.Errorf("error while setting environment: %v", err)
	}
	envVal, ok := os.LookupEnv(utils.EnvKey)
	if !ok {
		return nil, fmt.Errorf("error while looking up environment variable")
	}

	// if the file creation was skipped, resort to default
	if envVal == "" || skipped {
		envVal = utils.DefaultEnvVal
	}

	// load global vars in a map
	globalEnv := utils.DefaultEnvVal + utils.DefaultEnvFileSuffix //"global.env"
	globalPath, err := utils.CreateFilePath("env/" + globalEnv)
	if err != nil {
		return nil, fmt.Errorf("error while creating %v: %v", globalEnv, err)
	}
	globalMap, err := LoadEnvVars(globalPath)
	if err != nil {
		return nil, fmt.Errorf("error while loading %v: %v", globalPath, err)
	}

	// load custom vars in a map if necessary
	var customMap map[string]string
	envFileName := envVal + utils.DefaultEnvFileSuffix
	if globalPath != envFileName {
		completeFilePath, err := utils.CreateFilePath("env/" + envFileName)
		if err != nil {
			return nil, fmt.Errorf("error while creating %v: %v", envFileName, err)
		}

		customMap, err = LoadEnvVars(completeFilePath)
		if err != nil {
			return nil, fmt.Errorf("error while loading %v: %v", completeFilePath, err)
		}
	}

	maps.Copy(customMap, globalMap)
	return customMap, nil
}
