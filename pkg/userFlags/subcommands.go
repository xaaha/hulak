// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		operations := loadGraphQLOperations(args[0], *gqlEnv)
		if operations == nil {
			os.Exit(0)
		}
		if err := gqlexplorer.RunExplorer(operations); err != nil {
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
func loadGraphQLOperations(arg string, env string) []gqlexplorer.UnifiedOperation {
	resolved := resolveGQLPath(arg)
	var results []graphql.ProcessResult

	info, err := os.Stat(resolved)
	if err != nil {
		utils.PanicRedAndExit("cannot access %q: %v", arg, err)
	}

	if info.IsDir() {
		results = loadFromDirectory(resolved, env)
	} else {
		results = loadFromFile(resolved, env)
	}

	// load spinner while waiting
	raw, err := tui.RunWithSpinner("Fetching schemas...", func() (any, error) {
		var ops []gqlexplorer.UnifiedOperation
		var errors []string
		for _, result := range results {
			if result.Error != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", result.ApiInfo.Url, result.Error))
				continue
			}
			schema, schemaErr := graphql.FetchAndParseSchema(result.ApiInfo)
			if schemaErr != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", result.ApiInfo.Url, schemaErr))
				continue
			}
			ops = append(ops, gqlexplorer.CollectOperations(schema, result.ApiInfo.Url)...)
		}
		if len(ops) == 0 && len(errors) > 0 {
			return nil, fmt.Errorf("all schema fetches failed:\n  %s", strings.Join(errors, "\n  "))
		}
		for _, e := range errors {
			utils.PrintWarning("schema fetch warning: " + e)
		}
		return ops, nil
	})
	if err != nil {
		utils.PanicRedAndExit("Schema fetch error: %v", err)
	}
	operations, ok := raw.([]gqlexplorer.UnifiedOperation)
	if !ok && raw != nil {
		utils.PanicRedAndExit("unexpected result type from schema fetch")
	}
	return operations
}

func resolveGQLPath(arg string) string {
	if arg == "." {
		cwd, err := os.Getwd()
		if err != nil {
			utils.PanicRedAndExit("error getting current directory: %v", err)
		}
		return cwd
	}
	return filepath.Clean(arg)
}

func loadFromDirectory(dir string, env string) []graphql.ProcessResult {
	urlToFileMap, err := graphql.FindGraphQLFiles(dir)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	filePaths := make([]string, 0, len(urlToFileMap))
	for _, fp := range urlToFileMap {
		filePaths = append(filePaths, fp)
	}
	secretsMap := graphql.GetSecretsForEnv(urlToFileMap, env)
	if secretsMap == nil {
		return nil
	}
	return graphql.ProcessFilesConcurrent(filePaths, secretsMap)
}

func loadFromFile(filePath string, env string) []graphql.ProcessResult {
	rawURL, _, err := graphql.ValidateGraphQLFile(filePath)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	var secretsMap map[string]any
	if strings.Contains(rawURL, "{{") {
		secretsMap = graphql.GetSecretsForEnv(map[string]string{rawURL: filePath}, env)
		if secretsMap == nil {
			return nil
		}
	} else {
		secretsMap = map[string]any{}
	}
	return graphql.ProcessFilesConcurrent([]string{filePath}, secretsMap)
}
