// Package main initializes the project and runs the query
package main

import (
	"fmt"

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

	// Initialize project environment
	envMap := InitializeProject(env)

	// Which mode are we operating in
	hasDirFlags := dir != "" || dirseq != ""
	hasFileFlags := fp != "" || fileName != ""

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
		HandleAPIRequests(envMap, debug, filePathList, dir, dirseq)
	} else {
		utils.PrintWarning("No file or directory specified. Use -file, -fp, -dir, or -dirseq flags.")
	}
}
