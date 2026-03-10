// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
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
	if len(os.Args) < 2 {
		utils.PrintHelp()
		os.Exit(0)
	}

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
		args := gql.Args()
		if len(args) == 0 {
			utils.PrintGQLUsage()
			os.Exit(0)
		}
		operations, inputTypes, enumTypes, objectTypes, unionTypes, interfaceTypes := loadGraphQLOperations(args[0], *gqlEnv)
		if operations == nil {
			os.Exit(0)
		}
		if err := gqlexplorer.RunExplorer(operations, inputTypes, enumTypes, objectTypes, unionTypes, interfaceTypes); err != nil {
			utils.PanicRedAndExit("TUI error: %v", err)
		}
		os.Exit(0)

	default:
		utils.PrintRed("Enter a valid subcommand")
		utils.PrintHelpSubCommands()
		os.Exit(1)
	}
	return nil
}

// handles single file mode and directory mode along with unifying the operation
func loadGraphQLOperations(arg string, env string) (
	[]gqlexplorer.UnifiedOperation,
	map[string]graphql.InputType,
	map[string]graphql.EnumType,
	map[string]graphql.ObjectType,
	map[string]graphql.UnionType,
	map[string]graphql.InterfaceType,
) {
	prepared, err := graphql.PrepareSchemaLoad(arg, env)
	if err != nil {
		utils.PanicRedAndExit("Schema preparation error: %v", err)
	}
	if prepared.Cancelled {
		return nil, nil, nil, nil, nil, nil
	}

	// load spinner while waiting
	raw, err := tui.RunWithSpinnerAfter("Fetching schemas...", func() (any, error) {
		return graphql.FetchPreparedSchemas(prepared)
	})
	if err != nil {
		utils.PanicRedAndExit("Schema fetch error: %v", err)
	}
	loadResult, ok := raw.(graphql.LoadResult)
	if !ok && raw != nil {
		utils.PanicRedAndExit("unexpected result type from schema fetch")
	}

	if loadResult.Cancelled {
		return nil, nil, nil, nil, nil, nil
	}
	for _, warning := range loadResult.Warnings {
		utils.PrintWarning("schema fetch warning: " + warning)
	}

	inputTypes := make(map[string]graphql.InputType)
	enumTypes := make(map[string]graphql.EnumType)
	objectTypes := make(map[string]graphql.ObjectType)
	unionTypes := make(map[string]graphql.UnionType)
	interfaceTypes := make(map[string]graphql.InterfaceType)
	var ops []gqlexplorer.UnifiedOperation

	for i := range loadResult.Endpoints {
		endpoint := &loadResult.Endpoints[i]
		ops = append(ops, gqlexplorer.CollectOperations(&endpoint.Schema, endpoint.URL)...)
		for k, v := range endpoint.Schema.InputTypes {
			inputTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.EnumTypes {
			enumTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.ObjectTypes {
			objectTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.UnionTypes {
			unionTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.InterfaceTypes {
			interfaceTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
	}

	return ops, inputTypes, enumTypes, objectTypes, unionTypes, interfaceTypes
}
