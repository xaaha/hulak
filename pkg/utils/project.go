package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from the current working directory looking for
// the root marker, .hulak/, similar to how git searches for .git/.
// It stops at the user's home directory. Returns the project root path and true if found, or user home dir and false otherwise.
func FindProjectRoot() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	dir := cwd
	for home == "" || dir != home {
		candidate := filepath.Join(dir, EnvironmentFolder)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return cwd, false
}

// IsHulakProject checks whether an env/ directory exists anywhere between the
// current working directory and the user's home directory.
func IsHulakProject() bool {
	_, found := FindProjectRoot()
	return found
}

// GetProjectMarker returns root marker '.hulak/'
func GetProjectMarker() (string, error) {
	projectRoot, found := FindProjectRoot()
	if !found {
		return "", fmt.Errorf(
			"not a hulak project. run 'hulak init' to start a new one",
		)
	}

	return filepath.Join(projectRoot, ".hulak"), nil
}
