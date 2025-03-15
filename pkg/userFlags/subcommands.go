package userflags

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/utils"
)

// User subcommands
const (
	Migrate = "migrate"
	// future subcommands
	Init = "init"
	Help = "help"
)

var migrate *flag.FlagSet

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	migrate = flag.NewFlagSet("migrate", flag.ExitOnError)
}

// Loops through all the subcommands
func Subcommands() error {
	switch os.Args[1] {
	case Migrate:
		err := migrate.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("subcommands.go %v", err)
		}
		filePaths := migrate.Args()
		err = migration.CompleteMigration(filePaths)
		if err != nil {
			return fmt.Errorf("subcommands.go %v", err)
		}
		// add init, help  and other cases as necessary
	default:
		utils.PrintWarning("expected a subcommand")
		os.Exit(1)
	}
	return nil
}
