package utils

import (
	"fmt"
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

// Copies the Environment map[string]interface{} and returns a map[string]string
// EnvMap is a simple JSON without any nested properties.
// Mostly used for goroutines.
// Copies the Environment map[string]interface{} and returns a copy as map[string]interface{}.
// EnvMap is a simple JSON without any nested properties.
func CopyEnvMap(original map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, val := range original {
		result[key] = val // Direct copy
	}
	return result
}

// Searches for files matching the "matchFile" name (case-insensitive, .yaml/.yml or .json only)
// in the specified directory and its subdirectories. If no directory is specified, it starts from the project root.
// Skips all hidden folders like `.git`, `.vscode` or `.random` folder during traversal.
// Returns slice of matched file path and an error if no matching files are found or if there are file system errors.
func ListMatchingFiles(matchFile string, initialPath ...string) ([]string, error) {
	matchFile = strings.ToLower(matchFile)
	var result []string

	initAbsFp, err := CreateFilePath("")
	if err != nil {
		return nil, fmt.Errorf("error getting initial file path: %w", err)
	}

	var startPath string
	if len(initialPath) == 0 {
		startPath = initAbsFp
	} else {
		startPath = initialPath[0]
	}

	dirContents, err := os.ReadDir(startPath)
	if err != nil {
		return nil, ColorError("error reading directory "+startPath, err)
	}

	filePattern := [3]string{YAML, YML, JSON}

	for _, val := range dirContents {
		// Skip hidden directories
		if val.IsDir() && strings.HasPrefix(val.Name(), ".") {
			continue
		}

		// Process files
		if !val.IsDir() {
			lowerName := strings.ToLower(val.Name())
			for _, ext := range filePattern {
				if strings.HasSuffix(lowerName, ext) {
					ymlOrJsonFile := strings.TrimSuffix(lowerName, ext)
					if matchFile == ymlOrJsonFile {
						matchingFp := filepath.Join(startPath, val.Name())
						result = append(result, matchingFp)
					}
				}
			}
		}

		// Process subdirectories
		if val.IsDir() {
			subDirPath := filepath.Join(startPath, val.Name())
			matches, err := ListMatchingFiles(matchFile, subDirPath)
			if err != nil && !isNoMatchingFileError(err) {
				PrintRed("Skipping subdirectory" + val.Name() + "due to error: \n" + err.Error())
				continue
			}
			result = append(result, matches...)
		}
	}

	if len(result) == 0 {
		return []string{}, ColorError(
			"no files with matching name " + matchFile + " found in " + initAbsFp,
		)
	}
	return result, nil
}

// isNoMatchingFileError determines if the error is related to no matching files found.
func isNoMatchingFileError(err error) bool {
	return strings.Contains(err.Error(), "no files with matching name")
}

// takes in filepath and returns the name of the file
func FileNameWithoutExtension(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
