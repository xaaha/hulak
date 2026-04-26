//go:build !windows

package utils

import (
	"errors"
	"os"
	"path/filepath"
)

// UserConfigDir returns the per-app config directory for hulak: either
// $XDG_CONFIG_HOME/hulak or ~/.config/hulak. Always app-scoped — never
// returns the bare config root.
func UserConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", errors.New("path in $XDG_CONFIG_HOME is relative")
		}
		return filepath.Join(dir, ProjectName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
	}

	return filepath.Join(home, ".config", ProjectName), nil
}
