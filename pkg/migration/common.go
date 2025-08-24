// Package migration migrates collection, variables, responses to hulak
// Currently it only supports postman collection and variables
package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// readJSON reads a JSON file and checks whether file exists, is empty,
// or if an error occurs while reading the file. It returns the parsed content
func readJSON(filePath string) (map[string]any, error) {
	// Check if the file exists and get its info
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("\n file does not exist %s", filePath)
	} else if err != nil {
		return nil, fmt.Errorf("\n error checking file: %w", err)
	}
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("\n file is empty: %s", filePath)
	}
	var jsonStrFile map[string]any
	jsonByteVal, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("\n error reading the JSON file: %w", err)
	}

	err = json.Unmarshal(jsonByteVal, &jsonStrFile)
	if err != nil {
		return nil, fmt.Errorf("\n error unmarshalling the file: %w", err)
	}

	return jsonStrFile, nil
}

// sanitizeKey removes all special character and
// replaces dot (.) with underscores (_).
func sanitizeKey(key string) string {
	key = strings.ReplaceAll(key, ".", "_")
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	key = re.ReplaceAllString(key, "")
	return key
}

// addDotToTemplate adds a dot after opening braces in template expressions that don't already have one.
// Example: {{value}} becomes {{.value}}, but {{.anyV}} remains unchanged
func addDotToTemplate(key string) string {
	if key == "" {
		return key
	}
	// Create a regex to find pattern {{identifier}} where there's no dot after {{
	// The regex matches {{ followed by anything except a dot or closing braces,
	// followed by any characters until }}
	re := regexp.MustCompile(`\{\{([^.\}][^\}]*)\}\}`)
	// Replace each occurrence with {{. followed by the captured content
	result := re.ReplaceAllStringFunc(key, func(match string) string {
		// Extract the content inside {{ }}
		content := match[2 : len(match)-2]
		content = sanitizeKey(content)
		return "{{." + content + "}}"
	})

	return result
}

// createMap converts a JSON string into a map[string]any
func createMap(str string) map[string]any {
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, " ", "")
	str = strings.TrimSpace(str)

	result := make(map[string]any)
	// Unmarshal the cleaned JSON string into the result map
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return nil
	}

	return result
}
