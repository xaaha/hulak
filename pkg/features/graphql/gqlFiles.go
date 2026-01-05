package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"

	"github.com/xaaha/hulak/pkg/utils"
)

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
		return nil, fmt.Errorf("error listing files in '%s': %w", dirPath, err)
	}

	// Map of URL -> filePath to ensure uniqueness
	graphqlFiles := make(map[string]string)

	for _, filePath := range allFiles {
		// Skip response files early (performance optimization)
		if strings.Contains(filepath.Base(filePath), utils.ResponseBase) {
			continue
		}

		// Lightweight peek at kind field - no template substitution
		kind, err := peekKindField(filePath)
		if err != nil {
			// Silently skip malformed files
			continue
		}

		// Only process GraphQL files
		if !strings.EqualFold(kind, "GraphQL") {
			continue // Skip non-GraphQL files silently
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
