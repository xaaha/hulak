package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
)

// Introspect is the CLI handler for 'hulak gql' subcommand.
// Supported usage:
//   - hulak gql           (shows help)
//   - hulak gql .         (directory mode - find all GraphQL files)
//   - hulak gql <path>    (file mode - validate specific file)
func Introspect(args []string) {
	if len(args) == 0 {
		utils.PrintGQLUsage()
		return
	}

	firstArg := args[0]
	if firstArg == "." {
		handleDirectoryMode()
	} else {
		handleFileMode(firstArg)
	}
}

// handleDirectoryMode finds all GraphQL files in CWD and processes them concurrently.
func handleDirectoryMode() {
	cwd, err := os.Getwd()
	if err != nil {
		utils.PanicRedAndExit("Error getting current directory: %v", err)
	}

	urlToFileMap, err := FindGraphQLFiles(cwd)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	// Extract file paths from the map
	filePaths := make([]string, 0, len(urlToFileMap))
	for _, fp := range urlToFileMap {
		filePaths = append(filePaths, fp)
	}

	// Get secrets if any file needs template resolution
	secretsMap := getSecretsIfNeeded(urlToFileMap)
	if secretsMap == nil {
		return // User cancelled
	}

	// Process all files concurrently
	results := ProcessFilesConcurrent(filePaths, secretsMap)
	printResultsAndErrors(results)
}

// handleFileMode validates and processes a specific GraphQL file.
func handleFileMode(arg string) {
	filePath := filepath.Clean(arg)

	// Validate the file has kind: GraphQL and a URL
	rawURL, isValid, err := ValidateGraphQLFile(filePath)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	if !isValid {
		utils.PanicRedAndExit("File validation failed unexpectedly")
	}

	// Get secrets if template resolution is needed
	var secretsMap map[string]any
	if strings.Contains(rawURL, "{{") {
		secretsMap, _ = loadSecretsWithEnvSelector()
		if secretsMap == nil {
			return // User cancelled
		}
	} else {
		secretsMap = map[string]any{}
	}

	// Process single file using the same concurrent function (1 worker)
	results := ProcessFilesConcurrent([]string{filePath}, secretsMap)
	printResultsAndErrors(results)
}

// getSecretsIfNeeded checks if any URL needs template resolution and loads secrets.
// Returns empty map if no templates needed, nil if user cancelled.
func getSecretsIfNeeded(urlToFileMap map[string]string) map[string]any {
	if !NeedsEnvResolution(urlToFileMap) {
		return map[string]any{}
	}

	secretsMap, cancelled := loadSecretsWithEnvSelector()
	if cancelled {
		return nil
	}
	return secretsMap
}

// printResultsAndErrors prints successful results and collects errors to print at the end.
func printResultsAndErrors(results []ProcessResult) {
	var errors []ProcessResult

	fmt.Println("\nGraphQL files:")
	for _, r := range results {
		if r.Error != nil {
			errors = append(errors, r)
			continue
		}
		fmt.Printf("  URL:     %s\n", r.ApiInfo.Url)
		fmt.Printf("  Method:  %s\n", r.ApiInfo.Method)
		fmt.Printf("  Headers: %v\n", r.ApiInfo.Headers)
		fmt.Printf("  File:    %s\n\n", r.FilePath)
	}

	successCount := len(results) - len(errors)
	fmt.Printf("Total: %d file(s) processed successfully\n", successCount)

	// Print errors at the end
	if len(errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(errors))
		for _, e := range errors {
			fmt.Printf("  File:  %s\n", e.FilePath)
			fmt.Printf("  Error: %v\n\n", e.Error)
		}
	}
}

// loadSecretsWithEnvSelector shows the env selector TUI and loads secrets.
// Returns the secrets map and a boolean indicating if selection was cancelled.
func loadSecretsWithEnvSelector() (map[string]any, bool) {
	selectedEnv, err := envselect.RunEnvSelector()
	if err != nil {
		utils.PanicRedAndExit("Environment selector error: %v", err)
	}
	if selectedEnv == "" {
		fmt.Println("Environment selection cancelled.")
		return nil, true
	}

	// Load secrets from selected environment (no interactive prompts)
	secretsMap, err := envparser.LoadSecretsMap(selectedEnv)
	if err != nil {
		utils.PanicRedAndExit("Failed to load environment '%s': %v", selectedEnv, err)
	}

	return secretsMap, false
}
