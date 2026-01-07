// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/features/curl"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/utils"
)

//go:embed apiOptions.yaml
var embeddedFiles embed.FS

// User subcommands
const (
	Version = "version"
	Migrate = "migrate"
	Init    = "init"
	Help    = "help"
	GraphQL = "gql"
	Import  = "import"
)

var (
	migrate    *flag.FlagSet
	initialize *flag.FlagSet
	gql        *flag.FlagSet
	importCmd  *flag.FlagSet

	// Flag to indicate if environments should be created
	createEnvs *bool
	// Flag for import command output path
	outputPath *string
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	migrate = flag.NewFlagSet(Migrate, flag.ExitOnError)
	gql = flag.NewFlagSet(GraphQL, flag.ExitOnError)

	initialize = flag.NewFlagSet(Init, flag.ExitOnError)
	createEnvs = initialize.Bool(
		"env",
		false,
		"Create environment files based on following arguments",
	)

	importCmd = flag.NewFlagSet(Import, flag.ExitOnError)
	outputPath = importCmd.String(
		"o",
		"",
		"Output path for the generated .hk.yaml file",
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
		printHelp()
		os.Exit(0)

	case GraphQL:
		err := gql.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("\n invalid subcommand after gql %v", err)
		}
		paths := gql.Args()
		graphql.Introspect(paths)
		os.Exit(0)

	case Import:
		err := importCmd.Parse(os.Args[2:])
		if err != nil {
			return fmt.Errorf("\n invalid subcommand after import %v", err)
		}
		args := importCmd.Args()
		err = curl.ImportCurl(args, *outputPath)
		if err != nil {
			return err
		}
		os.Exit(0)

	default:
		utils.PrintRed("Enter a valid subcommand")
		printHelpSubCommands()
		os.Exit(1)
	}
	return nil
}
