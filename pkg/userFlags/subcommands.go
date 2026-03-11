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
	"github.com/xaaha/hulak/pkg/yamlparser"
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
		data, refreshFn, warnings := loadGraphQLOperations(args[0], *gqlEnv)
		if data.Operations == nil {
			os.Exit(0)
		}
		if err := gqlexplorer.RunExplorerWithRefresh(data, refreshFn, warnings); err != nil {
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
	gqlexplorer.ExplorerData,
	gqlexplorer.RefreshFunc,
	[]string,
) {
	prepared, err := graphql.PrepareSchemaLoad(arg, env)
	if err != nil {
		utils.PanicRedAndExit("Schema preparation error: %v", err)
	}
	if prepared.Cancelled {
		return gqlexplorer.ExplorerData{}, nil, nil
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
		return gqlexplorer.ExplorerData{}, nil, nil
	}
	refreshFn := func() (gqlexplorer.RefreshPayload, error) {
		freshPrepared, err := graphql.PrepareSchemaLoad(arg, prepared.Env)
		if err != nil {
			return gqlexplorer.RefreshPayload{}, err
		}
		if freshPrepared.Cancelled {
			return gqlexplorer.RefreshPayload{}, nil
		}
		freshLoadResult, err := graphql.FetchPreparedSchemas(freshPrepared)
		if err != nil {
			return gqlexplorer.RefreshPayload{}, err
		}
		return gqlexplorer.RefreshPayload{
			Data:     explorerDataFromLoadResult(freshLoadResult, freshPrepared.Results),
			Warnings: freshLoadResult.Warnings,
		}, nil
	}

	return explorerDataFromLoadResult(loadResult, prepared.Results), refreshFn, loadResult.Warnings
}

func explorerDataFromLoadResult(loadResult graphql.LoadResult, processResults []graphql.ProcessResult) gqlexplorer.ExplorerData {
	data := gqlexplorer.ExplorerData{
		InputTypes:     make(map[string]graphql.InputType),
		EnumTypes:      make(map[string]graphql.EnumType),
		ObjectTypes:    make(map[string]graphql.ObjectType),
		UnionTypes:     make(map[string]graphql.UnionType),
		InterfaceTypes: make(map[string]graphql.InterfaceType),
		APIInfos:       make(map[string]yamlparser.APIInfo),
	}

	// Build APIInfos map from ProcessResults
	for _, result := range processResults {
		if result.Error == nil {
			data.APIInfos[result.APIInfo.URL] = result.APIInfo
		}
	}

	for i := range loadResult.Endpoints {
		endpoint := &loadResult.Endpoints[i]
		data.Operations = append(data.Operations, gqlexplorer.CollectOperations(&endpoint.Schema, endpoint.URL)...)
		for k, v := range endpoint.Schema.InputTypes {
			data.InputTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.EnumTypes {
			data.EnumTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.ObjectTypes {
			data.ObjectTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.UnionTypes {
			data.UnionTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.InterfaceTypes {
			data.InterfaceTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
	}

	return data
}
