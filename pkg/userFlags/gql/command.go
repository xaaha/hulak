package gql

import (
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/cliflags"
	"github.com/xaaha/hulak/pkg/utils"
)

// New builds the `hulak gql` command. Registered at the top-level
// dispatch tree alongside other leaf commands.
func New() *cli.Command {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	envFlagVal := cliflags.RegisterEnv(fs, "", "Environment to use (skips interactive selector)")

	gqlCmd := &cli.Command{
		Name:    "gql",
		Aliases: []string{"graphql"},
		Short:   "Open the GraphQL explorer",
		Long:    "Launch an interactive TUI to browse and run GraphQL operations\ndefined in your .yml/.yaml files.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak gql .",
				Description: "Explore all GraphQL files in the current directory",
			},
			{
				Command:     "hulak gql path/to/schema.yml",
				Description: "Explore a single GraphQL source file",
			},
			{
				Command:     "hulak gql -env staging .",
				Description: "Use the staging environment (skip env picker)",
			},
		},
		Flags: fs,
		Args: []cli.ArgDef{
			{
				Name:     "path",
				Required: true,
				Desc:     "File or directory containing GraphQL definitions",
				Kind:     "yaml",
			},
		},
	}

	gqlCmd.Run = func(args []string) error {
		if len(args) == 0 {
			gqlCmd.PrintHelp()
			return nil
		}
		data, refreshFn, warnings, err := loadGraphQLOperations(args[0], *envFlagVal)
		if err != nil {
			return err
		}
		if data.Operations == nil {
			return nil
		}
		if err := gqlexplorer.RunExplorerWithRefresh(&data, refreshFn, warnings); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	}

	return gqlCmd
}
