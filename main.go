// Package main initializes the project and runs the query
package main

import (
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
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
		if !isInteractiveTerminal() {
			utils.PanicRedAndExit(
				"interactive mode requires a TTY. Use -f, -fp, -dir, or -dirseq flags. See 'hulak help'",
			)
		}

		if !flags.EnvSet {
			if err = envparser.CreateDefaultEnvs(nil); err != nil {
				utils.PanicRedAndExit("%v", err)
			}
		}
		fp = runInteractiveFlow(&env, flags.EnvSet)
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
func runInteractiveFlow(env *string, envSet bool) string {
	if !envSet {
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

func isInteractiveTerminal() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	stdinTTY := (stdinInfo.Mode() & os.ModeCharDevice) != 0
	stdoutTTY := (stdoutInfo.Mode() & os.ModeCharDevice) != 0

	return stdinTTY && stdoutTTY
}
