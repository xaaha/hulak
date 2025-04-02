// Package utils has all the utils required for hulak, including but not limited to
// CreateFilePath, CreateDir, CreateFiles, ListMatchingFiles, MergeMaps and more..
package utils

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// CreatePath creates and returns file or directory path by joining the project root with provided filePath
func CreatePath(filePath string) (string, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", err
	}
	finalFilePath := filepath.Join(projectRoot, filePath)

	return finalFilePath, nil
}

// SanitizeDirPath cleans up the directory path to avoid traversals
func SanitizeDirPath(dirPath string) (string, error) {
	cleanPath := filepath.Clean(dirPath)
	if !filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("invalid directory path: %s", dirPath)
	}
	return cleanPath, nil
}

// SanitizeFilePath cleans and ensures that the file is within the dir
func SanitizeFilePath(dirPath, filePath string) (string, error) {
	cleanDirPath, err := SanitizeDirPath(dirPath)
	if err != nil {
		return "", err
	}

	// Clean the file path and ensure it's within the directory
	cleanFilePath := filepath.Join(cleanDirPath, filepath.Clean(filePath))
	if !strings.HasPrefix(cleanFilePath, cleanDirPath) {
		return "", fmt.Errorf("file path %s is outside of the directory %s", filePath, dirPath)
	}

	return cleanFilePath, nil
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
		PrintRed("Error creating directory " + CrossMark)
		return err
	}
	PrintGreen("Created directory " + CheckMark)
	return nil
}

// CreateFile checks for the existence of a file at the given filePath,
// and creates it if it does not exist.
func CreateFile(filePath string) error {
	fileName := filepath.Base(filePath)
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			PrintRed(fmt.Sprintf("Error creating '%s': %s", fileName, CrossMark))
			return err
		}
		defer file.Close()
		PrintGreen(fmt.Sprintf("Created '%s': %s", fileName, CheckMark))
	} else if err != nil {
		PrintRed(fmt.Sprintf("Error checking '%s': %v", fileName, err))
		return err
	} else {
		if info.IsDir() {
			return fmt.Errorf("cannot create file '%s': path is a directory", filePath)
		}
		if info.Mode().IsRegular() {
			PrintWarning(fmt.Sprintf("File '%s' already exists.", filePath))
		}
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

// ListMatchingFiles Searches for files matching the "matchFile" name (case-insensitive, .yaml/.yml or .json only)
// in the specified directory and its subdirectories. If no directory is specified, it starts from the project root.
// Skips all hidden folders like `.git`, `.vscode` or `.random` folder during traversal.
// Returns slice of matched file path and an error if no matching files are found or if there are file system errors.
func ListMatchingFiles(matchFile string, initialPath ...string) ([]string, error) {
	if matchFile == "" {
		return nil, ColorError("#utils.go: matchFile can't be empty")
	}

	fileExtensions := []string{YAML, YML, JSON}
	for _, ext := range fileExtensions {
		matchFile = strings.TrimSuffix(matchFile, ext)
	}
	matchFile = strings.ToLower(matchFile)

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

	var result []string

	// check if we are explicitly starting in a hidden dir
	explicitlyHidden := string(filepath.Base(startPath)[0]) == "."

	err := filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories only if we're not explicitly starting in one
		// And only skip directories that aren't the explicitly specified one
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && !explicitlyHidden && path != startPath {
			return filepath.SkipDir
		}

		// Process files
		if !d.IsDir() {
			fileName := strings.ToLower(d.Name())
			for _, ext := range fileExtensions {
				if strings.HasSuffix(fileName, ext) {
					baseName := strings.TrimSuffix(fileName, ext)
					if matchFile == baseName {
						result = append(result, path)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, ColorError("error walking directory "+startPath, err)
	}

	if len(result) == 0 {
		return nil, ColorError(
			"no files with matching name " + matchFile + " found in " + startPath,
		)
	}
	return result, nil
}

// FileNameWithoutExtension takes in filepath and returns the name of the file
func FileNameWithoutExtension(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
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
