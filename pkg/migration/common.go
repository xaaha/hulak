package migration

import (
	"encoding/json"
	"fmt"
	"os"
)

// KeyValuePair represents a generic key-value pair used in various Postman structures
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
