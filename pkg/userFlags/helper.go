package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
)

// TODO: need to add -d for directory. -d-seq for running things in ascending order

// Returns a slice of file paths based on the flags -f and -fp.
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
	fmt.Println("  hulak -env prod -f fileName                    - Find and run all 'fileName'")
	fmt.Println("  hulak -env prod -fp path/tofile/getUser.yaml   - Run specific file")

	printHelpSubCommands()
}

// helper function to show valid subcommands
func printHelpSubCommands() {
	utils.PrintWarning("subcommand:")
	fmt.Println(
		"  hulak init                                     - Initializes default environment",
	)
	fmt.Println(
		"  hulak init -env global prod test               - Initializes specific environments",
	)
	fmt.Println("  hulak migrate <file1> <file2> ...              - Migrates specified files")
}
