package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Option pattern for configurable function behavior
type listFilesOptions struct {
	respectDotDirs bool
	skipDirs       []string
}

// ListFilesOption lists the options we should skip
type ListFilesOption func(*listFilesOptions)

// WithSkipDirs specifies directories to skip
func WithSkipDirs(dirs []string) ListFilesOption {
	return func(opts *listFilesOptions) {
		opts.skipDirs = dirs
	}
}

// WithRespectDotDirs controls whether to traverse dot directories
func WithRespectDotDirs(respect bool) ListFilesOption {
	return func(opts *listFilesOptions) {
		opts.respectDotDirs = respect
	}
}

// ListFiles generates all .yaml, .yml, or .json files in a directory, with configurable directory exclusion
// Files are added as they are discovered so it does not guarantee any files are run before the other
func ListFiles(dirPath string, options ...ListFilesOption) ([]string, error) {
	// Default folders to skip during file listing
	opts := listFilesOptions{
		skipDirs:       []string{"node_modules", ".git", ".svn", ".hg", ".idea", ".vscode"},
		respectDotDirs: true,
	}

	// Apply options
	for _, option := range options {
		option(&opts)
	}

	fileExtensions := []string{YAML, YML, JSON}
	result := make([]string, 0)

	if dirPath == "" {
		dirPath = "."
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(dirPath)

	// Check if directory exists and is accessible
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("error accessing directory '%s': %w", dirPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", dirPath)
	}

	// Convert to absolute path for consistency
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("error resolving absolute path for '%s': %w", dirPath, err)
	}

	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root dir itself
		if path == absPath {
			return nil
		}

		// Skip configured directories
		if d.IsDir() {
			dirName := filepath.Base(path)

			// Skip dot directories only if configured to do so
			if !opts.respectDotDirs && strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}

			// Skip directories in the skipDirs list
			if slices.Contains(opts.skipDirs, dirName) {
				return filepath.SkipDir
			}
		}

		if !d.IsDir() {
			fileName := strings.ToLower(filepath.Base(path))
			for _, ext := range fileExtensions {
				if strings.HasSuffix(fileName, ext) {
					result = append(result, path)
					break
				}
			}
		}

		return nil
	})
	if err != nil {
		return result, fmt.Errorf("error walking dir '%s': %w", absPath, err)
	}

	if len(result) == 0 {
		return result, fmt.Errorf("no YAML, YML, or JSON files found in directory %s", absPath)
	}

	return result, nil
}

/**
use list files like so
// Use default settings (traverse dot directories, skip common heavy dirs)
files, err := ListFiles("./")

// Skip specific directories
files, err := ListFiles("./", WithSkipDirs([]string{"node_modules", "vendor", "tmp"}))

// Skip dot directories (like original behavior)
files, err := ListFiles("./", WithRespectDotDirs(false))

// Custom configuration
files, err := ListFiles("./",
    WithSkipDirs([]string{"dist", "build"}),
    WithRespectDotDirs(true),
)
*/
