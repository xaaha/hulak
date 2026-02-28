// Package main initializes the project and runs the query
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/tui/apicaller"
	"github.com/xaaha/hulak/pkg/tui/envselect"
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

	// Check if this directory is set up for hulak
	if !utils.IsHulakProject() {
		ensureHulakProject()
	}
	// Which mode are we operating in
	hasDirFlags := dir != "" || dirseq != ""
	hasFileFlags := fp != "" || fileName != ""

	if !hasFileFlags && !hasDirFlags {
		if !isInteractiveTerminal() {
			utils.PanicRedAndExit(
				"interactive mode requires a TTY. Use -f, -fp, -dir, or -dirseq flags. See 'hulak help'",
			)
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
	envItems := envselect.EnvItems()
	if !envSet && len(envItems) == 0 {
		utils.PanicRedAndExit("%v", envselect.NoEnvFilesError())
	}

	fileItems, err := tui.FileItems()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	if len(fileItems) == 0 {
		utils.PanicRedAndExit("%v", tui.NoFilesError())
	}

	selection, err := apicaller.RunFilePath(envItems, fileItems, *env, envSet)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	if selection.Cancelled || selection.File == "" {
		os.Exit(0)
	}

	if selection.Env != "" {
		*env = selection.Env
	}

	return selection.File
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

/*
ensureHulakProject checks if the current directory is set up for hulak.
If not, it prompts the user to initialize (interactive) or exits with
instructions (non-interactive), similar to git's "not a git repository" check.
*/
func ensureHulakProject() {
	if !isInteractiveTerminal() {
		utils.PanicRedAndExit(
			"fatal: not a hulak project (env/ directory not found)\n\nRun 'hulak init' to set up this directory",
		)
	}

	fmt.Printf(
		"%sfatal: not a hulak project (env/ directory not found)%s\n\n",
		utils.Yellow,
		utils.ColorReset,
	)
	fmt.Println("'hulak init' will set up this directory:")
	fmt.Println("  - Create an 'env/' directory for storing environment secrets")
	fmt.Println("  - Create a 'global.env' file with default environment")
	fmt.Printf("  - Create an '%s' example file for reference\n\n", utils.APIOptions)
	fmt.Print("Initialize hulak project here? [y/N] ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			fmt.Println()
			if err := userflags.InitDefaultProject(); err != nil {
				utils.PanicRedAndExit("%v", err)
			}
			fmt.Println()
			return
		}
	}

	fmt.Println("\nTo set up manually, run: hulak init")
	os.Exit(0)
}
