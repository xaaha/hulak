package graphql

import (
	"fmt"
	"os"
	"path/filepath"

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
		utils.PrintWarning("GraphQL Usage (Upcoming Feature):")
		_ = utils.WriteCommandHelp([]*utils.CommandHelp{
			{Command: "hulak gql .", Description: "Find All GraphQL files in current directory"},
			{Command: "hulak gql <path/to/file>", Description: "Validate a specific GraphQL file"},
		})
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

		// Display results
		fmt.Println("\nGraphQL files found:")
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
