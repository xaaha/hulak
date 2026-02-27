// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"fmt"
)

var version = "dev"

func getVersion() {
	fmt.Printf("%s\n", version)
}
