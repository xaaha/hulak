// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"
	"fmt"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
)

// requireVaultProject returns an error if the current directory is not inside
// a hulak vault project. Checks that .hulak/ directory actually exists on disk
// (not just that FindProjectRoot found an env/ marker). The store.age file
// itself may not exist yet (fresh init, before first `set`).
func requireVaultProject() error {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return fmt.Errorf(
			"no vault project found\n\n" +
				"Run 'hulak init' to create one, or change to a hulak project directory",
		)
	}
	if !utils.DirExists(markerPath) {
		return fmt.Errorf(
			"this is a classic (env/) project, not a vault project\n\n" +
				"Run 'hulak secrets migrate' to upgrade to the encrypted vault",
		)
	}
	return nil
}

var (
	flagEnv string
	flagFP  string
	flagF   string
	flagDir string
	// dirseq runs directories in alphabetical order.
	// Note that in nested directories, the execution order
	// may not follow the file system appearance.
	// Go automatically sorts by depth, processing shallower directories first.
	// For example, consider the following run structure
	//
	//   dir_0/zeez.md
	//   dir_0/dir_0/dir_0/aa.md
	//   dir_0/dir_0/dir_0/hulak.md
	//   dir_0/dir_0/dir_1/is.md
	//
	// In the above case, the files in the shallowest directories will be processed before deeper ones.
	flagDirseq  string
	flagDebug   bool
	flagQuiet   bool
	flagTimeout time.Duration

	flagVersion bool
	flagHelp    bool
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	flag.StringVar(&flagEnv, "env", utils.DefaultEnvVal, "Environment file to use during the call")
	flag.StringVar(
		&flagEnv,
		"environment",
		utils.DefaultEnvVal,
		"Environment file to use during the call",
	)

	flag.StringVar(&flagFP, "fp", "", "Relative (or absolute) file path of the request file")
	flag.StringVar(&flagFP, "file-path", "", "Relative (or absolute) file path of the request file")

	flag.StringVar(&flagF, "f", "", "File name for making an API request (case-insensitive)")
	flag.StringVar(&flagF, "file", "", "File name for making an API request (case-insensitive)")

	flag.BoolVar(&flagDebug, "debug", false, "Enable debug mode for full request/response details")

	flag.BoolVar(&flagQuiet, "quiet", false, "Suppress the end-of-run summary table")
	flag.BoolVar(&flagQuiet, "q", false, "Suppress the end-of-run summary table")

	flag.StringVar(&flagDir, "dir", "", "Directory path to run concurrently")

	flag.StringVar(&flagDirseq, "dirseq", "", "Directory path to run in alphabetical order")

	flag.DurationVar(
		&flagTimeout,
		"timeout",
		0,
		"Per-request timeout, e.g. 5m or 90s (default 60s)",
	)

	flag.BoolVar(&flagVersion, "v", false, "Print the version")
	flag.BoolVar(&flagVersion, "version", false, "Print the version")

	flag.BoolVar(&flagHelp, "help", false, "Print help")
	flag.BoolVar(&flagHelp, "h", false, "Print help")
}
