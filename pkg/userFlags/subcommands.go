// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/utils"
)

//go:embed apiOptions.hk.yaml
var embeddedFiles embed.FS

// User subcommands
const (
	Version = "version"
	Migrate = "migrate"
	Init    = "init"
	Help    = "help"
	GraphQL = "gql"
)

var (
	migrate    *flag.FlagSet
	initialize *flag.FlagSet
	gql        *flag.FlagSet

	// Flag to indicate if environments should be created
	createEnvs *bool
	// gqlEnv is the environment flag for the gql subcommand
	gqlEnv *string
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	migrate = flag.NewFlagSet(Migrate, flag.ExitOnError)
	gql = flag.NewFlagSet(GraphQL, flag.ExitOnError)
	gqlEnv = gql.String("env", "", "Environment file to use (skips interactive selector)")

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
	case Version:
		getVersion()
		os.Exit(0)
	case Migrate:
		err := migrate.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("\n invalid subcommand %v", err)
		}
		filePaths := migrate.Args()
		err = migration.CompleteMigration(filePaths)
		if err != nil {
			return fmt.Errorf("file path error %v", err)
		}
		os.Exit(0)

	case Init:
		if err := handleInit(); err != nil {
			return err
		}
		os.Exit(0)

	case Help:
		utils.PrintHelp()
		os.Exit(0)

	case GraphQL:
		err := gql.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("\n invalid subcommand after gql %v", err)
		}
		paths := gql.Args()
		graphql.Introspect(paths, *gqlEnv)
		os.Exit(0)

	default:
		utils.PrintRed("Enter a valid subcommand")
		utils.PrintHelpSubCommands()
		os.Exit(1)
	}
	return nil
}
