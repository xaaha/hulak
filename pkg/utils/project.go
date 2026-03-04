package utils

import (
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from the current working directory looking for
// the env/ directory that marks a hulak project root, similar to how git
// searches for .git/. It stops at the user's home directory.
// Returns the project root path and true if found, or CWD and false otherwise.
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
	for {
		candidate := filepath.Join(dir, EnvironmentFolder)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if home != "" && dir == home {
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
