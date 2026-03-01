// Package main initializes the project and runs the query
package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
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

	// TUI mode, currently only supports -fp (single file run)
	if !hasFileFlags && !hasDirFlags {
		if !isInteractiveTerminal() {
			utils.PanicRedAndExit(
				"interactive mode requires a TTY. Use -f, -fp, -dir, or -dirseq flags. See 'hulak help'",
			)
		}
		fp = runInteractiveFlow(&env, flags.EnvSet)
		var envMap map[string]any
		if utils.FileHasTemplateVars(fp) {
			envMap = InitializeProject(env, false)
		}
		HandleAPIRequests(envMap, debug, []string{fp}, nil, fp)
		return
	}

	// CLI mode
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
		envMap = InitializeProject(env, true)
	}

	HandleAPIRequests(envMap, debug, slices.Concat(fileList, concurrentDir), sequentialDir, fp)
}

func runInteractiveFlow(env *string, envSet bool) string {
	itemsResult, err := tui.RunWithSpinnerAfter("Searching files...", func() (any, error) {
		return tui.FileItems()
	})
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	fileItems, ok := itemsResult.([]string)
	if !ok {
		utils.PanicRedAndExit("internal error: file discovery returned unexpected type")
	}

	selectedFile, err := tui.RunSelector(fileItems, "Select Request File: ", tui.NoFilesError())
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	if selectedFile == "" {
		os.Exit(0)
	}

	if !utils.FileHasTemplateVars(selectedFile) {
		return selectedFile
	}

	if !utils.IsHulakProject() {
		ensureHulakProject()
	}

	if envSet {
		return selectedFile
	}

	selectedEnv, err := envselect.RunEnvSelector()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	if selectedEnv == "" {
		os.Exit(0)
	}

	*env = selectedEnv

	return selectedFile
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

	utils.PrintWarning("error: environment resolution requires a Hulak project")
	fmt.Print("Initialize one here? [y/N] ")

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
