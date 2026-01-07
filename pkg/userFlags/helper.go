// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"github.com/xaaha/hulak/pkg/utils"
)

// GenerateFilePathList returns a slice of file paths based on the flags -f and -fp.
func GenerateFilePathList(fileName string, fp string) ([]string, error) {
	standardErrMsg := "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' "

	// Both inputs are empty, return an error
	if fileName == "" && fp == "" {
		return nil, utils.ColorError(standardErrMsg)
	}

	var filePathList []string

	// Add file path from -fp flag if provided
	if fp != "" {
		filePathList = append(filePathList, fp)
	}

	// Add matching paths for -f flag if provided
	if fileName != "" {
		if matchingPaths, err := utils.ListMatchingFiles(fileName); err != nil {
			utils.PrintRed("helper.go: error occurred while collecting file paths " + err.Error())
		} else {
			filePathList = append(filePathList, matchingPaths...)
		}
	}

	if len(filePathList) == 0 {
		return nil, utils.ColorError(standardErrMsg)
	}
	return filePathList, nil
}

// Helper function to print command usage
func printHelp() {
	utils.PrintWarning("Api Usage:")
	_ = utils.WriteCommandHelp([]*utils.CommandHelp{
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
			Command:     "hulak  -fp path/tofile/getUser.yaml -debug",
			Description: "Run in global environment with debug mode",
		},
		{
			Command:     "hulak -env prod -dir path/to/dir ",
			Description: "Run all files in the directory concurrently",
		},
		{
			Command:     "hulak -env prod -dirseq path/to/dir ",
			Description: "Run all files in the directory alphabetically",
		},
	})

	_ = utils.WriteCommandHelp([]*utils.CommandHelp{
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
			Command:     "hulak  -fp path/tofile/getUser.yaml -debug",
			Description: "Run in global environment with debug mode",
		},
		{
			Command:     "hulak -env prod -dir path/to/dir ",
			Description: "Run all files in the directory concurrently",
		},
		{
			Command:     "hulak -env prod -dirseq path/to/dir ",
			Description: "Run all files in the directory alphabetically",
		},
	})

	// w.Flush()

	printHelpSubCommands()
}

// helper function to show valid subcommands
func printHelpSubCommands() {
	utils.PrintWarning("Subcommands:")
	_ = utils.WriteCommandHelp([]*utils.CommandHelp{
		{Command: "hulak version", Description: "Prints hulak version"},
		{
			Command:     "hulak init",
			Description: "Initializes default environment and creates an apiOptions.yaml file",
		},
		{
			Command:     "hulak init -env global prod test",
			Description: "Initializes specific environments",
		},
		{
			Command:     "hulak migrate <file1> <file2> ...",
			Description: "Migrates postman env and collections",
		},
		{
			Command:     "hulak import curl 'curl command' -o path/to/file.hk.yaml",
			Description: "Import cURL command and create Hulak YAML file",
		},
	})
}
