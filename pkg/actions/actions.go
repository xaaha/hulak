// Package actions has all the actions we use in yaml parser
package actions

import (
	"crypto/sha256"
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

// Keyed by contents, not by stat: a refreshed token is often the same length as
// the one it replaced, and os.SameFile is a no-op on Windows (#251, #253).
var (
	valueCacheMu sync.RWMutex
	valueCache   = make(map[string]any)

	// Keyed by search root, since the MCP server chdirs between projects.
	// These entries cannot check themselves the way valueCache does: the
	// response file a name resolves to legitimately may not exist yet.
	pathCacheMu sync.RWMutex
	pathCache   = make(map[string]string)
)

// Package-level var so tests can count how often the walk runs.
var listMatchingFiles = utils.ListMatchingFiles

// One template reference is resolved several times per file parse, and a
// directory run repeats that per file, so an unwritten response file printed
// the same line 240 times on a 40-request run.
var (
	reportedMu sync.Mutex
	reported   = make(map[string]struct{})
)

func reportErrorOnce(msg string) {
	reportErrorOnceFor(msg, msg)
}

// reportErrorOnceFor dedupes on dedupeOn rather than on msg, for messages that
// abbreviate the path they name and so are not unique on their own.
func reportErrorOnceFor(dedupeOn, msg string) {
	if firstReport("error", dedupeOn) {
		utils.PrintErrorStderr(msg)
	}
}

func reportWarningOnce(msg string) {
	if firstReport("warning", msg) {
		utils.PrintWarningStderr(msg)
	}
}

func firstReport(kind, msg string) bool {
	key := kind + ": " + msg

	reportedMu.Lock()
	defer reportedMu.Unlock()

	if _, seen := reported[key]; seen {
		return false
	}
	reported[key] = struct{}{}
	return true
}

// GetValueOf gets the value of key from a json file.
func GetValueOf(key, fileName string) any {
	return processValueOf(key, fileName)
}

// ResetCache exists for the resolved paths. Results carry a digest of their own
// bytes and cannot go stale; where a bare filename points can still be made
// wrong by the project tree moving over a long-lived process.
func ResetCache() {
	valueCacheMu.Lock()
	clear(valueCache)
	valueCacheMu.Unlock()

	pathCacheMu.Lock()
	clear(pathCache)
	pathCacheMu.Unlock()

	reportedMu.Lock()
	clear(reported)
	reportedMu.Unlock()
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

// processValueOf processes GetValueOf action — returns the value at key from
// the resolved JSON file, or "" if anything goes wrong. Errors are printed to
// stderr; we can't return them because this is invoked as a template function
// whose signature is fixed at func(...) any. Stdout stays clean so any
// downstream `$(...)` capture still gets clean program output.
func processValueOf(key, fileName string) any {
	// Validate inputs
	if key == "" || fileName == "" {
		if key == "" {
			reportErrorOnce(
				fmt.Sprintf("provide key for %s action", utils.TemplateFuncGetValueOf),
			)
		} else {
			reportErrorOnce(
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
		reportErrorOnce(err.Error())
		return ""
	}

	raw, err := readResponseFile(jsonResFilePath)
	if err != nil {
		reportErrorOnce(err.Error())
		return ""
	}

	// Failures are cached too: they are as true of these bytes as a success is,
	// and the file gaining the key changes the digest.
	cacheKey := valueCacheKey(jsonResFilePath, raw, key)
	valueCacheMu.RLock()
	cached, hit := valueCache[cacheKey]
	valueCacheMu.RUnlock()
	if hit {
		return cached
	}

	result := extractFromJSON(key, jsonResFilePath, raw)

	valueCacheMu.Lock()
	valueCache[cacheKey] = result
	valueCacheMu.Unlock()

	return result
}

func valueCacheKey(filePath string, raw []byte, key string) string {
	digest := sha256.Sum256(raw)
	return filePath + "\x00" + string(digest[:]) + "\x00" + key
}

func extractFromJSON(key, filePath string, raw []byte) any {
	var content any
	if err := json.Unmarshal(raw, &content); err != nil {
		reportErrorOnce(fmt.Sprintf(
			"make sure %s has proper json content: %s",
			filepath.Base(filePath),
			err.Error(),
		))
		return ""
	}

	result, err := extractValueByKey(key, content)
	if err != nil {
		// Dedupe on the full path: the message abbreviates it, so two response
		// files under same-named directories would collapse into one line.
		reportErrorOnceFor(filePath+" "+key, fmt.Sprintf(
			"looking up value '%s': make sure '%s' exists and has key '%s'",
			key,
			filepath.Join(
				"...",
				utils.FileNameWithoutExtension(filepath.Dir(filePath)),
				filepath.Base(filePath),
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
	searchRoot, err := utils.CreatePath("")
	if err != nil {
		return "", fmt.Errorf("error getting initial file path for '%s': %s", fileName, err.Error())
	}
	cacheKey := searchRoot + string(filepath.Separator) + cleanFileName

	pathCacheMu.RLock()
	resolved, cached := pathCache[cacheKey]
	pathCacheMu.RUnlock()
	if cached {
		return resolved, nil
	}

	yamlPathList, err := listMatchingFiles(cleanFileName, searchRoot)
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
		reportWarningOnce(
			fmt.Sprintf("multiple '%s' files; using %s", cleanFileName, yamlPathList[0]),
		)
	}

	singlePath := yamlPathList[0]
	resolved = singlePath
	if !strings.HasSuffix(cleanFileName, utils.JSON) {
		dirPath := filepath.Dir(singlePath)
		jsonBaseName := utils.FileNameWithoutExtension(singlePath) + utils.ResponseFileName
		resolved = filepath.Join(dirPath, jsonBaseName)
	}

	pathCacheMu.Lock()
	pathCache[cacheKey] = resolved
	pathCacheMu.Unlock()

	return resolved, nil
}

func readResponseFile(filePath string) ([]byte, error) {
	raw, err := os.ReadFile(filePath)
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
	return raw, nil
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

	// Convert float64 to int64 if it represents a whole number
	return convertNumberToProperType(result), nil
}

// convertNumberToProperType rewrites float64 values that represent whole
// numbers as int or int64, so a JSON 30 renders as "30" and not "30.0". It
// rewrites in place, which is safe only because the document it walks was
// parsed by this call and is shared with no one.
func convertNumberToProperType(v any) any {
	switch value := v.(type) {
	case float64:
		// math.Trunc rather than a round trip through int64: converting a
		// float64 that does not fit the destination is implementation-defined
		// (arm64 saturates, amd64 wraps), so JSON 9223372036854775807
		// normalized one way on a darwin build and the other on linux.
		if value != math.Trunc(value) {
			return v
		}
		const int64Limit = 9223372036854775808.0 // 2^63, the first int64 cannot hold
		if value < -int64Limit || value >= int64Limit {
			return v
		}
		whole := int64(value)
		// For small numbers that fit in int, use int
		if whole >= int64(math.MinInt) && whole <= int64(math.MaxInt) {
			return int(whole)
		}
		// For larger numbers, use int64
		return whole
	case []any:
		for i, item := range value {
			value[i] = convertNumberToProperType(item)
		}
	case map[string]any:
		for k, item := range value {
			value[k] = convertNumberToProperType(item)
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
