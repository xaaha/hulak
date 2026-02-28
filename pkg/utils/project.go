package utils

import "os"

// IsHulakProject checks if the current working directory is set up as a hulak
// project by verifying that the env/ directory exists. This is analogous to
// how git checks for a .git/ directory.
func IsHulakProject() bool {
	envPath, err := CreatePath(EnvironmentFolder)
	if err != nil {
		return false
	}
	info, err := os.Stat(envPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}
