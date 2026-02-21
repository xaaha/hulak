// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"
	"os"
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
}

// ParseFlagsSubcmds Exports necessary flags and subcommands for main runner
func ParseFlagsSubcmds() (*AllFlags, error) {
	if len(os.Args) >= 2 {
		if HasFlag() {
			flag.Parse()
		} else {
			err := HandleSubcommands()
			if err != nil {
				return nil, err
			}
		}
	}

	envSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "env" {
			envSet = true
		}
	})

	return &AllFlags{
		Env:      Env(),
		EnvSet:   envSet,
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
