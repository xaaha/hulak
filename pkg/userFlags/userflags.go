package userflags

import (
	"flag"
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

// All user flags and subcommands
type FlagsSubcmds struct {
	Env      string
	FilePath string
	File     string
	Migrate  *flag.FlagSet
}

// Exports necessary flags and subcommands for main runner
func ParseFlagsSubcmds() (*FlagsSubcmds, error) {
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

	return &FlagsSubcmds{
		Env:      Env(),
		FilePath: FilePath(),
		File:     File(),
		Migrate:  migrate,
	}, nil
}

func HasFlag() bool {
	return os.Args[1][0] == '-'
}
