// Package actions has all the actions we use in yaml parser
package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xaaha/hulak/pkg/utils"
)

var warningTracker = make(map[string]bool)

// Cache structure to store both results and handle warnings
type valueCache struct {
	result any
	warned bool
	exists bool
}

// Global cache map with thread-safe access
var (
	valuesCacheMutex sync.RWMutex
	valuesCache      = make(map[string]valueCache)

	// Add file operation mutex
	fileOpsMutex sync.Map
)

// GetValueOf gets the value of key from a json file with caching
func GetValueOf(key, fileName string) any {
	// Create cache key combining file and key
	cacheKey := fmt.Sprintf("%s:%s", fileName, key)

	// Check cache first
	valuesCacheMutex.RLock()
	if cache, exists := valuesCache[cacheKey]; exists {
		valuesCacheMutex.RUnlock()
		return cache.result
	}
	valuesCacheMutex.RUnlock()

	// If not in cache, acquire write lock and process
	valuesCacheMutex.Lock()
	defer valuesCacheMutex.Unlock()

	// Double-check pattern in case another goroutine cached while we waited
	if cache, exists := valuesCache[cacheKey]; exists {
		return cache.result
	}

	// Process the file and get result
	result := processValueOf(key, fileName)

	// Cache the result
	valuesCache[cacheKey] = valueCache{
		result: result,
		exists: true,
	}

	return result
}

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

func getFileMutex(filePath string) *sync.Mutex {
	mutex, _ := fileOpsMutex.LoadOrStore(filePath, &sync.Mutex{})
	return mutex.(*sync.Mutex)
}

// processValueOf processes GetValueOf action
// gets the value of key from a json file.
// If a relative/absolute path is provided (e.g., "../../test.json"), it uses that exact path.
// Otherwise, it searches for matching files and uses _response.json suffix for non-JSON files.
func processValueOf(key, fileName string) any {
	if key == "" && fileName == "" {
		if key == "" {
			utils.PrintRed("Privide key for getValueOf action")
		} else {
			utils.PrintRed("Privide fileName/path to key for getValueOf action")
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
		// read-only operation for concurrent access
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
			utils.PrintWarning(fmt.Sprintf("Multiple '%s'. Using %s", cleanFileName, yamlPathList[0]))
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

	// Get file-specific mutex
	fileMutex := getFileMutex(jsonResFilePath)
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Check file existence under lock to prevent race conditions
	if _, err := os.Stat(jsonResFilePath); os.IsNotExist(err) {
		if isPath {
			utils.PrintRed(fmt.Sprintf("File '%s' does not exist", jsonResFilePath))
		} else {
			utils.PrintRed(fmt.Sprintf(
				"%s file does not exist. Either fetch the API response for '%s', or make sure the '%s' exists with '%s'.\n",
				jsonResFilePath, cleanFileName, jsonResFilePath, key,
			))
		}
		return ""
	}

	// Read file under lock
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

	// Value lookup is thread-safe as fileContent is local to this function
	result, err := utils.LookupValue(key, fileContent)
	if err != nil {
		msg := fmt.Sprintf(
			"error while looking up the value: '%s'.\nMake sure '%s' exists and has key '%s'",
			key,
			filepath.Join(
				"...",
				utils.FileNameWithoutExtension(filepath.Dir(jsonResFilePath)),
				filepath.Base(jsonResFilePath),
			),
			key,
		)
		utils.PrintRed(msg)
	}

	return result
}
