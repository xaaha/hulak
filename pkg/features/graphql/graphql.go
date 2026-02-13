package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// Introspect is the CLI handler for 'hulak gql' subcommand.
// Supported usage:
//   - hulak gql                     (shows help)
//   - hulak gql .                   (directory mode - interactive env selector)
//   - hulak gql -env prod .         (directory mode - uses 'prod' env)
//   - hulak gql <path>              (file mode - interactive env selector)
//   - hulak gql -env staging <path>  (file mode - uses 'staging' env)
//
// The env parameter is optional. If empty, shows the interactive env selector.
// If provided, skips the selector and uses the specified environment.
func Introspect(args []string, env string) {
	if len(args) == 0 {
		utils.PrintGQLUsage()
		return
	}

	firstArg := args[0]
	if firstArg == "." {
		handleDirectoryMode(env)
	} else {
		handleFileMode(firstArg, env)
	}
}

// handleDirectoryMode finds all GraphQL files in CWD and processes them concurrently.
// If env is provided, uses that environment. Otherwise shows the interactive selector.
func handleDirectoryMode(env string) {
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

	// Get secrets - use provided env or show selector if needed
	secretsMap := GetSecretsForEnv(urlToFileMap, env)
	if secretsMap == nil {
		return // User cancelled
	}

	// Process all files concurrently
	results := ProcessFilesConcurrent(filePaths, secretsMap)

	// TODO-gql: All printing below is temporary for Phase 1.
	// Phase 2 will pass schemas to TUI for interactive browsing.
	// Introspect each endpoint and display schema
	fmt.Printf("\nFound %d GraphQL endpoint(s)\n", len(results))
	fmt.Println(strings.Repeat("=", 60))

	successCount := 0
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("\n❌ Error processing %s: %v\n", result.FilePath, result.Error)
			continue
		}

		fmt.Printf("\nEndpoint: %s\n", result.ApiInfo.Url)
		fmt.Printf("File: %s\n", filepath.Base(result.FilePath))
		fmt.Println(strings.Repeat("-", 60))

		// Fetch and display schema
		schema, err := FetchAndParseSchema(result.ApiInfo)
		if err != nil {
			fmt.Printf("❌ Failed to fetch schema: %v\n", err)
			continue
		}

		DisplaySchema(schema)
		successCount++
		fmt.Println(strings.Repeat("=", 60))
	}

	fmt.Printf("\n✓ Successfully fetched %d/%d schema(s)\n", successCount, len(results))
}

// FetchAndParseSchema makes an introspection query and parses the schema.
// It takes an ApiInfo, sets the introspection query as the body, makes the HTTP call,
// parses the response, and converts it to our domain Schema model.
func FetchAndParseSchema(apiInfo yamlparser.ApiInfo) (Schema, error) {
	// Prepare introspection query body
	introspectionBody := map[string]any{"query": IntrospectionQuery}
	jsonData, err := json.Marshal(introspectionBody)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to marshal introspection query: %w", err)
	}

	// Set the body
	apiInfo.Body = bytes.NewReader(jsonData)

	// Make the HTTP call
	resp, err := apicalls.StandardCall(apiInfo, false)
	if err != nil {
		return Schema{}, fmt.Errorf("introspection request failed: %w", err)
	}

	// Extract response body
	if resp.Response == nil {
		return Schema{}, fmt.Errorf("no response data received")
	}

	// Convert body to JSON bytes
	var bodyBytes []byte
	switch v := resp.Response.Body.(type) {
	case string:
		bodyBytes = []byte(v)
	case []byte:
		bodyBytes = v
	default:
		// Body might already be parsed JSON, marshal it back
		bodyBytes, err = json.Marshal(v)
		if err != nil {
			return Schema{}, fmt.Errorf("failed to process response body: %w", err)
		}
	}

	// Parse introspection response
	introspectionData, err := ParseIntrospectionResponse(bodyBytes)
	if err != nil {
		return Schema{}, err
	}

	// Convert to domain model
	schema, err := ConvertToSchema(introspectionData)
	if err != nil {
		return Schema{}, err
	}

	return schema, nil
}

// handleFileMode validates and processes a specific GraphQL file.
// If env is provided, uses that environment. Otherwise shows the interactive selector.
func handleFileMode(arg string, env string) {
	filePath := filepath.Clean(arg)

	// Validate the file has kind: GraphQL and a URL
	rawURL, isValid, err := ValidateGraphQLFile(filePath)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	if !isValid {
		utils.PanicRedAndExit("File validation failed unexpectedly")
	}

	// Get secrets - use provided env or show selector if needed
	var secretsMap map[string]any
	if strings.Contains(rawURL, "{{") {
		secretsMap = loadSecretsForEnv(env)
		if secretsMap == nil {
			return // User cancelled
		}
	} else {
		secretsMap = map[string]any{}
	}

	// Process single file using the same concurrent function (1 worker)
	results := ProcessFilesConcurrent([]string{filePath}, secretsMap)

	if len(results) == 0 {
		utils.PanicRedAndExit("No results returned")
	}

	result := results[0]
	if result.Error != nil {
		utils.PanicRedAndExit("Error processing file: %v", result.Error)
	}

	// TODO-gql: All printing below is temporary for Phase 1.
	// Phase 2 will pass schema to TUI for interactive browsing.
	fmt.Printf("\nFetching schema from: %s\n", result.ApiInfo.Url)

	// Fetch and display schema
	schema, err := FetchAndParseSchema(result.ApiInfo)
	if err != nil {
		utils.PanicRedAndExit("Failed to fetch schema: %v", err)
	}

	DisplaySchema(schema)
	fmt.Println("\n✓ Schema introspection completed successfully")
}

// GetSecretsForEnv checks if any URL needs template resolution and loads secrets.
// If env is provided, uses that environment directly.
// If env is empty and templates are needed, shows the interactive selector.
// Returns empty map if no templates needed, nil if user cancelled.
func GetSecretsForEnv(urlToFileMap map[string]string, env string) map[string]any {
	// If env is explicitly provided, always load it (user knows what they want)
	if env != "" {
		return loadSecretsForEnv(env)
	}

	// No env provided - check if we need secrets at all
	if !NeedsEnvResolution(urlToFileMap) {
		return map[string]any{}
	}

	// Need secrets but no env provided - show interactive selector
	return loadSecretsForEnv("")
}

// loadSecretsForEnv loads secrets from the specified environment.
// If env is empty, shows the interactive env selector TUI.
// Returns nil if user cancelled the selector.
func loadSecretsForEnv(env string) map[string]any {
	selectedEnv := env

	// If no env provided, show the interactive selector
	if selectedEnv == "" {
		var err error
		selectedEnv, err = envselect.RunEnvSelector()
		if err != nil {
			utils.PanicRedAndExit("Environment selector error: %v", err)
		}
		if selectedEnv == "" {
			fmt.Println("Environment selection cancelled.")
			return nil
		}
	}

	// Load secrets from the environment
	secretsMap, err := envparser.LoadSecretsMap(selectedEnv)
	if err != nil {
		utils.PanicRedAndExit("Failed to load environment '%s': %v", selectedEnv, err)
	}

	return secretsMap
}
