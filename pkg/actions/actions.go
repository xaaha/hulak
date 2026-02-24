// Package actions has all the actions we use in yaml parser
package actions

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xaaha/hulak/pkg/utils"
)

// Cache structure to store both results and handle warnings
type valueCache struct {
	result any
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
	// Validate inputs
	if key == "" || fileName == "" {
		if key == "" {
			utils.PrintRed("Provide key for getValueOf action")
		} else {
			utils.PrintRed("Provide fileName/path to key for getValueOf action")
		}
		return ""
	}

	jsonResFilePath, err := resolveJSONFilePath(fileName)
	if err != nil {
		utils.PrintRed(err.Error())
		return ""
	}

	content, err := readJSONFile(jsonResFilePath)
	if err != nil {
		utils.PrintRed(err.Error())
		return ""
	}

	result, err := extractValueByKey(key, content)
	if err != nil {
		utils.PrintRed(fmt.Sprintf(
			"error while looking up the value: '%s'.\nMake sure '%s' exists and has key '%s'",
			key,
			filepath.Join(
				"...",
				utils.FileNameWithoutExtension(filepath.Dir(jsonResFilePath)),
				filepath.Base(jsonResFilePath),
			),
			key,
		))
		return ""
	}

	return result
}

// resolveJSONFilePath determines the correct JSON file path based on the input fileName
func resolveJSONFilePath(fileName string) (string, error) {
	cleanFileName := filepath.Clean(fileName)

	// Check if the fileName contains path separators or starts with ".."
	isPath := strings.Contains(cleanFileName, string(filepath.Separator)) ||
		strings.HasPrefix(cleanFileName, "..")

	if isPath {
		// Handle as a direct file path
		absPath, err := filepath.Abs(cleanFileName)
		if err != nil {
			return "", fmt.Errorf(
				"error resolving absolute path for '%s': %s",
				fileName, err.Error(),
			)
		}

		// If it's a JSON file, use it directly
		if strings.HasSuffix(cleanFileName, utils.JSON) {
			return absPath, nil
		}

		// For non-JSON files, look for _response.json
		dirPath := filepath.Dir(absPath)
		baseFileName := utils.FileNameWithoutExtension(absPath)
		return filepath.Join(dirPath, baseFileName+utils.ResponseFileName), nil
	}

	// Handle as a filename to search for
	yamlPathList, err := utils.ListMatchingFiles(cleanFileName)
	if err != nil {
		return "", fmt.Errorf(
			"error occurred while grabbing matching paths for '%s': %s",
			cleanFileName, err.Error(),
		)
	}

	if len(yamlPathList) == 0 {
		return "", fmt.Errorf("could not find matching files %s", cleanFileName)
	}

	// Handle multiple matches warning
	if len(yamlPathList) > 1 {
		utils.PrintWarning(fmt.Sprintf("Multiple '%s'. Using %s", cleanFileName, yamlPathList[0]))
	}

	singlePath := yamlPathList[0]
	if strings.HasSuffix(cleanFileName, utils.JSON) {
		return singlePath, nil
	}

	dirPath := filepath.Dir(singlePath)
	jsonBaseName := utils.FileNameWithoutExtension(singlePath) + utils.ResponseFileName
	return filepath.Join(dirPath, jsonBaseName), nil
}

// readJSONFile reads and parses a JSON file with proper locking
func readJSONFile(filePath string) (any, error) {
	// Get file-specific mutex
	fileMutex := getFileMutex(filePath)
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Check file existence under lock
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file '%s' does not exist", filePath)
	}

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"error occurred while reading the file '%s': %s",
			filepath.Base(filePath),
			err.Error(),
		)
	}

	// Parse JSON
	var content any
	err = json.Unmarshal(fileContent, &content)
	if err != nil {
		return nil, fmt.Errorf(
			"make sure %s has proper json content: %s",
			filepath.Base(filePath),
			err.Error(),
		)
	}

	return content, nil
}

// extractValueByKey extracts a value from JSON content using the provided key
func extractValueByKey(key string, content any) (any, error) {
	var result any
	var err error

	switch typedContent := content.(type) {
	case []any:
		// For array root, key must start with [
		if strings.HasPrefix(key, "[") && strings.Contains(key, "]") {
			// Wrap array in a map with empty key for LookupValue
			result, err = utils.LookupValue(key, map[string]any{
				"": typedContent,
			})
		} else {
			return "", fmt.Errorf("JSON content is an array, use [index] notation to access elements")
		}

	case map[string]any:
		// For object root
		result, err = utils.LookupValue(key, typedContent)

	default:
		return "", fmt.Errorf("unexpected JSON content format")
	}

	if err != nil {
		return "", err
	}

	// Convert float64 to int64 if it represents a whole number
	return convertNumberToProperType(result), nil
}

// convertNumberToProperType converts float64 values to int64 if they represent whole numbers
func convertNumberToProperType(v any) any {
	switch value := v.(type) {
	case float64:
		// Check if it's an integer (no decimal part)
		if value == float64(int64(value)) {
			// For small numbers that fit in int, use int
			if value >= float64(math.MinInt) && value <= float64(math.MaxInt) {
				return int(value)
			}
			// For larger numbers, use int64
			return int64(value)
		}
	case []any:
		for i, item := range value {
			value[i] = convertNumberToProperType(item)
		}
	case map[string]any:
		for k, item := range value {
			value[k] = convertNumberToProperType(item)
		}
	}
	return v
}
