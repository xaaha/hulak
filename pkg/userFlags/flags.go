package userflags

import (
	"flag"
	"time"
)

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
	flagDryRun  bool
	flagShow    bool
	flagTimeout time.Duration

	flagVersion bool
	flagHelp    bool
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	flag.StringVar(&flagEnv, "env", "", "Environment to use (omit to pick interactively)")
	flag.StringVar(
		&flagEnv,
		"environment",
		"",
		"Environment to use (omit to pick interactively)",
	)

	flag.StringVar(&flagFP, "fp", "", "Relative (or absolute) file path of the request file")
	flag.StringVar(&flagFP, "file-path", "", "Relative (or absolute) file path of the request file")

	flag.StringVar(&flagF, "f", "", "File name for making an API request (case-insensitive)")
	flag.StringVar(&flagF, "file", "", "File name for making an API request (case-insensitive)")

	flag.BoolVar(&flagDebug, "debug", false, "Enable debug mode for full request/response details")

	flag.BoolVar(&flagQuiet, "quiet", false, "Suppress the end-of-run summary table")
	flag.BoolVar(&flagQuiet, "q", false, "Suppress the end-of-run summary table")

	// --dry-run and --show are also registered per-subcommand via
	// cliflags.RegisterDryRun / RegisterShow in runcmd. Keep usage strings
	// and defaults in sync across both registration paths.
	flag.BoolVar(&flagDryRun, "dry-run", false, "Print the built request and exit without sending it")

	flag.BoolVar(&flagShow, "show", false, "Reveal sensitive headers (Authorization, Cookie, etc.) in --dry-run output")

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
