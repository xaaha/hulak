//go:build windows

package utils

import (
	"os"
	"path/filepath"
)

// UserConfigDir returns .config locaiton for windows
func UserConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ProjectName), nil
}
