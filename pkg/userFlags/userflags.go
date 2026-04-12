// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
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
	subCmds := subCommands()

	if len(os.Args) >= 2 {
		first := os.Args[1]
		switch {
		case subCmds.findSub(first) != nil:
			if err := subCmds.Execute(os.Args[1:]); err != nil {
				return nil, err
			}
			os.Exit(0)
		case strings.HasPrefix(first, "-"):
			flag.Parse()
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
		if f.Name == "env" {
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
