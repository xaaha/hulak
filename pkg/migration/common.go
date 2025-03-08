package migration

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadPmFile reads a Postman JSON file and returns the parsed content
func ReadPmFile(filePath string) (map[string]any, error) {
	var jsonStrFile map[string]any
	jsonByteVal, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading the JSON file: %w", err)
	}

	err = json.Unmarshal(jsonByteVal, &jsonStrFile)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling the file: %w", err)
	}

	return jsonStrFile, nil
}
