// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
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

var (
	migrate    *flag.FlagSet
	initialize *flag.FlagSet

	// Flag to indicate if environments should be created
	createEnvs *bool
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	migrate = flag.NewFlagSet(Migrate, flag.ExitOnError)

	initialize = flag.NewFlagSet(Init, flag.ExitOnError)
	createEnvs = initialize.Bool(
		"env",
		false,
		"Create environment files based on following arguments",
	)
}

// HandleSubcommands loops through all the subcommands
func HandleSubcommands() error {
	switch os.Args[1] {
	case Migrate:
		err := migrate.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("\n invalid subcommand %v", err)
		}
		filePaths := migrate.Args()
		err = migration.CompleteMigration(filePaths)
		if err != nil {
			return fmt.Errorf("\n invalid subcommand %v", err)
		}
		os.Exit(0)

	case Init:
		err := initialize.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("\n invalid subcommand %v", err)
		}
		// Check if -env flag is present
		if *createEnvs {
			envs := initialize.Args()
			if len(envs) > 0 {
				for _, env := range envs {
					if err := envparser.CreateDefaultEnvs(&env); err != nil {
						utils.PrintRed(err.Error())
					}
				}
			} else {
				utils.PrintWarning("No environment names provided after -env flag")
			}
		} else {
			if err := envparser.CreateDefaultEnvs(nil); err != nil {
				utils.PrintRed(err.Error())
			}
			utils.PrintGreen("Done " + utils.CheckMark)
		}
		os.Exit(0)

	case Help:
		printHelp()
		os.Exit(0)

	default:
		utils.PrintRed("Enter a valid subcommand")
		printHelpSubCommands()
		os.Exit(1)
	}
	return nil
}
