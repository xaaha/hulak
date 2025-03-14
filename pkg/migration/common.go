package migration

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

// ReadPmFile reads a Postman JSON file and returns the parsed content
func ReadPmFile(filePath string) (map[string]any, error) {
	// Check if the file exists and get its info
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist %s", filePath)
	} else if err != nil {
		return nil, fmt.Errorf("error checking file: %w", err)
	}
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("file is empty: %s", filePath)
	}
	var jsonStrFile map[string]any
	jsonByteVal, err := os.ReadFile(filePath)
	if err != nil {
		return nil, utils.ColorError("error reading the JSON file: %w", err)
	}

	err = json.Unmarshal(jsonByteVal, &jsonStrFile)
	if err != nil {
		return nil, utils.ColorError("error unmarshalling the file: %w", err)
	}

	return jsonStrFile, nil
}
