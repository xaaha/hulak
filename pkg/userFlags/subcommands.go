package userflags

import (
	"flag"
	"fmt"
	"os"
)

var migrate *flag.FlagSet

// pmEnvFp *string

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	// Migrate to hulak currently only supports
	// Postman's environment file or collection
	migrate = flag.NewFlagSet("migrate", flag.ExitOnError)
	// pmEnvFp = migrate.String("fp", "", "file path of the json env file from")
}

// hulak migrate "./globals.json"
func MigratePostManEnv() {
	switch os.Args[1] {
	case "migrate":
		tailArgs := migrate.Args()
		fmt.Println("Tails", tailArgs[0])
	}
}
