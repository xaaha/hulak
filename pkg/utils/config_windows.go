//go:build windows

package utils

import (
	"os"
	"path/filepath"
)

// UserConfigDir returns global .config path for respective os
func UserConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ProjectName), nil
}
