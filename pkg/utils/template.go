package utils

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var templateVarPattern = regexp.MustCompile(`\{\{\s*\.`)

// FileHasTemplateVars checks if a file contains Go template variable references
// (e.g., {{.token}}) that require environment variable resolution.
// It checks {{getFile ...}} references and inspects the referenced files as well.
func FileHasTemplateVars(filePath string) bool {
	return fileHasTemplateVars(filePath, map[string]bool{})
}

func fileHasTemplateVars(filePath string, visited map[string]bool) bool {
	resolvedPath, err := resolveFilePath(filePath)
	if err != nil {
		return false
	}
	if visited[resolvedPath] {
		return false
	}
	visited[resolvedPath] = true

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return false
	}
	if templateVarPattern.Match(content) {
		return true
	}

	for _, arg := range extractGetFileArgs(string(content)) {
		if fileHasTemplateVars(arg, visited) {
			return true
		}
	}

	return false
}

// ReferencedFiles returns the files a request pulls in via {{getFile}},
// resolved to absolute paths, followed transitively, and de-duplicated in
// first-seen order. Surfaces the query/body files (e.g. a GraphQL .gql) that
// live apart from the request.
//
// Errors only when filePath itself can't be resolved or read. A referenced
// file that doesn't exist yet is still listed but not recursed into.
func ReferencedFiles(filePath string) ([]string, error) {
	resolvedPath, err := resolveFilePath(filePath)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{resolvedPath: true}
	var deps []string
	collectFileRefs(resolvedPath, seen, &deps)
	return deps, nil
}

// collectFileRefs appends resolvedPath's referenced files to deps and recurses
// into readable ones. seen guards against cycles and duplicates.
func collectFileRefs(resolvedPath string, seen map[string]bool, deps *[]string) {
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return
	}
	for _, arg := range extractGetFileArgs(string(content)) {
		// Resolve exactly like the runtime getFile (actions.GetFile): relative
		// to the project root. Dedup and recursion key on the real file; a
		// not-yet-created file is still surfaced, anchored to the project root
		// so the reported path is where a run would look for it.
		resolved, err := resolveFilePath(arg)
		if err != nil {
			resolved = projectRootRel(arg)
		}
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		*deps = append(*deps, resolved)
		collectFileRefs(resolved, seen, deps)
	}
}

// projectRootRel anchors a getFile arg that does not resolve to an existing
// file. Absolute args pass through; relative args join the project root — the
// same base actions.GetFile uses — falling back to the cleaned arg when the
// root cannot be found.
func projectRootRel(arg string) string {
	clean := filepath.Clean(arg)
	if filepath.IsAbs(clean) {
		return clean
	}
	if root, ok := FindProjectRoot(); ok {
		return filepath.Join(root, clean)
	}
	return clean
}

// MapHasEnvVars recursively checks if any string value in the map
// contains "{{." which indicates an env variable reference.
func MapHasEnvVars(data map[string]any) bool {
	for _, val := range data {
		if hasEnvVar(val) {
			return true
		}
	}
	return false
}

func extractGetFileArgs(content string) []string {
	var args []string
	for i := 0; i < len(content); {
		open := strings.Index(content[i:], "{{")
		if open == -1 {
			break
		}
		open += i + 2
		closeIdx := strings.Index(content[open:], "}}")
		if closeIdx == -1 {
			break
		}
		closeIdx += open

		expr := strings.TrimSpace(content[open:closeIdx])
		i = closeIdx + 2

		if !strings.HasPrefix(expr, TemplateFuncGetFile) {
			continue
		}

		rest := strings.TrimSpace(strings.TrimPrefix(expr, TemplateFuncGetFile))
		if rest == "" {
			continue
		}

		if arg := parseTemplateArg(rest); arg != "" {
			args = append(args, arg)
		}
	}
	return args
}

func parseTemplateArg(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	switch input[0] {
	case '"', '\'':
		quote := input[0]
		for i := 1; i < len(input); i++ {
			if input[i] == quote {
				return input[1:i]
			}
		}
		return ""
	default:
		for i := 0; i < len(input); i++ {
			if input[i] == ' ' || input[i] == '\t' || input[i] == '\n' || input[i] == '\r' {
				return input[:i]
			}
		}
		return input
	}
}

func resolveFilePath(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("file path cannot be empty")
	}

	cleanPath := filepath.Clean(filePath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", err
	}

	_, statErr := os.Stat(absPath)
	if statErr == nil {
		return absPath, nil
	}
	if !os.IsNotExist(statErr) {
		return "", statErr
	}

	projectRoot, _ := FindProjectRoot()
	relPath := filepath.Join(projectRoot, cleanPath)
	if _, err := os.Stat(relPath); err != nil {
		return "", err
	}
	return relPath, nil
}

func hasEnvVar(val any) bool {
	switch v := val.(type) {
	case string:
		return strings.Contains(v, "{{.")
	case map[string]any:
		return MapHasEnvVars(v)
	case []any:
		if slices.ContainsFunc(v, hasEnvVar) {
			return true
		}
	}
	return false
}
