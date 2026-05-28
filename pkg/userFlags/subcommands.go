// Contains the top-level command tree and factories for non-env commands
// (run, version, init, migrate, doctor, gql, help). Env subcommand factories
// live in their respective env_*.go files; newEnvCmd assembles them here.
package userflags

import (
	"flag"

	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/doctor"
	"github.com/xaaha/hulak/pkg/userFlags/example"
	"github.com/xaaha/hulak/pkg/userFlags/gql"
	"github.com/xaaha/hulak/pkg/userFlags/initcmd"
	"github.com/xaaha/hulak/pkg/userFlags/runcmd"
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
		doctor.New(),
		gql.New(),
		newEnvCmd(),
		initcmd.NewGenDocs(subCommands),
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
			{Name: "files", Required: true, Desc: "Postman JSON export files"},
		},
		Run: migration.CompleteMigration,
	}
}

func newEnvCmd() *cli.Command {
	envCmd := &cli.Command{
		Name:    "secrets",
		Aliases: []string{"env"},
		Short:   "Manage encrypted environment secrets",
		Long: "Manage environment secrets stored in the encrypted vault (.hulak/store.age).\n\n" +
			"Secrets are organized by environment (e.g. global, staging, prod).\n\n" +
			"Three concern-scoped groups live here:\n" +
			"  - this level: environment listing, edit, backup/restore, sync, migrate.\n" +
			"  - `secrets keys ...`     for key-level CRUD inside an environment.\n" +
			"  - `secrets identity ...` for age identities and recipient management.\n\n" +
			"When --env is omitted on a command that takes one, you'll be prompted\n" +
			"to pick an environment from a TUI list.\n\n" +
			"'env' is kept as an alias of `secrets` for backward compatibility.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets list",
				Description: "List environment names defined in the vault",
			},
			{
				Command:     "hulak secrets keys list --env prod",
				Description: "List keys in the prod environment (values masked)",
			},
			{
				Command:     "hulak secrets keys set API_KEY sk-123 --env prod",
				Description: "Set a key in the prod environment",
			},
			{
				Command:     "hulak secrets keys get API_KEY --env staging",
				Description: "Get a value from the staging environment",
			},
			{
				Command:     "hulak secrets keys delete OLD_KEY --env staging",
				Description: "Delete a key from the staging environment",
			},
		},
	}

	envCmd.SubCommands = []*cli.Command{
		newEnvCreateCmd(),
		newEnvDeleteCmd(),
		newEnvListCmd(),
		newEnvKeysCmd(),
		newEnvEditCmd(),
		newEnvIdentityCmd(),
		newEnvRenameCmd(),
		newEnvSyncCmd(),
		newEnvMigrateCmd(),
		newEnvBackupCmd(),
		newEnvRestoreCmd(),
	}

	return envCmd
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
