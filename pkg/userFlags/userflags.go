package userflags

import (
	"flag"
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

// list of user flags and subcommands
type FlagsSubcmds struct {
	Env      string
	FilePath string
	File     string
	Migrate  *flag.FlagSet
}

func ParseFlagsSubcmds() (*FlagsSubcmds, error) {
	if len(os.Args) < 2 {
		utils.PrintWarning(
			// TODO: Use man
			"Provide a subcommand or a flag. See docs at https://github.com/xaaha/hulak",
		)
		os.Exit(1)
	}

	// handle subcommands
	err := Subcommands()
	if err != nil {
		return nil, err
	}

	// parse all flags
	flag.Parse()

	return &FlagsSubcmds{
		Env:      Env(),
		FilePath: FilePath(),
		File:     File(),
		Migrate:  migrate,
	}, nil
}
