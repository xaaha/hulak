// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/runner"
)

// AllFlags  All user flags and subcommands
type AllFlags struct {
	Env      string
	EnvSet   bool
	FilePath string
	File     string
	Debug    bool
	Dir      string
	Dirseq   string
	Timeout  time.Duration
}

// ParseFlagsSubcmds dispatches subcommands and root flags.
// Subcommands and root file/dir flags handle execution directly.
// Returns only for interactive mode (no file/dir targets were given).
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
				return nil, err
			}
			os.Exit(0)
		case strings.HasPrefix(first, "-"):
			if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
				return nil, fmt.Errorf("%s\nSee 'hulak help' for available flags", err)
			}
			switch {
			case flagVersion:
				getVersion()
				os.Exit(0)
			case flagHelp:
				subCmds.printHelp()
				os.Exit(0)
			}

			// Root flags with file/dir targets: execute directly
			if flagFP != "" || flagF != "" || flagDir != "" || flagDirseq != "" {
				envSet := false
				flag.Visit(func(f *flag.Flag) {
					if f.Name == "env" || f.Name == "environment" {
						envSet = true
					}
				})
				err := runner.Execute(&runner.Flags{
					Env:      flagEnv,
					EnvSet:   envSet,
					FilePath: flagFP,
					File:     flagF,
					Debug:    flagDebug,
					Dir:      flagDir,
					Dirseq:   flagDirseq,
					Timeout:  flagTimeout,
				})
				if err != nil {
					return nil, err
				}
				os.Exit(0)
			}
		default:
			return nil, fmt.Errorf("unknown command %q. See 'hulak help'", first)
		}
	}

	// Interactive mode: no subcommand, no file/dir flags.
	// Return just enough state for the TUI flow.
	envSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "env" || f.Name == "environment" {
			envSet = true
		}
	})

	return &AllFlags{
		Env:     flagEnv,
		EnvSet:  envSet,
		Debug:   flagDebug,
		Timeout: flagTimeout,
	}, nil
}
