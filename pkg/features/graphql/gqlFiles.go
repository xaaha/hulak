package graphql

import (
	"fmt"
	yaml "github.com/goccy/go-yaml"
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// ProcessResult represents the outcome of processing a single GraphQL file
type ProcessResult struct {
	FilePath string
	APIInfo  yamlparser.APIInfo
	Error    error
}

// NeedsEnvResolution checks if any URL or file in the map contains template
// variables that need environment resolution. It checks both URL strings and
// the raw file contents for dot-access patterns like {{.key}}. A multi-document
// file is skipped: processing rejects it whatever the environment.
func NeedsEnvResolution(urlToFileMap map[string]string) bool {
	for url, filePath := range urlToFileMap {
		if info, err := peekFileInfo(filePath); err == nil && info.parseErr != nil {
			continue
		}
		if strings.Contains(url, "{{") {
			return true
		}
		if utils.FileHasTemplateVars(filePath) {
			return true
		}
	}
	return false
}

type fileInfo struct {
	kind     string
	url      string
	needsEnv bool
	parseErr error
}

var (
	graphqlKindLine = regexp.MustCompile(`(?im)^kind[ \t]*:[ \t]*["']?graphql["']?[ \t]*(#.*)?\r?$`)
	urlLine         = regexp.MustCompile(`(?im)^url[ \t]*:[ \t]*(.+?)[ \t]*\r?$`)
	documentEnd     = regexp.MustCompile(`(?m)^(---|\.\.\.)([ \t].*)?\r?$`)
)

func peekUnparsable(filePath string, content []byte, decodeErr error) (fileInfo, error) {
	first := firstDocument(content)
	url := urlLine.FindSubmatch(first)
	if !graphqlKindLine.Match(first) || url == nil {
		return fileInfo{}, fmt.Errorf("cannot decode YAML: %w", decodeErr)
	}
	parseErr := utils.ValidateSingleYAMLDoc(filePath, content)
	if parseErr == nil {
		parseErr = fmt.Errorf("parsing %s: %w", filePath, decodeErr)
	}
	return fileInfo{kind: "graphql", url: scalarText(string(url[1])), parseErr: parseErr}, nil
}

func firstDocument(content []byte) []byte {
	start := 0
	for _, loc := range documentEnd.FindAllIndex(content, -1) {
		if strings.TrimSpace(stripComments(content[start:loc[0]])) != "" {
			return content[:loc[0]]
		}
		start = loc[1]
	}
	return content
}

func stripComments(text []byte) string {
	var b strings.Builder
	for line := range strings.SplitSeq(string(text), "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "%") {
			b.WriteString(t)
		}
	}
	return b.String()
}

func scalarText(raw string) string {
	if strings.HasPrefix(raw, "#") {
		return ""
	}
	if raw != "" && (raw[0] == '"' || raw[0] == '\'') {
		if end := strings.IndexByte(raw[1:], raw[0]); end >= 0 {
			return raw[1 : end+1]
		}
		return strings.TrimSpace(raw[1:])
	}
	if i := strings.Index(raw, " #"); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}

// peekFileInfo decodes a YAML file's first document and extracts kind, url, and
// whether any string value contains env variable references ({{.key}}).
// No template substitution is performed.
func peekFileInfo(filePath string) (fileInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fileInfo{}, fmt.Errorf("cannot open file: %w", err)
	}

	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return peekUnparsable(filePath, content, err)
	}

	data = utils.ConvertKeysToLowerCase(data)

	var info fileInfo
	if v, ok := data["kind"].(string); ok {
		info.kind = strings.ToLower(v)
	}
	if v, ok := data["url"].(string); ok {
		info.url = strings.TrimSpace(v)
	}
	info.parseErr = utils.ValidateSingleYAMLDoc(filePath, content)
	info.needsEnv = info.parseErr == nil && utils.MapHasEnvVars(data)
	return info, nil
}

// FindGraphQLFiles finds all files with kind: GraphQL and a non-empty url field
// in the given directory and its subdirectories. A file that is split or fails to
// parse is kept when its first document matches, so processing reports it.
// Returns a map where keys are URLs (can be templates like {{.baseUrl}}) and values are file paths,
// along with a bool indicating whether any file contains env variable references.
func FindGraphQLFiles(dirPath string) (map[string]string, bool, error) {
	allFiles, err := utils.ListFiles(dirPath)
	if err != nil {
		return nil, false, err
	}

	graphqlFiles := make(map[string]string)
	broken := make(map[string]bool)
	needsEnv := false

	for _, filePath := range allFiles {
		if strings.Contains(filepath.Base(filePath), utils.ResponseBase) ||
			strings.Contains(filepath.Base(filePath), utils.OptionsReference) {
			continue
		}

		info, err := peekFileInfo(filePath)
		if err != nil {
			continue
		}

		if !strings.EqualFold(info.kind, string(yamlparser.KindGraphQL)) {
			continue
		}
		if info.url == "" {
			continue
		}

		key := info.url
		_, taken := graphqlFiles[key]
		switch {
		case taken && info.parseErr != nil:
			key = filePath
		case broken[key]:
			graphqlFiles[graphqlFiles[key]] = graphqlFiles[key]
			delete(broken, key)
		}
		if info.parseErr != nil && key == info.url {
			broken[key] = true
		}
		graphqlFiles[key] = filePath
		if info.needsEnv {
			needsEnv = true
		}
	}

	if len(graphqlFiles) == 0 {
		return nil, false, fmt.Errorf(
			"no files with 'kind: GraphQL' and non-empty 'url' field found in directory: %s",
			dirPath,
		)
	}

	return graphqlFiles, needsEnv, nil
}

// ValidateGraphQLFile checks if a file exists and has a non-empty url field.
// Single-file mode intentionally does not require kind: GraphQL because the
// caller has already selected the file explicitly.
// Returns the raw URL (template or full URL) and whether env variable references were found,
// without performing template substitution.
func ValidateGraphQLFile(filePath string) (string, bool, error) {
	filePath = filepath.Clean(filePath)

	if !utils.FileExists(filePath) {
		return "", false, fmt.Errorf("file not found: %s", filePath)
	}

	info, err := peekFileInfo(filePath)
	if err != nil {
		return "", false, fmt.Errorf("error reading file '%s': %w", filePath, err)
	}
	if info.parseErr != nil {
		return "", false, info.parseErr
	}

	if info.url == "" {
		return "", false, fmt.Errorf("file '%s' has empty or missing 'url' field", filePath)
	}

	return info.url, info.needsEnv, nil
}

// ProcessGraphQLFile fully processes a GraphQL YAML file with template resolution.
// This follows the same pattern as SendAndSaveAPIRequest, using checkYamlFile() for
// template resolution and applying defaults (method=POST, Content-Type: application/json).
// The returned APIInfo has:
// - URL: Full URL with query parameters appended (using apicalls.PrepareURL)
// - URLParams: nil (params already merged into URL)
// - Body: nil - the caller must set the query body (e.g., introspection query or TUI-built query)
func ProcessGraphQLFile(filePath string, secretsMap map[string]any) (yamlparser.APIInfo, error) {
	redact, redactErr := apicalls.NewSecretRedactor(filePath, secretsMap)
	if redactErr != nil {
		return yamlparser.APIInfo{}, redactErr
	}

	graphqlConfig, _, err := yamlparser.FinalStructForGraphQL(filePath, secretsMap)
	if err != nil {
		return yamlparser.APIInfo{}, apicalls.RedactErr(redact, err)
	}

	apiInfo := graphqlConfig.PrepareGraphQLStruct()

	// Combine base URL with query parameters (same as StandardCall does)
	// This ensures the full URL is available for introspection and TUI display
	fullURL := apicalls.PrepareURL(apiInfo.URL, apiInfo.URLParams)
	apiInfo.URL = fullURL
	apiInfo.URLParams = nil // Params are now part of the URL

	return apiInfo, nil
}

// ProcessFilesConcurrent processes GraphQL files using a simple worker pool.
// Uses utils.GetWorkers to determine the number of concurrent workers.
// Returns all results (successful and failed) for the caller to handle.
func ProcessFilesConcurrent(filePaths []string, secretsMap map[string]any) []ProcessResult {
	if len(filePaths) == 0 {
		return nil
	}

	numFiles := len(filePaths)
	numWorkers := utils.GetWorkers(&numFiles)

	jobs := make(chan string, numFiles)
	results := make(chan ProcessResult, numFiles)

	// Start workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for job := range jobs {
				apiInfo, err := ProcessGraphQLFile(job, secretsMap)
				results <- ProcessResult{
					FilePath: job,
					APIInfo:  apiInfo,
					Error:    err,
				}
			}
		})
	}

	// Send jobs
	for _, fp := range filePaths {
		jobs <- fp
	}
	close(jobs)

	// Wait for workers and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []ProcessResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}
