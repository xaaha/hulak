package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// --- Options ----

// listFilesOptions is a structure for configurable function behavior
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

// --- helpers ---

// defaultOptions are the sane defaults list file will ignore
func defaultOptions() listFilesOptions {
	return listFilesOptions{
		skipDirs:       []string{"node_modules", ".git", ".svn", ".hg", ".idea", ".vscode"},
		respectDotDirs: true,
	}
}

// applyOptions applies the provided options
func applyOptions(options []ListFilesOption) listFilesOptions {
	opts := defaultOptions()
	for _, opt := range options {
		opt(&opts)
	}
	return opts
}

func isWantedFilePath(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, YAML) || strings.HasSuffix(name, YML) ||
		strings.HasSuffix(name, JSON)
}

func shouldSkipDir(dirName string, opts listFilesOptions) bool {
	if !opts.respectDotDirs && strings.HasPrefix(dirName, ".") {
		return true
	}
	return slices.Contains(opts.skipDirs, dirName)
}

// ListFiles generates all .yaml, .yml, or .json files in a directory, with configurable directory exclusion
// Files are added as they are discovered so it does not guarantee any files are run before the other
func ListFiles(dirPath string, options ...ListFilesOption) ([]string, error) {
	opts := applyOptions(options)

	if dirPath == "" {
		dirPath = "."
	}

	// Clean + verify directory
	clean := filepath.Clean(dirPath)

	info, err := os.Stat(clean)
	if err != nil {
		return nil, fmt.Errorf("error accessing directory '%s': %w", dirPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", dirPath)
	}

	abs, err := filepath.Abs(clean)
	if err != nil {
		return nil, fmt.Errorf("error resolving absolute path for '%s': %w", dirPath, err)
	}

	var result []string

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root dir itself
		if path == abs {
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(d.Name(), opts) {
				return fs.SkipDir
			}
			return nil
		}

		// Files
		if isWantedFilePath(d.Name()) {
			result = append(result, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking dir '%s': %w", abs, err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no YAML, YML, or JSON files found in directory %s", abs)
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
