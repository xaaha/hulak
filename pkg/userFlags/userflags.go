package userflags

import (
	"fmt"
	"os"
)

// write logic to check if we have enough arguments with
// and use this function to return the flag struct that main can use
// if the os.Args's second argument is migrate then run subcommands

func UserFalgs() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'subcommands' or 'flag'")
		os.Exit(1)
	}
}
