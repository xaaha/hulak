// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// errRunSubcommand is a sentinel returned by the run handler to signal
// that ParseFlagsSubcmds should return AllFlags instead of exiting.
var errRunSubcommand = errors.New("run subcommand")

// runResult holds the AllFlags populated by the run handler.
var runResult *AllFlags

// AllFlags  All user flags and subcommands
type AllFlags struct {
	Env      string
	EnvSet   bool
	FilePath string
	File     string
	Debug    bool
	Dir      string
	Dirseq   string
}

// ParseFlagsSubcmds Exports necessary flags and subcommands for main runner
func ParseFlagsSubcmds() (*AllFlags, error) {
	subCmds := subCommands()

	// Override Go's default flag error handling so we show hulak-style
	// errors instead of the raw flag.Usage dump.
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Usage = func() {}     // suppress default usage on error
	flag.CommandLine.SetOutput(io.Discard) // suppress "flag provided but not defined" to stderr

	if len(os.Args) >= 2 {
		first := os.Args[1]
		switch {
		case subCmds.findSub(first) != nil:
			if err := subCmds.Execute(os.Args[1:]); err != nil {
				if errors.Is(err, errRunSubcommand) {
					result := runResult
					runResult = nil
					return result, nil
				}
				return nil, err
			}
			os.Exit(0)
		case strings.HasPrefix(first, "-"):
			if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
				utils.PrintRed(fmt.Sprintf("%s\nSee 'hulak help' for available flags", err))
				os.Exit(1)
			}
			switch {
			case flagVersion:
				getVersion()
				os.Exit(0)
			case flagHelp:
				subCmds.printHelp()
				os.Exit(0)
			}
		default:
			utils.PrintRed(fmt.Sprintf("unknown command %q. See 'hulak help'", first))
			os.Exit(1)
		}
	}

	envSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "env" || f.Name == "environment" {
			envSet = true
		}
	})

	return &AllFlags{
		Env:      flagEnv,
		EnvSet:   envSet,
		FilePath: flagFP,
		File:     flagF,
		Debug:    flagDebug,
		Dir:      flagDir,
		Dirseq:   flagDirseq,
	}, nil
}
