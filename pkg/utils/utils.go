package utils

import (
	"fmt"
	"os"
	"path"
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

// Searches for files matching the "matchFile" name (case-insensitive, .yaml/.yml only)
// in the specified directory and its subdirectories. If no directory is specified, it starts from the project root.
// Skips all hidden folders like `.git`, `.vscode` or `.random` folder during traversal.
// Returns slice of mathed file path and an error if no matching files are found or if there are file system errors.
func ListMatchingFiles(matchFile string, initialPath ...string) ([]string, error) {
	matchFile = strings.ToLower(matchFile)
	var result []string

	// Get the initial path to start the search
	initAbsFp, err := CreateFilePath("")
	if err != nil {
		return nil, ColorError("error getting initial file path: %w", err)
	}

	var startPath string
	if len(initialPath) == 0 {
		startPath = initAbsFp
	} else {
		startPath = initialPath[0]
	}

	// Read the contents of the starting directory
	dirContents, err := os.ReadDir(startPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", startPath, err)
	}

	filePattern := [2]string{".yaml", ".yml"}

	for _, val := range dirContents {
		// Skip `.git` or `.vscode` directory
		if val.IsDir() && strings.HasPrefix(val.Name(), ".") {
			continue
		}

		// Process files
		if !val.IsDir() {
			lowerName := strings.ToLower(
				val.Name(),
			)
			for _, ext := range filePattern {
				if strings.HasSuffix(lowerName, ext) {
					yamlFile := strings.TrimSuffix(lowerName, ext)
					if matchFile == yamlFile {
						matchingFp := filepath.Join(startPath, val.Name())
						result = append(result, matchingFp)
					}
				}
			}
		}

		// Recurse into subdirectories, but skip hidden folders `.git` `.vscode`
		if val.IsDir() {
			subDirPath := filepath.Join(startPath, val.Name())
			matches, err := ListMatchingFiles(matchFile, subDirPath)
			if err != nil {
				dir := path.Base(subDirPath)
				return nil, fmt.Errorf("\n error processing subdirectory %s: %w", dir, err)
			}
			result = append(result, matches...)
		}
	}

	if len(result) == 0 {
		err := ColorError(
			"\n no files with matching name '" + matchFile + "' found in path: " + initAbsFp,
		)
		return nil, err
	}

	return result, nil
}
