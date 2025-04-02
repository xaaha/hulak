// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/xaaha/hulak/pkg/utils"
)

// TODO: need to add -d for directory. -d-seq for running things in ascending order

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
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	writeCommandHelp(w, []*CommandHelp{
		{"hulak -env prod -f fileName", "Find and run all 'fileName'"},
		{"hulak -env prod -fp path/tofile/getUser.yaml", "Run specific file"},
		{"hulak -env prod -fp path/tofile/getUser.yaml -debug", "Run in debug mode"},
	})

	w.Flush()

	printHelpSubCommands()
}

// helper function to show valid subcommands
func printHelpSubCommands() {
	utils.PrintWarning("Subcommands:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	writeCommandHelp(w, []*CommandHelp{
		{"hulak init", "Initializes default environment"},
		{"hulak init -env global prod test", "Initializes specific environments"},
		{"hulak migrate <file1> <file2> ...", "Migrates specified files"},
	})

	w.Flush()
}

// CommandHelp holds a command and its description
type CommandHelp struct {
	Command     string
	Description string
}

// writeCommandHelp writes commands and descriptions with proper alignment
func writeCommandHelp(w *tabwriter.Writer, commands []*CommandHelp) {
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %s\t- %s\n", cmd.Command, cmd.Description)
	}
}
