package graphql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	yaml "github.com/goccy/go-yaml"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// ResolutionResult represents the outcome of resolving a single template URL
type ResolutionResult struct {
	RawURL      string // Original URL (potentially with templates)
	ResolvedURL string // Final resolved URL (empty if error)
	FilePath    string // Path to the GraphQL file
	Error       error  // nil if successful
}

// ResolutionSummary aggregates all resolution results
type ResolutionSummary struct {
	Successful []ResolutionResult // Successfully resolved files
	Failed     []ResolutionResult // Files that failed resolution
	TotalFiles int                // Total number of files processed
}

// workItem represents a single file to process
type workItem struct {
	rawURL   string
	filePath string
}

// HasErrors returns true if any resolutions failed
func (rs *ResolutionSummary) HasErrors() bool {
	return len(rs.Failed) > 0
}

// GetResolvedMap converts successful results to the traditional map[url]filepath format
func (rs *ResolutionSummary) GetResolvedMap() map[string]string {
	resolved := make(map[string]string, len(rs.Successful))
	for _, result := range rs.Successful {
		resolved[result.ResolvedURL] = result.FilePath
	}
	return resolved
}

// FormatErrors formats all errors in a user-friendly way with file paths and error details
func (rs *ResolutionSummary) FormatErrors() string {
	if len(rs.Failed) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Failed to resolve %d file(s):\n\n", len(rs.Failed))

	for i, failure := range rs.Failed {
		fmt.Fprintf(&sb, "%d. File: %s\n", i+1, failure.FilePath)
		fmt.Fprintf(&sb, "   URL:  %s\n", failure.RawURL)
		fmt.Fprintf(&sb, "   Error: %v\n", failure.Error)

		// Add helpful context for common errors
		if strings.Contains(failure.Error.Error(), "key") &&
			strings.Contains(failure.Error.Error(), "not found") {
			sb.WriteString("   Hint: Check your environment variables\n")
		}

		if i < len(rs.Failed)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// peekKindField reads a YAML file and extracts only the 'kind' field
// without performing template substitution. This prevents template
// substitution errors from being displayed for non-GraphQL files.
func peekKindField(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var data map[string]any
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return "", fmt.Errorf("cannot decode YAML: %w", err)
	}

	// Convert keys to lowercase (consistent with yamlparser)
	data = utils.ConvertKeysToLowerCase(data)

	// Extract kind field
	kindValue, exists := data["kind"]
	if !exists {
		return "", nil // No error, just no kind field
	}

	// Return kind as string (lowercase for consistency)
	if kind, ok := kindValue.(string); ok {
		return strings.ToLower(kind), nil
	}

	return "", nil // Kind exists but not a string
}

// peekURLField reads a YAML file and extracts only the 'url' field
// without performing template substitution. Returns the raw URL value
// which could be a template string like "{{.baseUrl}}" or a full URL.
func peekURLField(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var data map[string]any
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return "", fmt.Errorf("cannot decode YAML: %w", err)
	}

	// Convert keys to lowercase (consistent with yamlparser)
	data = utils.ConvertKeysToLowerCase(data)

	// Extract url field
	urlValue, exists := data["url"]
	if !exists {
		return "", nil // No url field
	}

	// Return URL as string (could be template or full URL)
	if url, ok := urlValue.(string); ok {
		return strings.TrimSpace(url), nil
	}

	return "", nil // URL exists but not a string
}

// FindGraphQLFiles finds all files with kind: GraphQL and a non-empty url field
// in the given directory and its subdirectories.
// Returns a map where keys are URLs (can be templates like {{.baseUrl}}) and values are file paths.
// This ensures each unique URL/template is only represented once.
// NOTE: This function does NOT perform template substitution - URLs are returned as-is.
func FindGraphQLFiles(dirPath string) (map[string]string, error) {
	// Get all YAML/JSON files recursively
	allFiles, err := utils.ListFiles(dirPath)
	if err != nil {
		return nil, err
	}

	// Map of URL -> filePath to ensure uniqueness
	graphqlFiles := make(map[string]string)

	for _, filePath := range allFiles {
		// Skip response files early (performance optimization)
		if strings.Contains(filepath.Base(filePath), utils.ResponseBase) ||
			strings.Contains(filepath.Base(filePath), utils.ApiOptions) {
			continue
		}

		// Lightweight peek at kind field - no template substitution
		kind, err := peekKindField(filePath)
		if err != nil {
			// Silently skip malformed files
			continue
		}

		// Only process GraphQL files, and skip non Graphql files
		if !strings.EqualFold(kind, string(yamlparser.KindGraphQL)) {
			continue
		}

		// Peek at URL field - no template substitution
		url, err := peekURLField(filePath)
		if err != nil {
			// Silently skip files we can't read
			continue
		}

		// Skip files with empty URL (silently)
		if url == "" {
			continue
		}

		// Valid GraphQL file with non-empty URL (template or full URL)
		// Store URL -> filePath mapping (later files with same URL will overwrite)
		graphqlFiles[url] = filePath
	}

	if len(graphqlFiles) == 0 {
		return nil, fmt.Errorf(
			"no files with 'kind: GraphQL' and non-empty 'url' field found in directory: %s",
			dirPath,
		)
	}

	return graphqlFiles, nil
}

// ValidateGraphQLFile checks if a file exists, has kind: GraphQL, and has a non-empty url field.
// Returns the raw URL (template or full URL) without performing template substitution.
// This ensures consistent behavior with FindGraphQLFiles (Phase 1 - discovery only).
func ValidateGraphQLFile(filePath string) (string, bool, error) {
	// Clean the path
	filePath = filepath.Clean(filePath)

	// Check if file exists
	if !utils.FileExists(filePath) {
		return "", false, fmt.Errorf("file not found: %s", filePath)
	}

	// Peek at kind (fast check, no template substitution)
	kind, err := peekKindField(filePath)
	if err != nil {
		return "", false, fmt.Errorf("error reading file '%s': %w", filePath, err)
	}

	// Check if kind is GraphQL
	if !strings.EqualFold(kind, "GraphQL") {
		return "", false, fmt.Errorf("file '%s' does not have 'kind: GraphQL'", filePath)
	}

	// Peek at URL field (no template substitution)
	url, err := peekURLField(filePath)
	if err != nil {
		return "", false, fmt.Errorf("error reading URL from file '%s': %w", filePath, err)
	}

	// Check URL is non-empty
	if url == "" {
		return "", false, fmt.Errorf("file '%s' has empty or missing 'url' field", filePath)
	}

	// Return raw URL (could be template like {{.graphqlUrl}} or full URL)
	return url, true, nil
}

// resolveWorker processes work items from the workChan and sends results to resultChan
// Each worker runs independently and handles template resolution with timeout
func resolveWorker(
	wg *sync.WaitGroup,
	workChan <-chan workItem,
	resultChan chan<- ResolutionResult,
	secretsMap map[string]any,
	timeout time.Duration,
) {
	defer wg.Done()

	for work := range workChan {
		result := ResolutionResult{
			RawURL:   work.rawURL,
			FilePath: work.filePath,
		}

		// Check if resolution is needed
		if !strings.Contains(work.rawURL, "{{") {
			// No template, validate and pass through
			url := yamlparser.URL(work.rawURL)
			if !url.IsValidURL() {
				result.Error = fmt.Errorf("invalid URL '%s'", work.rawURL)
			} else {
				result.ResolvedURL = work.rawURL
			}
			resultChan <- result
			continue
		}

		// Process with timeout (matching init.go pattern)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		doneChan := make(chan struct{})
		errChan := make(chan error, 1)
		var resolvedURL string

		// Execute resolution in goroutine
		go func() {
			apiInfo, err := ProcessGraphQLFile(work.filePath, secretsMap)
			if err != nil {
				errChan <- err
			} else {
				resolvedURL = apiInfo.Url
				close(doneChan)
			}
		}()

		// Wait for completion or timeout
		select {
		case <-doneChan:
			// Validate resolved URL
			url := yamlparser.URL(resolvedURL)
			if !url.IsValidURL() {
				result.Error = fmt.Errorf("invalid resolved URL '%s'", resolvedURL)
			} else {
				result.ResolvedURL = resolvedURL
			}
		case err := <-errChan:
			result.Error = fmt.Errorf("error processing file: %w", err)
		case <-ctx.Done():
			result.Error = fmt.Errorf("timeout after %v", timeout)
		}

		cancel()
		resultChan <- result
	}
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

// ResolveTemplateURLsConcurrent processes multiple GraphQL files concurrently
// using a worker pool pattern. It resolves template URLs and collects all results
// and errors, allowing partial success (some files can succeed while others fail).
// Returns a ResolutionSummary containing successful and failed resolutions.
func ResolveTemplateURLsConcurrent(
	urlToFileMap map[string]string,
	secretsMap map[string]any,
) (*ResolutionSummary, error) {
	// Handle edge case: empty input
	if len(urlToFileMap) == 0 {
		return &ResolutionSummary{TotalFiles: 0}, nil
	}

	// Fast path: If no templates needed, just validate URLs
	hasTemplates := false
	for url := range urlToFileMap {
		if strings.Contains(url, "{{") {
			hasTemplates = true
			break
		}
	}

	if !hasTemplates {
		// No templates, just validate URLs and return
		summary := &ResolutionSummary{TotalFiles: len(urlToFileMap)}
		for rawURL, filePath := range urlToFileMap {
			result := ResolutionResult{
				RawURL:   rawURL,
				FilePath: filePath,
			}

			url := yamlparser.URL(rawURL)
			if !url.IsValidURL() {
				result.Error = fmt.Errorf("invalid URL '%s'", rawURL)
				summary.Failed = append(summary.Failed, result)
			} else {
				result.ResolvedURL = rawURL
				summary.Successful = append(summary.Successful, result)
			}
		}
		return summary, nil
	}

	// Configuration
	numOfFiles := len(urlToFileMap)
	maxWorkers := utils.GetWorkers(&numOfFiles)
	timeout := 30 * time.Second // Per-file timeout

	// Channels for work distribution and result collection
	workChan := make(chan workItem, len(urlToFileMap))
	resultChan := make(chan ResolutionResult, len(urlToFileMap))

	var wg sync.WaitGroup

	// Fill work channel
	for rawURL, filePath := range urlToFileMap {
		workChan <- workItem{rawURL: rawURL, filePath: filePath}
	}
	close(workChan)

	// Start worker pool
	for range maxWorkers {
		wg.Add(1)
		go resolveWorker(&wg, workChan, resultChan, secretsMap, timeout)
	}

	// Close result channel when all workers complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	summary := &ResolutionSummary{
		TotalFiles: len(urlToFileMap),
	}

	for result := range resultChan {
		if result.Error != nil {
			summary.Failed = append(summary.Failed, result)
		} else {
			summary.Successful = append(summary.Successful, result)
		}
	}

	return summary, nil
}
