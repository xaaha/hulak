// Package graphql provides query management capabilities
package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
)

const (
	queriesDir     = "graphql_queries"
	queryExtension = ".graphql"
	metaExtension  = ".meta.yaml"
)

// QueryMetadata stores metadata about a saved query
type QueryMetadata struct {
	Name        string         `json:"name"        yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Endpoint    string         `json:"endpoint"    yaml:"endpoint"`
	Variables   map[string]any `json:"variables"   yaml:"variables"`
	CreatedAt   time.Time      `json:"created_at"  yaml:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"  yaml:"updated_at"`
	Tags        []string       `json:"tags"        yaml:"tags"`
}

// QueryManager manages GraphQL queries
type QueryManager struct {
	queriesPath string
}

// NewQueryManager creates a new query manager
func NewQueryManager() (*QueryManager, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, utils.ColorError("failed to get working directory: %w", err)
	}

	queriesPath := filepath.Join(root, queriesDir)

	// Create queries directory if it doesn't exist
	if err := os.MkdirAll(queriesPath, utils.DirPer); err != nil {
		return nil, utils.ColorError("failed to create queries directory: %w", err)
	}

	return &QueryManager{
		queriesPath: queriesPath,
	}, nil
}

// SaveQuery saves a query to disk
func (qm *QueryManager) SaveQuery(name, query string, metadata QueryMetadata) error {
	if name == "" {
		return utils.ColorError("query name cannot be empty")
	}

	// Sanitize filename
	safeName := sanitizeFilename(name)

	// Validate query
	if err := ValidateQuery(query); err != nil {
		return utils.ColorError("invalid query: %w", err)
	}

	// Update metadata timestamps
	now := time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = now
	}
	metadata.UpdatedAt = now
	metadata.Name = name

	// Save query file
	queryPath := filepath.Join(qm.queriesPath, safeName+queryExtension)
	if err := os.WriteFile(queryPath, []byte(query), utils.FilePer); err != nil {
		return utils.ColorError("failed to write query file: %w", err)
	}

	// Save metadata file
	metaPath := filepath.Join(qm.queriesPath, safeName+metaExtension)
	metaBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return utils.ColorError("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaBytes, utils.FilePer); err != nil {
		return utils.ColorError("failed to write metadata file: %w", err)
	}

	return nil
}

// LoadQuery loads a query from disk
func (qm *QueryManager) LoadQuery(name string) (string, QueryMetadata, error) {
	safeName := sanitizeFilename(name)

	// Load query
	queryPath := filepath.Join(qm.queriesPath, safeName+queryExtension)
	queryBytes, err := os.ReadFile(queryPath)
	if err != nil {
		return "", QueryMetadata{}, utils.ColorError("failed to read query file: %w", err)
	}

	// Load metadata
	metaPath := filepath.Join(qm.queriesPath, safeName+metaExtension)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return string(queryBytes), QueryMetadata{Name: name}, nil // Return query without metadata
	}

	var metadata QueryMetadata
	if err := yaml.Unmarshal(metaBytes, &metadata); err != nil {
		return string(queryBytes), QueryMetadata{Name: name}, nil // Return query without metadata
	}

	return string(queryBytes), metadata, nil
}

// ListQueries lists all saved queries
func (qm *QueryManager) ListQueries() ([]SavedQuery, error) {
	files, err := os.ReadDir(qm.queriesPath)
	if err != nil {
		return nil, utils.ColorError("failed to read queries directory: %w", err)
	}

	queries := make([]SavedQuery, 0)
	processedNames := make(map[string]bool)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Only process .graphql files
		if !strings.HasSuffix(filename, queryExtension) {
			continue
		}

		// Get base name without extension
		baseName := strings.TrimSuffix(filename, queryExtension)

		// Skip if already processed
		if processedNames[baseName] {
			continue
		}
		processedNames[baseName] = true

		// Load query and metadata
		query, metadata, err := qm.LoadQuery(baseName)
		if err != nil {
			utils.PrintWarning(fmt.Sprintf("Failed to load query %s: %v", baseName, err))
			continue
		}

		queries = append(queries, SavedQuery{
			Name:        metadata.Name,
			Description: metadata.Description,
			Query:       query,
			Variables:   metadata.Variables,
			FilePath:    filepath.Join(qm.queriesPath, filename),
		})
	}

	return queries, nil
}

// DeleteQuery deletes a saved query
func (qm *QueryManager) DeleteQuery(name string) error {
	safeName := sanitizeFilename(name)

	// Delete query file
	queryPath := filepath.Join(qm.queriesPath, safeName+queryExtension)
	if err := os.Remove(queryPath); err != nil && !os.IsNotExist(err) {
		return utils.ColorError("failed to delete query file: %w", err)
	}

	// Delete metadata file
	metaPath := filepath.Join(qm.queriesPath, safeName+metaExtension)
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return utils.ColorError("failed to delete metadata file: %w", err)
	}

	return nil
}

// UpdateQuery updates an existing query
func (qm *QueryManager) UpdateQuery(name, newQuery string, metadata QueryMetadata) error {
	// Check if query exists
	if _, _, err := qm.LoadQuery(name); err != nil {
		return utils.ColorError("query not found: %w", err)
	}

	// Save updated query (this will update timestamps)
	return qm.SaveQuery(name, newQuery, metadata)
}

// ExportQuery exports a query to a hulak yaml file
func (qm *QueryManager) ExportQuery(name, endpoint string, headers map[string]string) error {
	query, metadata, err := qm.LoadQuery(name)
	if err != nil {
		return utils.ColorError("failed to load query: %w", err)
	}

	// Create hulak yaml structure
	hulakFile := map[string]any{
		"method": "POST",
		"url":    endpoint,
		"headers": map[string]string{
			"Content-Type": "application/json",
		},
		"body": map[string]any{
			"graphql": map[string]any{
				"query":     query,
				"variables": metadata.Variables,
			},
		},
	}

	// Merge custom headers
	if headers != nil {
		for k, v := range headers {
			hulakFile["headers"].(map[string]string)[k] = v
		}
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(hulakFile)
	if err != nil {
		return utils.ColorError("failed to marshal to YAML: %w", err)
	}

	// Save to file
	safeName := sanitizeFilename(name)
	outputPath := filepath.Join(qm.queriesPath, safeName+".hk.yaml")

	if err := os.WriteFile(outputPath, yamlBytes, utils.FilePer); err != nil {
		return utils.ColorError("failed to write output file: %w", err)
	}

	utils.PrintGreen(fmt.Sprintf("Exported query to: %s", outputPath))
	return nil
}

// SearchQueries searches queries by name, description, or tags
func (qm *QueryManager) SearchQueries(searchTerm string) ([]SavedQuery, error) {
	allQueries, err := qm.ListQueries()
	if err != nil {
		return nil, err
	}

	searchTerm = strings.ToLower(searchTerm)
	results := make([]SavedQuery, 0)

	for _, query := range allQueries {
		// Search in name
		if strings.Contains(strings.ToLower(query.Name), searchTerm) {
			results = append(results, query)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(query.Description), searchTerm) {
			results = append(results, query)
			continue
		}

		// Search in query text
		if strings.Contains(strings.ToLower(query.Query), searchTerm) {
			results = append(results, query)
			continue
		}
	}

	return results, nil
}

// GetQueryStats returns statistics about saved queries
func (qm *QueryManager) GetQueryStats() (map[string]int, error) {
	queries, err := qm.ListQueries()
	if err != nil {
		return nil, err
	}

	stats := map[string]int{
		"total":     len(queries),
		"queries":   0,
		"mutations": 0,
	}

	for _, query := range queries {
		queryLower := strings.ToLower(query.Query)
		if strings.Contains(queryLower, "mutation") {
			stats["mutations"]++
		} else {
			stats["queries"]++
		}
	}

	return stats, nil
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)

	return strings.ToLower(replacer.Replace(name))
}

// ValidateQueryVariables validates that all variables in the query are provided
func ValidateQueryVariables(query string, variables map[string]any) error {
	// Extract variable names from query
	requiredVars := extractVariableNames(query)

	// Check if all required variables are provided
	for _, varName := range requiredVars {
		if _, ok := variables[varName]; !ok {
			return utils.ColorError(fmt.Sprintf("missing required variable: %s", varName))
		}
	}

	return nil
}

// extractVariableNames extracts variable names from a GraphQL query
func extractVariableNames(query string) []string {
	vars := make([]string, 0)
	inVariable := false
	currentVar := ""

	for i := 0; i < len(query); i++ {
		if query[i] == '$' {
			inVariable = true
			currentVar = ""
			continue
		}

		if inVariable {
			if (query[i] >= 'a' && query[i] <= 'z') ||
				(query[i] >= 'A' && query[i] <= 'Z') ||
				(query[i] >= '0' && query[i] <= '9') ||
				query[i] == '_' {
				currentVar += string(query[i])
			} else {
				if currentVar != "" {
					vars = append(vars, currentVar)
				}
				inVariable = false
				currentVar = ""
			}
		}
	}

	if currentVar != "" {
		vars = append(vars, currentVar)
	}

	return vars
}
