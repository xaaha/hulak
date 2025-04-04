// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

// AllFlags  All user flags and subcommands
type AllFlags struct {
	Env      string
	FilePath string
	File     string
	Debug    bool
	Dir      string
	Dirseq   string
}

// ParseFlagsSubcmds Exports necessary flags and subcommands for main runner
func ParseFlagsSubcmds() (*AllFlags, error) {
	if len(os.Args) < 2 {
		utils.PrintWarning(
			"Provide a subcommand or a flag. See full docs at https://github.com/xaaha/hulak",
		)
		printHelp()
		os.Exit(1)
	}

	// hulak expects either a subcommand or user flag
	// Check if the first argument is a flag (starts with '-')
	if HasFlag() {
		flag.Parse()
	} else {
		err := HandleSubcommands()
		if err != nil {
			return nil, err
		}
	}

	return &AllFlags{
		Env:      Env(),
		FilePath: FilePath(),
		File:     File(),
		Debug:    Debug(),
		Dir:      Dir(),
		Dirseq:   Dirseq(),
	}, nil
}

// HasFlag checks if user passed in a flag with -
func HasFlag() bool {
	return os.Args[1][0] == '-'
}
