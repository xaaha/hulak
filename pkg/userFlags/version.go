// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"fmt"
)

// Until I have more time to build a solid makefile, this should suffice
const version = "v0.1.2"

func getVersion() {
	fmt.Printf("%s\n", version)
}
