// Package envparser contains environment parsing and functions around it
package envparser

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

/*
Sets default environment for the user.
Global is default if -env flagName is not provided.
Also, asks the user if they want to create the file in env folder
*/
func setEnvironment(envFromFlag string) (bool, error) {
	fileCreationSkipped := false
	// set default value if the env is empty
	if os.Getenv(utils.EnvKey) == "" {
		err := os.Setenv(utils.EnvKey, utils.DefaultEnvVal)
		if err != nil {
			return fileCreationSkipped, utils.ColorError(
				"error setting environment variable: %v",
				err,
			)
		}
	}
	// get a list of env files and get their file name
	environmentFiles, err := utils.GetEnvFiles()
	if err != nil {
		return fileCreationSkipped, err
	}
	var envFromFiles []string
	for _, file := range environmentFiles {
		file = strings.ToLower(file)
		fileName := strings.ReplaceAll(file, utils.DefaultEnvFileSuffix, "")
		envFromFiles = append(envFromFiles, fileName)
	}

	// compare both values
	if !slices.Contains(envFromFiles, envFromFlag) {
		fmt.Printf("'%v.env' not found in the env directory\n", envFromFlag)
		// ask to create the file, if the file does not exist
		fmt.Printf("Create '%v.env'? (y/n)", envFromFlag)
		reader := bufio.NewReader(os.Stdin)
		responses, err := reader.ReadString('\n')
		if err != nil {
			return fileCreationSkipped, utils.ColorError("failed to read responses: %v", err)
		}
		if strings.TrimSpace(responses) == "y" || strings.TrimSpace(responses) == "Y" {
			err := CreateDefaultEnvs(&envFromFlag)
			if err != nil {
				return fileCreationSkipped, utils.ColorError(
					"failed to create environment file: %v",
					err,
				)
			}
		} else {
			fileCreationSkipped = true
			envFromFlag = utils.DefaultEnvVal
			utils.PrintGreen("Skipping file Creation")
		}
	}

	err = os.Setenv(utils.EnvKey, envFromFlag)
	utils.PrintGreen("Environment: " + os.Getenv(utils.EnvKey))
	if err != nil {
		return fileCreationSkipped, err
	}
	return fileCreationSkipped, nil
}

// trimQuotes removes double quotes " " or single quotes ' from env secrets
// and returns true if the quotes were trimmed
func trimQuotes(str string) (string, bool) {
	if len(str) >= 2 {
		if str[0] == str[len(str)-1] && (str[0] == '"' || str[0] == '\'') {
			return str[1 : len(str)-1], true
		}
	}
	return str, false
}

// handleEnvVarValue handles assessing if a value from a secret is from existing
// environment variables, if it stats with "$". If so, it pulls from there and
// defaults to empty string if not found, otherwise returns the original value
func handleEnvVarValue(val string) string {
	// OsEnvIdentifier the identifier for OS environment variables
	const OsEnvIdentifier = "$"
	if after, ok := strings.CutPrefix(val, OsEnvIdentifier); ok {
		if envVal, exists := os.LookupEnv(after); exists {
			return envVal
		}
		return ""
	}
	return val
}

// LoadEnvVars returns map of the key-value pair from the provided .env filepath
func LoadEnvVars(filePath string) (map[string]any, error) {
	hulakEnvironmentVariable := make(map[string]any)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines and comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove empty lines
		splitStr := strings.Split(line, "\n")

		// Trim spaces around the line and around "="
		var trimmedStr string
		for _, eachLine := range splitStr {
			trimmedStr = strings.TrimSpace(eachLine)
		}

		// Parse key-value pairs
		secret := strings.SplitN(trimmedStr, "=", 2)
		if len(secret) < 2 {
			// If there is no "=" or invalid format, skip
			continue
		}
		key := strings.TrimSpace(secret[0])
		val := strings.TrimSpace(secret[1])
		val = handleEnvVarValue(val)

		val, wasTrimmed := trimQuotes(val)

		// Infer value type and assign to the map
		hulakEnvironmentVariable[key] = inferType(val, wasTrimmed)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hulakEnvironmentVariable, nil
}

// Helper function to infer type of a value
func inferType(val string, wasTrimmed bool) any {
	if !wasTrimmed {
		if intValue, err := strconv.Atoi(val); err == nil {
			return intValue
		}
		if floatValue, err := strconv.ParseFloat(val, 64); err == nil {
			return floatValue
		}
		if boolValue, err := strconv.ParseBool(val); err == nil {
			return boolValue
		}
	}
	return val
}

/*
GenerateSecretsMap creates final map of environment variables and it's values
User's Choice > Global.
When user has custom env they want to use, it merges custom with global env.
Replaces global key with custom when keys repeat
*/
func GenerateSecretsMap(envFromFlag string) (map[string]any, error) {
	skipped, err := setEnvironment(envFromFlag)
	if err != nil {
		return nil, utils.ColorError("error while setting environment: %w", err)
	}

	// Retrieve the environment value
	envVal, ok := os.LookupEnv(utils.EnvKey)
	if !ok {
		return nil, utils.ColorError("error while looking up environment variable")
	}

	if envVal == "" || skipped {
		envVal = utils.DefaultEnvVal
	}

	// Load global environment variables
	globalMap, err := loadEnvFile(utils.DefaultEnvVal + utils.DefaultEnvFileSuffix)
	if err != nil {
		return nil, err
	}

	// copy instead of calling the function twice
	customMap := utils.CopyEnvMap(globalMap)

	// Load and merge custom environment variables if applicable
	envFileName := envVal + utils.DefaultEnvFileSuffix
	if globalFilePath := utils.DefaultEnvVal + utils.DefaultEnvFileSuffix; globalFilePath != envFileName {
		customMap, err = mergeCustomEnvVars(customMap, envFileName)
		if err != nil {
			return nil, err
		}
	}

	return customMap, nil
}

// Helper to load environment variables from a file
func loadEnvFile(fileName string) (map[string]any, error) {
	filePath, err := utils.CreatePath(filepath.Join(utils.EnvironmentFolder, fileName))
	if err != nil {
		return nil, utils.ColorError("error while creating file path for "+fileName, err)
	}

	envVars, err := LoadEnvVars(filePath)
	if err != nil {
		return nil, utils.ColorError("error while loading env vars from "+filePath, err)
	}

	return envVars, nil
}

// Helper to merge custom environment variables
func mergeCustomEnvVars(
	baseMap map[string]any,
	customFileName string,
) (map[string]any, error) {
	customVars, err := loadEnvFile(customFileName)
	if err != nil {
		return nil, err
	}

	maps.Copy(baseMap, customVars)

	return baseMap, nil
}
