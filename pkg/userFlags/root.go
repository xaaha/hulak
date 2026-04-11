package userflags

import (
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/utils"
)

// NewRoot builds the full command-tree for hulak
func NewRoot() *Command {
	root := &Command{
		Name:  "hulak",
		Long:  "hulak — a file-based API client for the terminal",
		Flags: flag.CommandLine,
		Examples: []*utils.CommandHelp{
			{Command: "hulak", Description: "Interactive mode: pick a file, then an environment"},
			{
				Command:     "hulak -env staging -fp getUser.yaml",
				Description: "Run a specific file with a specific environment",
			},
			{
				Command:     "hulak -env global -f getUser",
				Description: "Find and run all files named 'getUser'",
			},
			{
				Command:     "hulak -fp getUser.yaml -debug",
				Description: "Run in debug mode (full request/response details)",
			},
			{
				Command:     "hulak -env prod -dir path/to/dir",
				Description: "Run all files in a directory concurrently",
			},
			{
				Command:     "hulak -env prod -dirseq path/to/dir",
				Description: "Run all files in a directory sequentially",
			},
		},
	}

	root.SubCommands = []*Command{
		newVersionCmd(),
		newInitCmd(),
		newMigrateCmd(),
		newDoctorCmd(),
		newGQLCmd(),
		newHelpCmd(root),
	}

	return root
}

func newVersionCmd() *Command {
	return &Command{
		Name:  "version",
		Short: "Print hulak version",
		Long:  "Print the current hulak version.",
		Run: func(_ []string) error {
			getVersion()
			return nil
		},
	}
}

func newInitCmd() *Command {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	createEnvFlag := fs.Bool(
		"env",
		false,
		"Create specific environment files instead of the default setup",
	)

	return &Command{
		Name:  "init",
		Short: "Initialize a hulak project",
		Long: fmt.Sprintf(
			"Set up a new hulak project in the current directory.\n\n"+
				"Creates an env/ directory with global.env, a .gitignore entry,\n"+
				"and an example '%s' file. Use -env to create specific environments.",
			utils.APIOptions,
		),
		Examples: []*utils.CommandHelp{
			{Command: "hulak init", Description: "Default setup (global.env + example file)"},
			{
				Command:     "hulak init -env staging prod",
				Description: "Create staging.env and prod.env",
			},
		},
		Flags: fs,
		Args: []ArgDef{
			{Name: "envNames", Desc: "Environment names to create (used with -env)"},
		},
		Run: func(args []string) error {
			if *createEnvFlag {
				if len(args) == 0 {
					utils.PrintWarning("No environment names provided after -env flag")
					return nil
				}
				for _, env := range args {
					if err := envparser.CreateDefaultEnvs(&env); err != nil {
						utils.PrintRed(err.Error())
					}
				}
				return nil
			}
			return InitDefaultProject()
		},
	}
}

func newMigrateCmd() *Command {
	return &Command{
		Name:  "migrate",
		Short: "Migrate Postman collections to hulak format",
		Long:  "Convert Postman v2.1 environment and collection JSON exports into hulak .hk.yaml and .env files.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak migrate collection.json", Description: "Migrate a Postman collection"},
			{
				Command:     "hulak migrate env.json collection.json",
				Description: "Migrate environment and collection together",
			},
		},
		Args: []ArgDef{
			{Name: "files", Required: true, Desc: "Postman JSON export files"},
		},
		Run: migration.CompleteMigration,
	}
}

func newDoctorCmd() *Command {
	return &Command{
		Name:  "doctor",
		Short: "Check project health",
		Long:  "Inspect your hulak project for common issues: missing .gitignore entries,\nloose file permissions on env files, and secrets leaked into git history.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak doctor", Description: "Run all health checks"},
		},
		Run: func(_ []string) error {
			runDoctor()
			return nil
		},
	}
}

func newGQLCmd() *Command {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	envFlag := fs.String("env", "", "Environment to use (skips interactive selector)")

	gqlCmd := &Command{
		Name:    "gql",
		Aliases: []string{"graphql", "GraphQL"},
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
		Args: []ArgDef{
			{
				Name:     "path",
				Required: true,
				Desc:     "File or directory containing GraphQL definitions",
			},
		},
	}

	gqlCmd.Run = func(args []string) error {
		if len(args) == 0 {
			gqlCmd.printHelp()
			return nil
		}
		data, refreshFn, warnings := loadGraphQLOperations(args[0], *envFlag)
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

func newHelpCmd(root *Command) *Command {
	return &Command{
		Name:  "help",
		Short: "Show help for hulak",
		Run: func(_ []string) error {
			root.printHelp()
			return nil
		},
	}
}
