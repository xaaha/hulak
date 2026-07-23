package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

// templateVarPattern matches a dot-access template reference like {{.token}}
// or {{ .token }}, tolerating whitespace between the braces and the dot the
// same way Go's template engine does at substitution time.
var templateVarPattern = regexp.MustCompile(`\{\{\s*\.`)

// FileHasTemplateVars reports whether a request file's YAML values contain env
// variable references (e.g. {{.token}}) that require environment resolution.
//
// It decodes the YAML so comments never count — they are dropped by the decoder
// and never reach runtime substitution. It does not follow {{getFile ...}}
// references either: getFile dumps the referenced file's raw content into
// context and hulak never re-templates it (single-pass substitution), so an env
// var inside a referenced file can never resolve and must not force env loading.
func FileHasTemplateVars(filePath string) bool {
	resolvedPath, err := resolveFilePath(filePath)
	if err != nil {
		return false
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return false
	}

	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return false
	}
	return MapHasEnvVars(data)
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
		resolved, err := ResolveProjectFile(arg)
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

// anchorToRoot is the shared getFile anchoring rule: an absolute path passes
// through, a relative path is joined to the project root. filepath.Join keeps
// this correct on every OS separator.
func anchorToRoot(cleanPath, root string) string {
	if filepath.IsAbs(cleanPath) {
		return cleanPath
	}
	return filepath.Join(root, cleanPath)
}

// projectRootRel anchors a getFile arg that does not resolve to an existing
// file, so the lister can still report where a run would look for it. Falls
// back to the cleaned arg when no project root is found.
func projectRootRel(arg string) string {
	clean := filepath.Clean(arg)
	root, ok := FindProjectRoot()
	if !ok {
		return clean
	}
	return anchorToRoot(clean, root)
}

// MapHasEnvVars recursively checks whether any string value in the map holds a
// dot-access template reference ({{.key}} or {{ .key }}), which indicates an
// env variable reference.
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

// resolveFilePath locates a file for static analysis (env-var detection, dep
// listing). Unlike ResolveProjectFile it enforces no project containment and
// tolerates a caller-supplied absolute path outside any project, since the
// caller has already located the file. A relative path is joined to the project
// root (never cwd-first), falling back to cwd only when no project root exists.
func resolveFilePath(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("file path cannot be empty")
	}

	root, _ := FindProjectRoot()
	path := anchorToRoot(filepath.Clean(filePath), root)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

// ResolveProjectFile resolves a getFile path to an absolute path inside the
// project root. It is the single source of truth for getFile resolution, used
// by both the runtime (actions.GetFile) and the dependency lister.
//
// A relative path is always project-root-relative, never cwd-relative, so
// resolution is identical no matter where hulak is invoked from. An absolute
// path is used as-is. Either way the result must live inside the project root
// and point at an existing regular file.
func ResolveProjectFile(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("file path cannot be empty")
	}

	projectRoot, found := FindProjectRoot()
	if !found {
		return "", errors.New("not a hulak project: could not find project root")
	}

	absPath := anchorToRoot(filepath.Clean(filePath), projectRoot)
	if !withinRoot(absPath, projectRoot) {
		return "", fmt.Errorf("access denied: file path %s is outside the project root", filePath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist %s", absPath)
		}
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, not a file", filePath)
	}
	return absPath, nil
}

// withinRoot reports whether absPath is root itself or nested under it, using a
// path-segment comparison so a sibling like /project-evil does not count as
// being inside /project.
func withinRoot(absPath, root string) bool {
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func hasEnvVar(val any) bool {
	switch v := val.(type) {
	case string:
		return templateVarPattern.MatchString(v)
	case map[string]any:
		return MapHasEnvVars(v)
	case []any:
		if slices.ContainsFunc(v, hasEnvVar) {
			return true
		}
	}
	return false
}
