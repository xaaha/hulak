package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from the current working directory looking for
// root markers: .hulak/ (primary) or env/ (legacy), similar to how git searches for .git/.
// It stops at the user's home directory. Returns the project root path and true if found,
// or the original current working directory and false otherwise.
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
		if IsProjectDir(dir) {
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

// IsProjectDir reports whether dir is a hulak project root: it holds a .hulak/
// (encrypted vault) or env/ (legacy plaintext) marker directory.
func IsProjectDir(dir string) bool {
	return DirExists(filepath.Join(dir, HiddenProjectName)) ||
		DirExists(filepath.Join(dir, EnvironmentFolder))
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
