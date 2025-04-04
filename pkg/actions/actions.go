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

	cleanPath := filepath.Clean(filePath)

	// Get current working directory as the base allowed directory
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Try to resolve as absolute path first
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %s: %w", cleanPath, err)
	}

	// Check if file exists and is readable
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try relative to working directory if absolute path doesn't exist
			relPath := filepath.Join(workingDir, cleanPath)
			fileInfo, err = os.Stat(relPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("file does not exist %s", absPath)
				}
				return "", fmt.Errorf("error accessing file %s: %w", filePath, err)
			}
			absPath = relPath
		} else {
			return "", fmt.Errorf("error accessing file %s: %w", filePath, err)
		}
	}

	// ensure the file is within the working directory or explicitly allowed directories
	if !strings.HasPrefix(absPath, workingDir) {
		// If you want to allow specific directories outside the working dir, add checks here
		// For example, checking if it's in an allowed config directory
		return "", fmt.Errorf(
			"access denied: file path %s is outside the allowed directory",
			filePath,
		)
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

// GetValueOf gets the value of key from a json file.
// If a relative/absolute path is provided (e.g., "../../test.json"), it uses that exact path.
// Otherwise, it searches for matching files and uses _response.json suffix for non-JSON files.
func GetValueOf(key, fileName string) any {
	if key == "" || fileName == "" {
		if key == "" {
			utils.PrintRed("provide 'key' for getValueOf action")
		} else {
			utils.PrintRed("provide 'fileName' for getValueOf action")
		}
		return ""
	}

	cleanFileName := filepath.Clean(fileName)
	var jsonResFilePath string

	// Check if the fileName contains path separators or starts with ".."
	isPath := strings.Contains(cleanFileName, string(filepath.Separator)) ||
		strings.HasPrefix(cleanFileName, "..")

	if isPath {
		// Handle as a direct file path
		absPath, err := filepath.Abs(cleanFileName)
		if err != nil {
			utils.PrintRed(fmt.Sprintf(
				"error resolving absolute path for '%s': %s",
				fileName, err.Error(),
			))
			return ""
		}

		// If it's a JSON file, use it directly
		if strings.HasSuffix(cleanFileName, utils.JSON) {
			jsonResFilePath = absPath
		} else {
			// For non-JSON files, look for _response.json
			dirPath := filepath.Dir(absPath)
			baseFileName := utils.FileNameWithoutExtension(absPath)
			jsonResFilePath = filepath.Join(dirPath, baseFileName+utils.ResponseFileName)
		}
	} else {
		yamlPathList, err := utils.ListMatchingFiles(cleanFileName)
		if err != nil {
			utils.PrintRed(fmt.Sprintf(
				"error occurred while grabbing matching paths for '%s': %s",
				cleanFileName, err.Error(),
			))
			return ""
		}

		if len(yamlPathList) == 0 {
			utils.PrintRed("could not find matching files " + cleanFileName)
			return ""
		}

		// Handle multiple matches warning
		if len(yamlPathList) > 1 {
			warningKey := fmt.Sprintf("%s_%s", cleanFileName, yamlPathList[0])
			if !warningTracker[warningKey] {
				utils.PrintWarning(fmt.Sprintf("Multiple '%s'. Using %s", cleanFileName, yamlPathList[0]))
				warningTracker[warningKey] = true
			}
		}

		singlePath := yamlPathList[0]
		if strings.HasSuffix(cleanFileName, utils.JSON) {
			jsonResFilePath = singlePath
		} else {
			dirPath := filepath.Dir(singlePath)
			jsonBaseName := utils.FileNameWithoutExtension(singlePath) + utils.ResponseFileName
			jsonResFilePath = filepath.Join(dirPath, jsonBaseName)
		}
	}

	// Check if the JSON file exists
	if _, err := os.Stat(jsonResFilePath); os.IsNotExist(err) {
		if isPath {
			utils.PrintRed(fmt.Sprintf(
				"File '%s' does not exist",
				jsonResFilePath,
			))
		} else {
			utils.PrintRed(fmt.Sprintf(
				"%s file does not exist. Either fetch the API response for '%s', or make sure the '%s' exists with '%s'.\n",
				jsonResFilePath,
				cleanFileName,
				jsonResFilePath,
				key,
			))
		}
		return ""
	}

	// Read and parse the JSON file
	file, err := os.Open(jsonResFilePath)
	if err != nil {
		utils.PrintRed(fmt.Sprintf(
			"error occurred while opening the file '%s': %s",
			filepath.Base(jsonResFilePath),
			err.Error(),
		))
		return ""
	}
	defer file.Close()

	var fileContent map[string]any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&fileContent); err != nil {
		utils.PrintRed(
			"make sure " + filepath.Base(jsonResFilePath) +
				" has proper json content: " + err.Error(),
		)
		return ""
	}

	// Look up the value in the JSON content
	result, err := utils.LookupValue(key, fileContent)
	if err != nil {
		utils.PanicRedAndExit(
			"error while looking up the value: '%s'.\nMake sure '%s' exists and has key '%s'",
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
