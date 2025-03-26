// Package actions has all the actions we use in yaml parser
package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

var warningTracker = make(map[string]bool)

// GetFile reads the content of a file given its relative or absolute path
func GetFile(filePath string) (string, error) {
	// Check if file path is empty
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Convert relative path to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %s: %w", filePath, err)
	}

	// Check if file exists and is readable
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try relative to working directory if absolute path doesn't exist
			workingDir, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get working directory: %w", err)
			}

			relPath := filepath.Join(workingDir, filePath)
			fileInfo, err = os.Stat(relPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("file does not exist %s ", absPath)
				}
				return "", fmt.Errorf("error accessing file %s: %w", filePath, err)
			}
			absPath = relPath
		} else {
			return "", fmt.Errorf("error accessing file %s: %w", filePath, err)
		}
	}

	// Check if it's a regular file (not a directory)
	if fileInfo.IsDir() {
		return "", fmt.Errorf("%s is a directory, not a file", filePath)
	}

	// Read the file content preserving all formatting
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Return the content as is, preserving all newlines and formatting
	return string(content), nil
}

// GetValueOf gets the value of key from a json file. If the file does not have '.json' suffix
// getValueOf looks for _response.json file automatically. If the file does not exist
func GetValueOf(key, fileName string) any {
	if key == "" && fileName == "" {
		utils.PanicRedAndExit("replaceVars.go: key and fileName can't be empty")
	}

	yamlPathList, err := utils.ListMatchingFiles(fileName)
	if err != nil {
		utils.PrintRed(fmt.Sprintf(
			"replaceVars.go: error occurred while grabbing matching paths for '%s': %s",
			fileName, err.Error(),
		))
		return ""
	}

	var singlePath string
	var jsonResFilePath string

	if strings.HasSuffix(fileName, utils.JSON) {
		// For .json files, use exact match from the list
		if len(yamlPathList) > 0 {
			// Take the first matching .json file
			singlePath = yamlPathList[0]
			jsonResFilePath = singlePath
		} else {
			utils.PrintRed("could not find matching files " + fileName)
			return ""
		}
	} else {
		// Default behavior for files without .json name
		if len(yamlPathList) > 0 {
			singlePath = yamlPathList[0]
		} else {
			utils.PrintRed("could not find matching files " + fileName)
			return ""
		}

		dirPath := filepath.Dir(singlePath)
		jsonBaseName := utils.FileNameWithoutExtension(singlePath) + utils.ResponseFileName
		jsonResFilePath = filepath.Join(dirPath, jsonBaseName)
	}

	if len(yamlPathList) > 1 {
		warningKey := fmt.Sprintf("%s_%s", fileName, singlePath)
		if !warningTracker[warningKey] {
			utils.PrintWarning(fmt.Sprintf("Multiple '%s'. Using %s", fileName, singlePath))
			warningTracker[warningKey] = true
		}
	}

	// If the file does not exist
	if _, err := os.Stat(jsonResFilePath); os.IsNotExist(err) {
		utils.PrintRed(fmt.Sprintf(
			"%s file does not exist. Either fetch the API response for '%s', or make sure the '%s' exists with '%s'. \n",
			jsonResFilePath,
			fileName,
			jsonResFilePath,
			key,
		))
		return ""
	}

	file, err := os.Open(jsonResFilePath)
	if err != nil {
		utils.PrintRed(
			fmt.Sprintf(
				"replaceVars.go: error occurred while opening the file '%s': %s",
				filepath.Base(jsonResFilePath),
				err.Error(),
			),
		)
	}
	defer file.Close()

	var fileContent map[string]any
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&fileContent)
	if err != nil {
		utils.PrintRed(
			"replaceVars.go: make sure " + filepath.Base(
				jsonResFilePath,
			) + " has proper json content" + err.Error(),
		)
	}

	result, err := utils.LookupValue(key, fileContent)
	if err != nil {
		utils.PanicRedAndExit(
			"replaceVars.go: error while looking up the value: '%s'. \nMake sure '%s' exists and has key '%s'",
			key,
			filepath.Join(
				"...",
				utils.FileNameWithoutExtension(filepath.Dir(jsonResFilePath)),
				filepath.Base(jsonResFilePath),
			),
			key,
		)
	}

	return result
}
