package utils

import (
	"os"
	"path/filepath"
)

// Creates and returns file path by joining the project root with provided fileName
func CreateFilePath(fileName string) (string, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(projectRoot, fileName)

	return filePath, nil
}
