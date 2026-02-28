// Package main initializes the project and runs the query
package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
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

	hasDirFlags := dir != "" || dirseq != ""
	hasFileFlags := fp != "" || fileName != ""

	// TUI mode: always requires env/ project structure
	if !hasFileFlags && !hasDirFlags {
		if !isInteractiveTerminal() {
			utils.PanicRedAndExit(
				"interactive mode requires a TTY. Use -f, -fp, -dir, or -dirseq flags. See 'hulak help'",
			)
		}
		if !utils.IsHulakProject() {
			ensureHulakProject()
		}
		fp = runInteractiveFlow(&env, flags.EnvSet)
		envMap := InitializeProject(env)
		HandleAPIRequests(envMap, debug, []string{fp}, nil, fp)
		return
	}

	// CLI mode: discover all files, then conditionally load env
	fileList, concurrentDir, sequentialDir := DiscoverFilePaths(
		fileName,
		fp,
		dir,
		dirseq,
		hasDirFlags,
	)

	allPaths := slices.Concat(fileList, concurrentDir, sequentialDir)

	var envMap map[string]any
	// check if any file in the list contains template variable
	// Returns true at the first match (short-circuits), so best-case is O(1)
	// and worst-case (no templates anywhere) is O(n) file reads.
	if slices.ContainsFunc(allPaths, utils.FileHasTemplateVars) {
		if !utils.IsHulakProject() {
			ensureHulakProject()
		}
		envMap = InitializeProject(env)
	}

	HandleAPIRequests(envMap, debug, slices.Concat(fileList, concurrentDir), sequentialDir, fp)
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
			"fatal: not a hulak project \n\nRun 'hulak init' to set up",
		)
	}

	utils.PrintWarning("fatal: not a hulak project")
	fmt.Print("Would you like to create one? [y/N] ")

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
