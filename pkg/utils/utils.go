// Package utils has all the utils required for hulak, including but not limited to
// CreateFilePath, CreateDir, CreateFiles, ListMatchingFiles, MergeMaps and more..
package utils

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Username returns the current OS username. Uses os/user.Current() which
// works in containers and cron where env vars may be stripped.
// Falls back to "owner" if the username cannot be determined.
func Username() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "owner"
}

// CreatePath creates and returns file or directory path by joining the project root with provided filePath.
// It walks up from the current directory to find the hulak project root (env/ directory).
// If no project root is found, it falls back to the current working directory.
func CreatePath(filePath string) (string, error) {
	projectRoot, _ := FindProjectRoot()
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		projectRoot = cwd
	}
	return filepath.Join(projectRoot, filePath), nil
}

func SanitizeFileName(s string) string {
	// Replace common separators with underscore
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	// Remove any other invalid characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	s = reg.ReplaceAllString(s, "")

	// Ensure it doesn't start with a number (optional, but good practice)
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "r_" + s
	}

	return strings.ToLower(s)
}

// SanitizeDirPath cleans up the directory path to avoid traversals
func SanitizeDirPath(dirPath string) (string, error) {
	cleanPath := filepath.Clean(dirPath)
	if cleanPath == "" {
		cleanPath = "."
	}
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("error converting to absolute path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("error accessing path %s: %w", dirPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", dirPath)
	}
	return absPath, nil
}

// CreateDir checks for the existence of a directory at the given path,
// and creates it with permissions 0755 if it does not exist.
func CreateDir(dirPath string) error {
	info, err := os.Stat(dirPath)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path '%s' exists but is a file", dirPath)
		}
		return nil // Dir already exists
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err := os.Mkdir(dirPath, DirPer); err != nil {
		return fmt.Errorf("creating directory %s: %w", dirPath, err)
	}
	PrintSuccessStderr("Created directory " + dirPath)
	return nil
}

// CreateFile checks for the existence of a file at the given filePath,
// and creates it if it does not exist.
func CreateFile(filePath string) error {
	info, err := os.Stat(filePath)

	// File does not exist → create it
	if os.IsNotExist(err) {
		f, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer func() {
			cerr := f.Close()
			if cerr != nil && err == nil {
				err = cerr
			}
		}()
		return nil
	}

	// An actual error other than "not exists"
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("cannot create file '%s': path is a directory", filePath)
	}

	return nil
}

// GetEnvFiles returns a list of environment file names from the env folder
func GetEnvFiles() ([]string, error) {
	var environmentFiles []string
	// get a list of envFileName
	envPath, err := CreatePath(EnvironmentFolder)
	if err != nil {
		return environmentFiles, err
	}
	contents, err := os.ReadDir(envPath)
	if err != nil {
		return environmentFiles, err
	}

	// discard any folder in the env directory
	for _, fileOrDir := range contents {
		if !fileOrDir.IsDir() {
			lowerCasedEnvFromFile := strings.ToLower(fileOrDir.Name())
			environmentFiles = append(environmentFiles, lowerCasedEnvFromFile)
		}
	}
	return environmentFiles, nil
}

// ConvertKeysToLowerCase converts all keys in a map to lowercase recursively
// except "variables" as Graphql variables is case-sensitive
func ConvertKeysToLowerCase(dict map[string]any) map[string]any {
	loweredMap := make(map[string]any)
	for key, val := range dict {
		// for graphql variables are case sensitive
		if key == "variables" {
			loweredMap[key] = val
			continue
		}
		lowerKey := strings.ToLower(key)
		// If val is a map and the key isn't "variables", process it recursively.
		switch almostFinalValue := val.(type) {
		case map[string]any:
			loweredMap[lowerKey] = ConvertKeysToLowerCase(almostFinalValue)
		default:
			loweredMap[lowerKey] = almostFinalValue
		}
	}
	return loweredMap
}

// CopyEnvMap Copies the Environment map[string]any and returns a map[string]string
// EnvMap is a simple JSON without any nested properties. Mostly used for goroutines.
func CopyEnvMap(original map[string]any) map[string]any {
	result := make(map[string]any)
	maps.Copy(result, original)
	return result
}

// ListMatchingFiles searches for files matching the "matchFile" name (case-insensitive, .yaml/.yml or .json only)
// in the specified directory and its subdirectories. If no directory is specified, it starts from the project root.
// Includes all directories in traversal, including hidden ones.
// Returns slice of matched file paths and an error if no matching files are found or if there are file system errors.
func ListMatchingFiles(matchFile string, initialPath ...string) ([]string, error) {
	if matchFile == "" {
		return nil, errors.New(ErrFileSearchEmpty)
	}

	fileExtensions := []string{YAML, YML, JSON}

	// Get base name by removing any supported extension. The .hk marker is
	// stripped after the yaml/yml suffix so "login.hk.yaml" reduces to "login"
	// and `-f login` finds it (not just `-f login.hk`).
	baseName := matchFile
	for _, ext := range fileExtensions {
		baseName = strings.TrimSuffix(baseName, ext)
	}
	baseName = strings.TrimSuffix(strings.ToLower(baseName), ProjectExt)

	// Determine the start path
	startPath := ""
	if len(initialPath) == 0 {
		var err error
		startPath, err = CreatePath("")
		if err != nil {
			return nil, fmt.Errorf("error getting initial file path: %w", err)
		}
	} else {
		startPath = initialPath[0]
	}

	// List all files in the directory
	allFiles, err := ListFiles(startPath)
	if err != nil {
		return nil, err
	}

	// Filter files by matching base name
	var result []string
	for _, filePath := range allFiles {
		fileName := strings.ToLower(filepath.Base(filePath))

		// Check if the file has a supported extension
		hasMatchingExtension := false
		for _, ext := range fileExtensions {
			if strings.HasSuffix(fileName, ext) {
				hasMatchingExtension = true
				break
			}
		}

		// If it has a supported extension, compare base names
		if hasMatchingExtension {
			fileBaseName := fileName
			for _, ext := range fileExtensions {
				fileBaseName = strings.TrimSuffix(fileBaseName, ext)
			}
			fileBaseName = strings.TrimSuffix(fileBaseName, ProjectExt)

			// If base names match, add to results
			if fileBaseName == baseName {
				result = append(result, filePath)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf(
			"no files with matching name '%s' found in '%s'",
			matchFile,
			startPath,
		)
	}

	return result, nil
}

// FileNameWithoutExtension takes in filepath and returns the name of the file
func FileNameWithoutExtension(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// HasTildePrefix reports whether path uses ~ as a home-directory reference that
// ExpandPath will expand: exactly "~", or "~/" (plus the native "~\" on
// Windows). A path like "~data" is a literal relative path, not a home
// reference, so callers deciding whether a path is explicit must use this same
// test rather than a bare "~" prefix check.
func HasTildePrefix(path string) bool {
	return path == "~" ||
		strings.HasPrefix(path, "~/") ||
		strings.HasPrefix(path, "~"+string(os.PathSeparator))
}

// ExpandPath resolves a leading ~ to the home directory and returns an
// absolute, cleaned path.
//
// ~ is not native on Windows — hulak supports it as a convention. A separator
// must follow (~/ always, plus the native ~\ on Windows) so a real file named
// "~data" is left alone. os.UserHomeDir yields %USERPROFILE% on Windows.
func ExpandPath(path string) (string, error) {
	if HasTildePrefix(path) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding ~: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}

// MergeMaps merges the secondary map into the main map.
// If keys are repeated, values from the secondary map replace those in the main map.
func MergeMaps(main, sec map[string]string) map[string]string {
	if main == nil {
		main = make(map[string]string)
	}
	if sec == nil {
		return main
	}
	// Merge sec map into main map
	maps.Copy(main, sec)
	return main
}

// stat checks if a file exists and is accessible at the given path
func stat(path string) (os.FileInfo, error) {
	return os.Stat(filepath.Clean(path))
}

// Returns true if the file exists and is readable, false otherwise
func FileExists(path string) bool {
	info, err := stat(path)
	return err == nil && !info.IsDir()
}

func DirExists(path string) bool {
	info, err := stat(path)
	return err == nil && info.IsDir()
}

// AtomicWriteFile writes data to path via a temporary file + rename, so a
// reader of path sees either the previous contents or the new ones, never a
// partial write. Not crash-safe: there is no fsync, so a power loss can leave
// the rename durable and the data not. Creates parent directories with dirPerm
// if they don't exist.
func AtomicWriteFile(path string, data []byte, filePerm, dirPerm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Unique temp name, not a fixed path+".tmp": two goroutines writing the
	// same destination would otherwise share one temp file and each delete it
	// out from under the other, so both renames fail. A directory run reaches
	// that with sibling request files whose stems match (login.yaml and
	// login.yml both save to login_response.json).
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}
	// os.CreateTemp always creates at 0600; filePerm is the caller's intent.
	if err := tmp.Chmod(filePerm); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := renameWithRetry(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize file: %w", err)
	}
	return nil
}

// renameWithRetry works around Windows, where a rename over a file another
// process holds open fails with a sharing violation: Go opens files for
// reading without FILE_SHARE_DELETE, so a concurrent reader, an editor tab or
// a virus scanner blocks the replace. The reader's window is short, so a few
// retries clear it. On Unix the first attempt succeeds and this costs nothing.
func renameWithRetry(from, to string) error {
	var err error
	for attempt := range 4 {
		if err = os.Rename(from, to); err == nil {
			return nil
		}
		if attempt < 3 {
			time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
		}
	}
	return err
}
