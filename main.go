// Package main initializes the project and runs the query
package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/runner"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/tui/envselect"
	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

// requestSchemaJSON is the canonical hulak request schema, embedded so the
// MCP write_request tool can validate content without a filesystem lookup.
//
//go:embed assets/schema.json
var requestSchemaJSON []byte

func main() {
	userflags.SetRequestSchema(requestSchemaJSON)

	// Subcommands and root file/dir flags handle execution directly
	// inside ParseFlagsSubcmds. It only returns here for interactive mode.
	flags, err := userflags.ParseFlagsSubcmds()
	if err != nil {
		// Runner failures already printed per-file detail via printOutcome
		// and a summary line — exit silently with non-zero so we don't
		// print the failure twice.
		if runner.IsRunFailure(err) {
			os.Exit(1)
		}
		utils.PanicRedAndExit("%v", err)
	}

	if !isInteractiveTerminal() {
		utils.PanicRedAndExit(
			"interactive mode requires a TTY. Use 'hulak run <path>'. See 'hulak help'",
		)
	}

	filePath := runInteractiveFlow(&flags.Env, flags.EnvSet)

	var envMap map[string]any
	if utils.FileHasTemplateVars(filePath) {
		var initErr error
		envMap, initErr = runner.InitializeProject(flags.Env, false)
		if initErr != nil {
			utils.PanicRedAndExit("%v", initErr)
		}
	}

	if err := runner.ExecuteSingleFile(envMap, flags.Debug, filePath, flags.Timeout); err != nil {
		// Same suppression as above: runner.IsRunFailure means printOutcome
		// already explained the failure on stderr; just flip the exit code.
		// The non-IsRunFailure branch is defensive — handleAPIRequests is
		// the only thing ExecuteSingleFile can fail on today, and it always
		// returns *runFailureError. Kept so a future caller change doesn't
		// silently swallow a different error type.
		if !runner.IsRunFailure(err) {
			utils.PrintErrorStderr(err.Error())
		}
		os.Exit(1)
	}
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

	selectedEnv, cancelled, err := envselect.RunEnvSelector()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	if cancelled {
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

	utils.PrintErrorStderr("environment resolution requires a Hulak project")
	confirmed, err := utils.ConfirmAction("Initialize one here? [y/N] ")
	if err != nil {
		utils.PanicRedAndExit("failed to read input: %v", err)
	}
	if confirmed {
		fmt.Fprintln(os.Stderr)
		if err := envparser.CreateDefaultEnvs(nil); err != nil {
			utils.PanicRedAndExit("%v", err)
		}
		fmt.Fprintln(os.Stderr)
		return
	}

	fmt.Fprintln(os.Stderr, "\nTo set up manually, run: hulak init")
	os.Exit(0)
}
