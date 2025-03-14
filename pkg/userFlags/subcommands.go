package userflags

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/migration"
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
	switch os.Args[1] {
	case "migrate":
		_ = migrate.Parse(os.Args[2:])
		filePaths := migrate.Args()
		err := migration.CompleteMigration(filePaths)
		if err != nil {
			utils.PrintRed("error on migration: " + err.Error())
			return
		}
	default:
		fmt.Println("expected 'foo' or 'bar' subcommands")
		os.Exit(1)
	}
}
