// Package actions has all the actions we use in yaml parser
package actions

import (
	"encoding/base64"
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

// BasicAuth takes a username and password, joins them with a colon,
// base64-encodes the result, and returns the full header value "Basic <encoded>".
// Both arguments are treated as plain strings — use .env template vars for secrets.
func BasicAuth(username, password string) string {
	credentials := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encoded
}

// GetFile reads the content of a file referenced by a {{getFile}} template.
// Resolution is delegated to utils.ResolveProjectFile: relative paths are
// project-root-relative (never cwd-relative), absolute paths are used as-is,
// and either way the file must live inside the project root.
func GetFile(filePath string) (string, error) {
	absPath, err := utils.ResolveProjectFile(filePath)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return string(content), nil
}

func getFileMutex(filePath string) *sync.Mutex {
	mutex, _ := fileOpsMutex.LoadOrStore(filePath, &sync.Mutex{})
	return mutex.(*sync.Mutex)
}

// processValueOf processes GetValueOf action — returns the value at key from
// the resolved JSON file, or "" if anything goes wrong. Errors are printed to
// stderr; we can't return them because this is invoked as a template function
// whose signature is fixed at func(...) any. Stdout stays clean so any
// downstream `$(...)` capture still gets clean program output.
func processValueOf(key, fileName string) any {
	// Validate inputs
	if key == "" || fileName == "" {
		if key == "" {
			utils.PrintErrorStderr(
				fmt.Sprintf("provide key for %s action", utils.TemplateFuncGetValueOf),
			)
		} else {
			utils.PrintErrorStderr(
				fmt.Sprintf(
					"provide fileName/path to key for %s action",
					utils.TemplateFuncGetValueOf,
				),
			)
		}
		return ""
	}

	jsonResFilePath, err := resolveJSONFilePath(fileName)
	if err != nil {
		utils.PrintErrorStderr(err.Error())
		return ""
	}

	content, err := readJSONFile(jsonResFilePath)
	if err != nil {
		utils.PrintErrorStderr(err.Error())
		return ""
	}

	result, err := extractValueByKey(key, content)
	if err != nil {
		utils.PrintErrorStderr(fmt.Sprintf(
			"looking up value '%s': make sure '%s' exists and has key '%s'",
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
		utils.PrintWarningStderr(
			fmt.Sprintf("multiple '%s' files; using %s", cleanFileName, yamlPathList[0]),
		)
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
