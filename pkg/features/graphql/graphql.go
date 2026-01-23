package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// Introspect is the CLI handler for 'hulak gql' subcommand.
// Supported usage:
//   - hulak gql           (shows help)
//   - hulak gql .         (directory mode - find all GraphQL files)
//   - hulak gql <path>    (file mode - validate specific file)
func Introspect(args []string) {
	// No args = show help and return
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

// handleDirectoryMode finds all GraphQL files in CWD and resolves template URLs if needed.
func handleDirectoryMode() {
	cwd, err := os.Getwd()
	if err != nil {
		utils.PanicRedAndExit("Error getting current directory: %v", err)
	}

	urlToFileMap, err := FindGraphQLFiles(cwd)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	// Check if any URLs contain template variables that need env resolution
	if needsEnvResolution(urlToFileMap) {
		secretsMap, cancelled := loadSecretsWithEnvSelector()
		if cancelled {
			return
		}

		// Resolve template URLs
		urlToFileMap, err = ResolveTemplateURLs(urlToFileMap, secretsMap)
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}
	}

	// Display results
	printGraphQLFiles(urlToFileMap)
}

// handleFileMode validates and processes a specific GraphQL file.
func handleFileMode(arg string) {
	filePath := filepath.Clean(arg)

	// First validate the file has kind: GraphQL and a URL
	rawURL, isValid, err := ValidateGraphQLFile(filePath)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	if !isValid {
		utils.PanicRedAndExit("File validation failed unexpectedly")
	}

	var apiInfo yamlparser.ApiInfo

	// Check if URL contains template variables that need env resolution
	if strings.Contains(rawURL, "{{") {
		secretsMap, cancelled := loadSecretsWithEnvSelector()
		if cancelled {
			return
		}

		// Process file with template resolution
		apiInfo, err = ProcessGraphQLFile(filePath, secretsMap)
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}
	} else {
		// No templates - process with empty secrets map
		apiInfo, err = ProcessGraphQLFile(filePath, map[string]any{})
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}
	}

	// Display result
	fmt.Println("\nGraphQL file:")
	fmt.Printf("  URL:     %s\n", apiInfo.Url)
	fmt.Printf("  Method:  %s\n", apiInfo.Method)
	fmt.Printf("  Headers: %v\n", apiInfo.Headers)
	fmt.Printf("  File:    %s\n", filePath)
}

// needsEnvResolution checks if any URL in the map contains template variables.
// This catches all template types: {{.key}}, {{getValueOf key fileName}}, {{getFile fileName}} since they all start with "{{".
func needsEnvResolution(urlToFileMap map[string]string) bool {
	for url := range urlToFileMap {
		if strings.Contains(url, "{{") {
			return true
		}
	}
	return false
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

// printGraphQLFiles displays the discovered GraphQL files and their URLs.
//
//	TODO-gql: Remove later
func printGraphQLFiles(urlToFileMap map[string]string) {
	fmt.Println("GraphQL files found:")
	for url, filePath := range urlToFileMap {
		fmt.Printf("  URL:  %s\n", url)
		fmt.Printf("  File: %s\n\n", filePath)
	}
	fmt.Printf("Total: %d unique GraphQL endpoint(s)\n", len(urlToFileMap))
}
