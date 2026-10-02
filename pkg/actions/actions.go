// Package actions has all the actions we use in yaml parser
package actions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xaaha/hulak/pkg/utils"
)

// cachedFile is one parsed response file plus the stat it was read from.
// The parsed tree is handed to every caller that asks for a key out of this
// file, so it must never be written to after it lands here — see
// normalizeNumbers in readJSONFile.
type cachedFile struct {
	info    os.FileInfo
	content any
}

var (
	contentCacheMu sync.RWMutex
	contentCache   = make(map[string]cachedFile)

	// Add file operation mutex
	fileOpsMutex sync.Map
)

// GetValueOf gets the value of key from a json file.
//
// Nothing about the lookup is memoized across a changing file: every call
// stats the file it is about to read and throws away the parsed copy the
// moment the file on disk no longer matches it. A single run refreshes a
// token mid-flight, and every reference after that has to see the new one
// (#251, #253).
func GetValueOf(key, fileName string) any {
	return processValueOf(key, fileName)
}

// ResetCache drops every parsed response file. Entries invalidate themselves
// against the file on disk, so this is for the coarser staleness a stat can't
// see: a process serving tool calls for hours resolves bare filenames against
// a project tree that changes underneath it.
func ResetCache() {
	contentCacheMu.Lock()
	defer contentCacheMu.Unlock()
	clear(contentCache)
}

// BasicAuth takes a username and password, joins them with a colon,
// base64-encodes the result, and returns the full header value "Basic <encoded>".
// Both arguments are treated as plain strings — use .env template vars for secrets.
func BasicAuth(username, password string) string {
	credentials := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encoded
}

// GetFile reads the content of a file referenced by a {{getFile}} template.
// Resolution is delegated to utils.ResolveProjectFile: relative paths are
// project-root-relative (never cwd-relative), absolute paths are used as-is,
// and either way the file must live inside the project root.
func GetFile(filePath string) (string, error) {
	absPath, err := utils.ResolveProjectFile(filePath)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return string(content), nil
}

func getFileMutex(filePath string) *sync.Mutex {
	mutex, _ := fileOpsMutex.LoadOrStore(filePath, &sync.Mutex{})
	return mutex.(*sync.Mutex)
}

// processValueOf processes GetValueOf action — returns the value at key from
// the resolved JSON file, or "" if anything goes wrong. Errors are printed to
// stderr; we can't return them because this is invoked as a template function
// whose signature is fixed at func(...) any. Stdout stays clean so any
// downstream `$(...)` capture still gets clean program output.
func processValueOf(key, fileName string) any {
	// Validate inputs
	if key == "" || fileName == "" {
		if key == "" {
			utils.PrintErrorStderr(
				fmt.Sprintf("provide key for %s action", utils.TemplateFuncGetValueOf),
			)
		} else {
			utils.PrintErrorStderr(
				fmt.Sprintf(
					"provide fileName/path to key for %s action",
					utils.TemplateFuncGetValueOf,
				),
			)
		}
		return ""
	}

	jsonResFilePath, err := resolveJSONFilePath(fileName)
	if err != nil {
		utils.PrintErrorStderr(err.Error())
		return ""
	}

	content, err := readJSONFile(jsonResFilePath)
	if err != nil {
		utils.PrintErrorStderr(err.Error())
		return ""
	}

	result, err := extractValueByKey(key, content)
	if err != nil {
		utils.PrintErrorStderr(fmt.Sprintf(
			"looking up value '%s': make sure '%s' exists and has key '%s'",
			key,
			filepath.Join(
				"...",
				utils.FileNameWithoutExtension(filepath.Dir(jsonResFilePath)),
				filepath.Base(jsonResFilePath),
			),
			key,
		))
		return ""
	}

	return result
}

// resolveJSONFilePath determines the correct JSON file path based on the input fileName
func resolveJSONFilePath(fileName string) (string, error) {
	cleanFileName := filepath.Clean(fileName)

	// Check if the fileName contains path separators or starts with ".."
	isPath := strings.Contains(cleanFileName, string(filepath.Separator)) ||
		strings.HasPrefix(cleanFileName, "..")

	if isPath {
		// Handle as a direct file path
		absPath, err := filepath.Abs(cleanFileName)
		if err != nil {
			return "", fmt.Errorf(
				"error resolving absolute path for '%s': %s",
				fileName, err.Error(),
			)
		}

		// If it's a JSON file, use it directly
		if strings.HasSuffix(cleanFileName, utils.JSON) {
			return absPath, nil
		}

		// For non-JSON files, look for _response.json
		dirPath := filepath.Dir(absPath)
		baseFileName := utils.FileNameWithoutExtension(absPath)
		return filepath.Join(dirPath, baseFileName+utils.ResponseFileName), nil
	}

	// Handle as a filename to search for
	yamlPathList, err := utils.ListMatchingFiles(cleanFileName)
	if err != nil {
		return "", fmt.Errorf(
			"error occurred while grabbing matching paths for '%s': %s",
			cleanFileName, err.Error(),
		)
	}

	if len(yamlPathList) == 0 {
		return "", fmt.Errorf("could not find matching files %s", cleanFileName)
	}

	// Handle multiple matches warning
	if len(yamlPathList) > 1 {
		utils.PrintWarningStderr(
			fmt.Sprintf("multiple '%s' files; using %s", cleanFileName, yamlPathList[0]),
		)
	}

	singlePath := yamlPathList[0]
	if strings.HasSuffix(cleanFileName, utils.JSON) {
		return singlePath, nil
	}

	dirPath := filepath.Dir(singlePath)
	jsonBaseName := utils.FileNameWithoutExtension(singlePath) + utils.ResponseFileName
	return filepath.Join(dirPath, jsonBaseName), nil
}

// readJSONFile returns the parsed contents of filePath, reusing the previous
// parse only while the file on disk still matches the one it came from.
//
// The stat happens before the read, never after: a rewrite landing between the
// two stores fresh content against a stale stat, so the next call re-reads. The
// other order would store stale content against a fresh stat and serve it.
func readJSONFile(filePath string) (any, error) {
	// Get file-specific mutex
	fileMutex := getFileMutex(filePath)
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Check file existence under lock
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file '%s' does not exist", filePath)
		}
		return nil, fmt.Errorf(
			"error occurred while reading the file '%s': %s",
			filepath.Base(filePath),
			err.Error(),
		)
	}

	if content, ok := cachedContent(filePath, info); ok {
		return content, nil
	}

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"error occurred while reading the file '%s': %s",
			filepath.Base(filePath),
			err.Error(),
		)
	}

	// Parse JSON
	var content any
	err = json.Unmarshal(fileContent, &content)
	if err != nil {
		return nil, fmt.Errorf(
			"make sure %s has proper json content: %s",
			filepath.Base(filePath),
			err.Error(),
		)
	}

	// Normalize here rather than per extraction: callers share this tree, and
	// a converting walk per lookup would be a write to a map another goroutine
	// is reading.
	content = normalizeNumbers(content)

	contentCacheMu.Lock()
	contentCache[filePath] = cachedFile{info: info, content: content}
	contentCacheMu.Unlock()

	return content, nil
}

// cachedContent returns the parsed copy of filePath when fresh describes the
// same file it was read from.
func cachedContent(filePath string, fresh os.FileInfo) (any, bool) {
	contentCacheMu.RLock()
	defer contentCacheMu.RUnlock()

	entry, exists := contentCache[filePath]
	if !exists {
		return nil, false
	}
	if !sameFile(entry.info, fresh) {
		return nil, false
	}
	return entry.content, true
}

// sameFile reports whether two stats describe the same file contents. Every
// comparison it can't make confidently has to come back false: a wrong "not
// the same" costs one re-read, a wrong "same" serves a token that expired.
//
// Identity carries the weight. Response files are written through
// utils.AtomicWriteFile, which renames a temp file over the target, so every
// write hulak performs leaves a different inode behind no matter what the size
// and timestamp say. Size and mtime catch an in-place edit by something else.
func sameFile(cached, fresh os.FileInfo) bool {
	return os.SameFile(cached, fresh) &&
		cached.Size() == fresh.Size() &&
		cached.ModTime().Equal(fresh.ModTime())
}

// extractValueByKey extracts a value from JSON content using the provided key
func extractValueByKey(key string, content any) (any, error) {
	var result any
	var err error

	switch typedContent := content.(type) {
	case []any:
		// For array root, key must start with [
		if strings.HasPrefix(key, "[") && strings.Contains(key, "]") {
			// Wrap array in a map with empty key for LookupValue
			result, err = utils.LookupValue(key, map[string]any{
				"": typedContent,
			})
		} else {
			return "", fmt.Errorf("JSON content is an array, use [index] notation to access elements")
		}

	case map[string]any:
		// For object root
		result, err = utils.LookupValue(key, typedContent)

	default:
		return "", fmt.Errorf("unexpected JSON content format")
	}

	if err != nil {
		return "", err
	}

	return result, nil
}

// normalizeNumbers rewrites float64 values that represent whole numbers as int
// or int64, so a JSON 30 renders as "30" and not "30.0". Runs once over a
// freshly parsed document; the maps and slices it rewrites are not shared with
// anything yet.
func normalizeNumbers(v any) any {
	switch value := v.(type) {
	case float64:
		// Check if it's an integer (no decimal part)
		if value == float64(int64(value)) {
			// For small numbers that fit in int, use int
			if value >= float64(math.MinInt) && value <= float64(math.MaxInt) {
				return int(value)
			}
			// For larger numbers, use int64
			return int64(value)
		}
	case []any:
		for i, item := range value {
			value[i] = normalizeNumbers(item)
		}
	case map[string]any:
		for k, item := range value {
			value[k] = normalizeNumbers(item)
		}
	}
	return v
}

// AttachFile resolves a file referenced by an {{attachFile}} template and
// returns a marker naming it, not its contents.
//
// getFile inlines what it reads; a file destined for an upload cannot travel
// that way, because the request map is re-encoded to YAML before the body is
// built. The body encoder resolves this marker and opens the file itself, so
// the bytes go straight from disk to the socket.
func AttachFile(filePath string) (string, error) {
	absPath, err := utils.ResolveAttachPath(filePath)
	if err != nil {
		return "", err
	}
	return utils.FileRef(absPath), nil
}
