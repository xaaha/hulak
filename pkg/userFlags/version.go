// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"fmt"
)

// Until I have more time to build a solid makefile, this should suffice
const version = "v0.1.0-beta.5.3"

func getVersion() {
	fmt.Printf("%s\n", version)
}
