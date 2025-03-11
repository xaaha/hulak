package userflags

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

var migrate *flag.FlagSet

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	migrate = flag.NewFlagSet("migrate", flag.ExitOnError)
}

// Takes in filePath to a json file that needs migration.
// Run as: hulak migrate "./filePath.json"
func Migrate() {
	if len(os.Args) < 2 {
		utils.PrintWarning(
			"Provide a subcommand or a flag. See docs at https://github.com/xaaha/hulak",
		)
		os.Exit(1)
	}

	// TODO-1: can't exit because, we expect a flag or a subcommand

	switch os.Args[1] {
	case "migrate":
		_ = migrate.Parse(os.Args[2:])
		fmt.Println("subcommand 'foo'")
		fmt.Println("  tail:", migrate.Args())
	default:
		fmt.Println("expected 'foo' or 'bar' subcommands")
		os.Exit(1)
	}
}
