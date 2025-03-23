package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// KeyValuePair represents a generic key-value pair used in various Postman structures
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// ReadPmFile reads a Postman JSON file and returns the parsed content
func ReadPmFile(filePath string) (map[string]any, error) {
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

// AddDotToTemplate adds a dot after opening braces in template expressions that don't already have one.
// Example: {{value}} becomes {{.value}}, but {{.anyV}} remains unchanged
func AddDotToTemplate(s string) string {
	if s == "" {
		return s
	}
	// Create a regex to find pattern {{identifier}} where there's no dot after {{
	// The regex matches {{ followed by anything except a dot or closing braces,
	// followed by any characters until }}
	re := regexp.MustCompile(`\{\{([^.\}][^\}]*)\}\}`)
	// Replace each occurrence with {{. followed by the captured content
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the content inside {{ }}
		content := match[2 : len(match)-2]
		return "{{." + content + "}}"
	})

	return result
}
