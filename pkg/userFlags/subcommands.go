// Builds the root command tree. Heavy leaves (run, init, doctor, gql,
// example, secrets) come from their own subpackages via New() constructors;
// trivial ones (version, migrate, help) stay here because a folder per
// 20-line handler is more friction than it's worth.
package userflags

import (
	"flag"

	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/doctor"
	"github.com/xaaha/hulak/pkg/userFlags/example"
	"github.com/xaaha/hulak/pkg/userFlags/gql"
	"github.com/xaaha/hulak/pkg/userFlags/initcmd"
	"github.com/xaaha/hulak/pkg/userFlags/mcpcmd"
	"github.com/xaaha/hulak/pkg/userFlags/runcmd"
	"github.com/xaaha/hulak/pkg/userFlags/secrets"
	"github.com/xaaha/hulak/pkg/utils"
)

// subCommands builds the full sub-command tree for hulak
func subCommands() *cli.Command {
	root := &cli.Command{
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

	root.SubCommands = []*cli.Command{
		runcmd.New(),
		newVersionCmd(),
		initcmd.New(),
		example.New(),
		newMigrateCmd(),
		newCompletionCmd(),
		doctor.New(),
		gql.New(),
		mcpcmd.New(version),
		secrets.New(),
		initcmd.NewGenDocs(
			subCommands,
			initcmd.GeneratedOutput{RelPath: "completions/hulak.zsh", Write: generateZshCompletion},
			initcmd.GeneratedOutput{RelPath: "completions/hulak.bash", Write: generateBashCompletion},
		),
		newHelpCmd(root),
	}

	return root
}

func newVersionCmd() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Short: "Print hulak version",
		Long:  "Print the current hulak version.\n\nUseful for bug reports and verifying installs.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak version", Description: "Print the installed hulak version"},
		},
		Run: func(_ []string) error {
			getVersion()
			return nil
		},
	}
}

func newMigrateCmd() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Short: "Migrate Postman collections to hulak format",
		Long: "Convert Postman v2.1 environment and collection JSON exports into hulak .hk.yaml and .env files.\n\n" +
			"Only Postman collections and environments are supported at this time.\n" +
			"To migrate plaintext env/ files to the encrypted vault, use 'hulak secrets migrate' instead.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak migrate collection.json", Description: "Migrate a Postman collection"},
			{
				Command:     "hulak migrate env.json collection.json",
				Description: "Migrate environment and collection together",
			},
		},
		Args: []cli.ArgDef{
			{Name: "files", Required: true, Desc: "Postman JSON export files", Kind: "file"},
		},
		Run: migration.CompleteMigration,
	}
}

func newHelpCmd(root *cli.Command) *cli.Command {
	return &cli.Command{
		Name:  "help",
		Short: "Show help for hulak",
		Long:  "Print the top-level hulak help.\n\nFor help on a specific command, use `hulak <command> --help` instead.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak help", Description: "Show top-level help"},
			{Command: "hulak secrets --help", Description: "Show help for a specific command"},
			{Command: "hulak secrets keys --help", Description: "Show help for a nested subcommand"},
		},
		Run: func(_ []string) error {
			root.PrintHelp()
			return nil
		},
	}
}
