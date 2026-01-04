package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
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

// hasValidURLField checks if a YAML file has a non-empty url field.
// It only checks for presence and non-empty value, not URL validity.
// This allows templates like {{.baseUrl}} to pass validation.
func hasValidURLField(filePath string, _ map[string]any) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	var data map[string]any
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return false
	}

	// Convert keys to lowercase (consistent with yamlparser behavior)
	data = utils.ConvertKeysToLowerCase(data)

	// Check if url field exists
	urlValue, exists := data["url"]
	if !exists {
		return false
	}

	// Check if url is not empty
	switch url := urlValue.(type) {
	case string:
		// Trim whitespace and check non-empty
		return strings.TrimSpace(url) != ""
	default:
		// Non-string types (null, number, bool) are invalid
		return false
	}
}

// FindGraphQLFiles finds all files with kind: GraphQL and a valid url field
// in the given directory and its subdirectories.
// Returns a list of absolute file paths.
func FindGraphQLFiles(dirPath string, secretsMap map[string]any) ([]string, error) {
	// Get all YAML/JSON files recursively
	allFiles, err := utils.ListFiles(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error listing files in '%s': %w", dirPath, err)
	}

	var graphqlFiles []string

	for _, filePath := range allFiles {
		// Skip response files early (performance optimization)
		if strings.Contains(filepath.Base(filePath), utils.ResponseBase) {
			continue
		}

		// Check kind field first to avoid template substitution
		// errors from non-GraphQL files (API, Auth, etc.)
		kind, err := peekKindField(filePath)
		if err != nil {
			// Warn about unreadable files but continue checking others
			utils.PrintWarning(fmt.Sprintf(
				"Warning: Could not read file '%s':\n  %v",
				filepath.Base(filePath), err))
			continue
		}

		if !strings.EqualFold(kind, "GraphQL") {
			continue // Not a GraphQL file, skip it
		}

		// Full parsing with template substitution for GraphQL files only
		// Template errors displayed here are useful for debugging
		config, err := yamlparser.ParseConfig(filePath, secretsMap)
		if err != nil {
			// Warn but continue - other GraphQL files might be valid
			utils.PrintWarning(fmt.Sprintf(
				"Warning: Could not parse GraphQL file '%s'",
				filepath.Base(filePath)))
			continue
		}

		// Safety check after full parsing
		if !config.IsGraphql() {
			continue
		}

		// Check if it has a valid URL field
		if !hasValidURLField(filePath, secretsMap) {
			continue
		}

		graphqlFiles = append(graphqlFiles, filePath)
	}

	if len(graphqlFiles) == 0 {
		return nil, fmt.Errorf(
			"no files with 'kind: GraphQL' and 'url' field found in directory: %s",
			dirPath,
		)
	}

	return graphqlFiles, nil
}

// ValidateGraphQLFile checks if a file exists, has kind: GraphQL, and has a valid url field.
// Returns true if valid, error with description if invalid.
func ValidateGraphQLFile(filePath string, secretsMap map[string]any) (bool, error) {
	// Clean the path
	filePath = filepath.Clean(filePath)

	// Check if file exists
	if !utils.FileExists(filePath) {
		return false, fmt.Errorf("file not found: %s", filePath)
	}

	// Parse the file
	config, err := yamlparser.ParseConfig(filePath, secretsMap)
	if err != nil {
		return false, fmt.Errorf("error parsing file '%s': %w", filePath, err)
	}

	// Check if kind is GraphQL
	if !config.IsGraphql() {
		return false, fmt.Errorf("file '%s' does not have 'kind: GraphQL'", filePath)
	}

	// Check if it has a valid URL field
	if !hasValidURLField(filePath, secretsMap) {
		return false, fmt.Errorf(
			"file '%s' is missing required 'url' field for GraphQL introspection",
			filePath,
		)
	}

	return true, nil
}

// Introspect is a placeholder CLI handler for 'hulak gql' subcommand.
// Any additional arguments after the first are silently ignored.
func Introspect(args []string) {
	secretsMap := make(map[string]any)

	// Determine mode based on first argument only
	// Ignore any additional arguments
	var mode string
	var targetPath string

	if len(args) == 0 {
		utils.PrintWarning("GraphQL Usage:")
		_ = utils.WriteCommandHelp([]*utils.CommandHelp{
			{Command: "hulak gql .", Description: "Find All GraphQL in current CWD"},
			{Command: "hulak gql <path/to/file>", Description: "Use provided specific path"},
		},
		)
	} else {
		firstArg := args[0]
		if firstArg == "." {
			mode = "directory"
		} else {
			// Assume it's a file path
			mode = "file"
			targetPath = firstArg
		}
	}

	if mode == "directory" {
		// Directory mode: find all GraphQL files in CWD
		cwd, err := os.Getwd()
		if err != nil {
			utils.PanicRedAndExit("Error getting current directory: %v", err)
		}

		files, err := FindGraphQLFiles(cwd, secretsMap)
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}

		// Placeholder output - just print the list
		fmt.Println("GraphQL files found:")
		for _, file := range files {
			fmt.Println(file)
		}
	} else {
		// File mode: validate specific file
		filePath := filepath.Clean(targetPath)

		isValid, err := ValidateGraphQLFile(filePath, secretsMap)
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}

		if !isValid {
			utils.PanicRedAndExit("File validation failed unexpectedly")
		}

		// Placeholder output
		fmt.Printf("Valid GraphQL file: %s\n", filePath)
	}
}
