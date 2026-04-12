// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"

	"github.com/xaaha/hulak/pkg/utils"
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
	flagDirseq string
	flagDebug  bool

	flagVersion bool
	flagHelp    bool
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	flag.StringVar(&flagEnv, "env", utils.DefaultEnvVal, "Environment file to use during the call")
	flag.StringVar(&flagEnv, "environment", utils.DefaultEnvVal, "Environment file to use during the call")

	flag.StringVar(&flagFP, "fp", "", "Relative (or absolute) file path of the request file")
	flag.StringVar(&flagFP, "file-path", "", "Relative (or absolute) file path of the request file")

	flag.StringVar(&flagF, "f", "", "File name for making an API request (case-insensitive)")
	flag.StringVar(&flagF, "file", "", "File name for making an API request (case-insensitive)")

	flag.BoolVar(&flagDebug, "debug", false, "Enable debug mode for full request/response details")

	flag.StringVar(&flagDir, "dir", "", "Directory path to run concurrently")

	flag.StringVar(&flagDirseq, "dirseq", "", "Directory path to run in alphabetical order")

	flag.BoolVar(&flagVersion, "v", false, "Print the version")
	flag.BoolVar(&flagVersion, "version", false, "Print the version")

	flag.BoolVar(&flagHelp, "help", false, "Print help")
	flag.BoolVar(&flagHelp, "h", false, "Print help")
}
