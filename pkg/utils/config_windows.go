//go:build windows

package utils

import (
	"errors"
	"os"
)

// UserConfigDir returns .config locaiton for windows
func UserConfigDir() (string, error) {
	dir := os.Getenv("AppData")
	if dir == "" {
		return "", errors.New("%AppData% is not defined")
	}
	return dir, nil
}
