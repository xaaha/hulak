package utils

import (
	"fmt"
)

func PrintGQLUsage() {
	PrintWarning("GraphQL Usage (Upcoming Feature):")
	_ = WriteCommandHelp([]*CommandHelp{
		{Command: "hulak gql .", Description: "Find All GraphQL files in current directory"},
		{Command: "hulak gql <path/to/file>", Description: "Validate a specific GraphQL file"},
	})
}

// helper function to show valid subcommands
func PrintHelpSubCommands() {
	PrintWarning("Subcommands:")
	_ = WriteCommandHelp([]*CommandHelp{
		{Command: "hulak version", Description: "Prints hulak version"},
		{
			Command: "hulak init",
			Description: fmt.Sprintf(
				"Initializes default environment and creates an '%s' file",
				ApiOptions,
			),
		},
		{
			Command:     "hulak init -env global prod test",
			Description: "Initializes specific environments",
		},
		{
			Command:     "hulak migrate <file1> <file2> ...",
			Description: "Migrates postman env and collections",
		},
	})
}

// Helper function to print command usage
func PrintHelp() {
	PrintWarning("Api Usage:")
	_ = WriteCommandHelp([]*CommandHelp{
		{Command: "hulak", Description: "Interactive mode: select environment and file"},
		{Command: "hulak -env staging", Description: "Interactive file picker with staging environment"},
		{Command: "hulak -env global -f fileName", Description: "Find and run all 'fileName'"},
		{
			Command:     "hulak -env staging -fp path/tofile/getUser.yaml",
			Description: "Run specific file with provided file path",
		},
		{
			Command:     "hulak -env prod -fp path/tofile/getUser.yaml -debug",
			Description: "Run in debug mode",
		},
		{
			Command:     "hulak -fp path/tofile/getUser.yaml -debug",
			Description: "Run in global environment with debug mode",
		},
		{
			Command:     "hulak -env prod -dir path/to/dir",
			Description: "Run all files in the directory concurrently",
		},
		{
			Command:     "hulak -env prod -dirseq path/to/dir",
			Description: "Run all files in the directory alphabetically",
		},
	})

	PrintHelpSubCommands()
	PrintGQLUsage()
}
