package userflags

import (
	"fmt"
)

var version = "dev"

func getVersion() {
	fmt.Printf("%s\n", version)
}
