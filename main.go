// Package main initializes the project and runs the query
package main

import (
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/tui/fileselect"
	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	// Parse command line flags and subcmds
	flags, err := userflags.ParseFlagsSubcmds()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	// Extract flags
	env := flags.Env
	fp := flags.FilePath
	fileName := flags.File
	debug := flags.Debug
	dir := flags.Dir
	dirseq := flags.Dirseq

	// Which mode are we operating in
	hasDirFlags := dir != "" || dirseq != ""
	hasFileFlags := fp != "" || fileName != ""

	if !hasFileFlags && !hasDirFlags {
		fp = runInteractiveFlow(&env)
		hasFileFlags = true
	}

	// Initialize project environment
	envMap := InitializeProject(env)

	var filePathList []string

	if hasFileFlags {
		filePathList, err = userflags.GenerateFilePathList(fileName, fp)
		if err != nil {
			// Only panic if no directory flags are provided
			if !hasDirFlags {
				utils.PanicRedAndExit("%v", err)
			} else {
				// When directory flags are present, just warn about the file flag error
				utils.PrintWarning(fmt.Sprintf("Warning with file flags: %v", err))
			}
		}
	}

	if hasFileFlags || hasDirFlags {
		HandleAPIRequests(envMap, debug, filePathList, dir, dirseq, fp)
	}
}

/*
runInteractiveFlow prompts the user to select an environment and file
when no file or directory flags are provided.
It updates env in-place if the user picks one, and returns the selected file path.
*/
func runInteractiveFlow(env *string) string {
	if *env == utils.DefaultEnvVal {
		selected, err := envselect.RunEnvSelector()
		if err != nil {
			utils.PanicRedAndExit("%v", err)
		}
		if selected == "" {
			os.Exit(0)
		}
		*env = selected
	}

	selected, err := fileselect.RunFileSelector()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	if selected == "" {
		os.Exit(0)
	}
	return selected
}
