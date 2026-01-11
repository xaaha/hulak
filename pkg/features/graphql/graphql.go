package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
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
		// Directory mode: find all GraphQL files in CWD
		cwd, err := os.Getwd()
		if err != nil {
			utils.PanicRedAndExit("Error getting current directory: %v", err)
		}

		urlToFileMap, err := FindGraphQLFiles(cwd)
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}

		// Check if any URLs contain template variables that need env resolution
		needsEnv := false
		for url := range urlToFileMap {
			if strings.Contains(url, "{{") {
				needsEnv = true
				break
			}
		}

		// Show env selector if templates need resolution
		if needsEnv {
			selectedEnv, err := envselect.RunEnvSelector()
			if err != nil {
				utils.PanicRedAndExit("Environment selector error: %v", err)
			}
			if selectedEnv == "" {
				fmt.Println("Environment selection cancelled.")
				return
			}

			// TODO: Clean up later. Just show what we've selected for now
			fmt.Printf("Selected environment: %s\n\n", selectedEnv)
		}

		// TEMP: CLEANUP LATER.. Display results for now
		fmt.Println("GraphQL files found:")
		for url, filePath := range urlToFileMap {
			fmt.Printf("  URL:  %s\n", url)
			fmt.Printf("  File: %s\n\n", filePath)
		}

		fmt.Printf("Total: %d unique GraphQL endpoint(s)\n", len(urlToFileMap))
	} else {
		// File mode: validate specific file
		filePath := filepath.Clean(firstArg)

		url, isValid, err := ValidateGraphQLFile(filePath)
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}

		if !isValid {
			utils.PanicRedAndExit("File validation failed unexpectedly")
		}

		// Display result
		fmt.Println("\nGraphQL file:")
		fmt.Printf("  URL:  %s\n", url)
		fmt.Printf("  File: %s\n", filePath)
	}
}
