package userflags

import (
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

// write logic to check if we have enough arguments with
// and use this function to return the flag struct that main can use
// if the os.Args's second argument is migrate then run subcommands

func UserFalgs() {
	if len(os.Args) < 2 {
		utils.PrintWarning(
			"Provide a subcommand or a flag. See docs at https://github.com/xaaha/hulak",
		)
		os.Exit(1)
	}
}
