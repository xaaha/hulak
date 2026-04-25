package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from the current working directory looking for
// root markers: .hulak/ (primary) or env/ (legacy), similar to how git searches for .git/.
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
		// .hulak/ is the primary marker (encrypted store)
		if DirExists(filepath.Join(dir, HiddenProjectName)) {
			return dir, true
		}
		// env/ is the legacy marker (plaintext .env files)
		if DirExists(filepath.Join(dir, EnvironmentFolder)) {
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

// GetProjectMarker returns root marker '.hulak/' of a hulak project
func GetProjectMarker() (string, error) {
	projectRoot, found := FindProjectRoot()
	if !found {
		return "", fmt.Errorf(
			"not a hulak project. run 'hulak init' to start a new one",
		)
	}

	return filepath.Join(projectRoot, ".hulak"), nil
}
