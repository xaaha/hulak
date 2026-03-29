//go:build !windows

package utils

import (
	"errors"
	"os"
	"path/filepath"
)

// UserConfigDir returns .config locaiton for unix
func UserConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", errors.New("path in $XDG_CONFIG_HOME is relative")
		}
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
	}

	return filepath.Join(home, ".config", ProjectName), nil
}
