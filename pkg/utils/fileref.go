package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileRefPrefix marks a resolved attachment inside a template result.
//
// A template action can only hand back a string, and the request map is
// re-encoded to YAML mid-pipeline, so file bytes cannot travel that path.
// attachFile returns this marker plus an absolute path instead, and the body
// encoder opens the file itself.
const FileRefPrefix = "hulak:file:"

// FileRef builds the marker for an already-resolved absolute path.
func FileRef(absPath string) string {
	return FileRefPrefix + absPath
}

// FileRefPath returns the path carried by a marker. Only an exact whole-value
// match counts: a marker embedded in a longer string is a mistake the caller
// should report, not a literal to send.
func FileRefPath(value string) (string, bool) {
	path, ok := strings.CutPrefix(value, FileRefPrefix)
	if !ok || path == "" {
		return "", false
	}
	return path, true
}

// ResolveAttachPath resolves a path for attachFile.
//
// It deliberately differs from ResolveProjectFile: an upload usually comes from
// outside the repo, so an explicitly absolute or ~-prefixed path may leave the
// project root. A relative path still resolves against the project root and
// still may not escape it, because "assets/../../../etc/passwd" reads as
// innocuous where "/etc/passwd" does not. Leaving the repo has to be spelled
// out in the path, which is the disclosure; no warning is needed on top.
func ResolveAttachPath(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("attachFile needs a path")
	}

	explicit := filepath.IsAbs(filePath) || strings.HasPrefix(filePath, "~")

	absPath, err := ExpandPath(filePath)
	if err != nil {
		return "", err
	}

	if !explicit {
		projectRoot, found := FindProjectRoot()
		if !found {
			return "", fmt.Errorf("not a hulak project: could not find project root")
		}
		absPath = anchorToRoot(filepath.Clean(filePath), projectRoot)
		if !withinRoot(absPath, projectRoot) {
			return "", fmt.Errorf(
				"attachFile %s escapes the project root; pass an absolute or ~ path to attach it deliberately",
				filePath,
			)
		}
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("attachFile: no such file %s", absPath)
		}
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("attachFile: %s is a directory", filePath)
	}

	return absPath, nil
}
