package userflags

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/utils"
)

// NewRoot builds the full command tree for hulak
func NewRoot() *Command {
	root := &Command{
		Name: "hulak",
		Long: "hulak — a file-based API client for the terminal",
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
		Long:  "Print the current hulak version",
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
		"Create environment files based on following arguments",
	)

	return &Command{
		Name:  "init",
		Short: "Initialize a hulak project",
		Long: fmt.Sprintf(
			"Initialize a new hulak project.\n\n"+
				"Without flags, creates a default environment and '%s' file.\n"+
				"With -env, creates specific environment files.",
			utils.APIOptions,
		),
		Flags: fs,
		Args: []ArgDef{
			{Name: "envNames", Desc: "Environment names to create (with -env flag)"},
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
		Short: "Migrate postman env and collections",
		Long:  "Migrate Postman v2.1 environment and collection files to hulak format.",
		Args: []ArgDef{
			{Name: "files", Required: true, Desc: "Postman export files to migrate"},
		},
		Run: migration.CompleteMigration,
	}
}

func newDoctorCmd() *Command {
	return &Command{
		Name:  "doctor",
		Short: "Check project health",
		Long:  "Check project health: gitignore, permissions, git history.",
		Run: func(_ []string) error {
			runDoctor()
			return nil
		},
	}
}

func newGQLCmd() *Command {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	envFlag := fs.String("env", "", "Environment file to use (skips interactive selector)")

	return &Command{
		Name:    "gql",
		Aliases: []string{"graphql", "GraphQL"},
		Short:   "Open the GraphQL explorer",
		Long: "Open the GraphQL explorer for files and directories.\n\n" +
			"Examples:\n" +
			"  hulak gql .                          All GraphQL files in current directory\n" +
			"  hulak gql path/to/file.yml           One GraphQL source file\n" +
			"  hulak gql -env staging path/to/dir   Pre-selected environment",
		Flags: fs,
		Args: []ArgDef{
			{Name: "path", Required: true, Desc: "File or directory path"},
		},
		Run: func(args []string) error {
			if len(args) == 0 {
				utils.PrintGQLUsage()
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
		},
	}
}

func newHelpCmd(root *Command) *Command {
	return &Command{
		Name:  "help",
		Short: "Show help for hulak",
		Run: func(_ []string) error {
			root.printHelp(os.Stdout)
			return nil
		},
	}
}
