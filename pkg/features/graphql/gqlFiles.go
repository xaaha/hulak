package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	yaml "github.com/goccy/go-yaml"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// ProcessResult represents the outcome of processing a single GraphQL file
type ProcessResult struct {
	FilePath string
	ApiInfo  yamlparser.ApiInfo
	Error    error
}

// NeedsEnvResolution checks if any URL in the map contains template variables.
func NeedsEnvResolution(urlToFileMap map[string]string) bool {
	for url := range urlToFileMap {
		if strings.Contains(url, "{{") {
			return true
		}
	}
	return false
}

type fileInfo struct {
	kind     string
	url      string
	needsEnv bool
}

// peekFileInfo decodes a YAML file once and extracts kind, url, and whether
// any string value contains env variable references ({{.key}}).
// No template substitution is performed.
func peekFileInfo(filePath string) (fileInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return fileInfo{}, fmt.Errorf("cannot open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var data map[string]any
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return fileInfo{}, fmt.Errorf("cannot decode YAML: %w", err)
	}

	data = utils.ConvertKeysToLowerCase(data)

	var info fileInfo
	if v, ok := data["kind"].(string); ok {
		info.kind = strings.ToLower(v)
	}
	if v, ok := data["url"].(string); ok {
		info.url = strings.TrimSpace(v)
	}
	info.needsEnv = mapHasEnvVars(data)
	return info, nil
}

// mapHasEnvVars recursively checks if any string value in the map
// contains "{{." which indicates an env variable reference.
func mapHasEnvVars(data map[string]any) bool {
	for _, val := range data {
		switch v := val.(type) {
		case string:
			if strings.Contains(v, "{{.") {
				return true
			}
		case map[string]any:
			if mapHasEnvVars(v) {
				return true
			}
		}
	}
	return false
}

// FindGraphQLFiles finds all files with kind: GraphQL and a non-empty url field
// in the given directory and its subdirectories.
// Returns a map where keys are URLs (can be templates like {{.baseUrl}}) and values are file paths,
// along with a bool indicating whether any file contains env variable references.
func FindGraphQLFiles(dirPath string) (map[string]string, bool, error) {
	allFiles, err := utils.ListFiles(dirPath)
	if err != nil {
		return nil, false, err
	}

	graphqlFiles := make(map[string]string)
	needsEnv := false

	for _, filePath := range allFiles {
		if strings.Contains(filepath.Base(filePath), utils.ResponseBase) ||
			strings.Contains(filepath.Base(filePath), utils.ApiOptions) {
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

		graphqlFiles[info.url] = filePath
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

// ValidateGraphQLFile checks if a file exists, has kind: GraphQL, and has a non-empty url field.
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

	if !strings.EqualFold(info.kind, "GraphQL") {
		return "", false, fmt.Errorf("file '%s' does not have 'kind: GraphQL'", filePath)
	}

	if info.url == "" {
		return "", false, fmt.Errorf("file '%s' has empty or missing 'url' field", filePath)
	}

	return info.url, info.needsEnv, nil
}

// ProcessGraphQLFile fully processes a GraphQL YAML file with template resolution.
// This follows the same pattern as SendAndSaveAPIRequest, using checkYamlFile() for
// template resolution and applying defaults (method=POST, Content-Type: application/json).
// The returned ApiInfo has:
// - Url: Full URL with query parameters appended (using apicalls.PrepareURL)
// - UrlParams: nil (params already merged into Url)
// - Body: nil - the caller must set the query body (e.g., introspection query or TUI-built query)
func ProcessGraphQLFile(filePath string, secretsMap map[string]any) (yamlparser.ApiInfo, error) {
	graphqlConfig, _, err := yamlparser.FinalStructForGraphQL(filePath, secretsMap)
	if err != nil {
		return yamlparser.ApiInfo{}, err
	}

	apiInfo := graphqlConfig.PrepareGraphQLStruct()

	// Combine base URL with query parameters (same as StandardCall does)
	// This ensures the full URL is available for introspection and TUI display
	fullURL := apicalls.PrepareURL(apiInfo.Url, apiInfo.UrlParams)
	apiInfo.Url = fullURL
	apiInfo.UrlParams = nil // Params are now part of the URL

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
					ApiInfo:  apiInfo,
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
