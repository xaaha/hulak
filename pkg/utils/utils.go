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

// toLowercaseMap converts all map keys to lowercase recursively
func ToLowercaseMap(m map[string]interface{}) map[string]interface{} {
	loweredMap := make(map[string]interface{})
	for k, v := range m {
		lowerKey := strings.ToLower(k)
		switch v := v.(type) {
		case map[string]interface{}:
			loweredMap[lowerKey] = ToLowercaseMap(v)
		default:
			loweredMap[lowerKey] = v
		}
	}
	return loweredMap
}
