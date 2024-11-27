package utils

import (
	"os"
	"path/filepath"
	"strings"
)

type Utilities struct{}

// Creates and returns file path by joining the project root with provided filePath
func CreateFilePath(filePath string) (string, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", err
	}
	finalFilePath := filepath.Join(projectRoot, filePath)

	return finalFilePath, nil
}

// Get a list of environment file names from the env folder
func (u *Utilities) GetEnvFiles() ([]string, error) {
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
			lowerCasedEnvFromFile := strings.ToLower(fileOrDir.Name())
			environmentFiles = append(environmentFiles, lowerCasedEnvFromFile)
		}
	}
	return environmentFiles, nil
}

// converts all keys in a map to lowercase recursively
func ConvertKeysToLowerCase(dict map[string]interface{}) map[string]interface{} {
	loweredMap := make(map[string]interface{})
	for key, val := range dict {
		lowerKey := strings.ToLower(key)
		switch almostFinalValue := val.(type) {
		case map[string]interface{}:
			loweredMap[lowerKey] = ConvertKeysToLowerCase(almostFinalValue)
		default:
			loweredMap[lowerKey] = almostFinalValue
		}
	}
	return loweredMap
}

// Copies the Environment map[string]string and returns a CopyEnvMap
// EnvMap is a simple json without any nested properties.
// Mostly used for go routines
func CopyEnvMap(original map[string]string) map[string]string {
	result := map[string]string{}
	for key, val := range original {
		result[key] = val
	}
	return result
}
