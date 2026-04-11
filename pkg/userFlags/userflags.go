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
	subCmds := SubCommands()

	if len(os.Args) >= 2 {
		if !hasFlag() {
			if err := subCmds.Execute(os.Args[1:]); err != nil {
				return nil, err
			}
			os.Exit(0)
		}

		flag.Parse()
		switch {
		case *vFlag || *versionFlag:
			getVersion()
			os.Exit(0)
		case *help || *h:
			subCmds.printHelp()
			os.Exit(0)
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

// hasFlag checks if user passed in a flag with -
func hasFlag() bool {
	return os.Args[1][0] == '-'
}
