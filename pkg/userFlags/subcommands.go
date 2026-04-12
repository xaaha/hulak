package userflags

import (
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/utils"
)

// subCommands builds the full sub-command tree for hulak
func subCommands() *command {
	root := &command{
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

	root.SubCommands = []*command{
		newVersionCmd(),
		newInitCmd(),
		newMigrateCmd(),
		newDoctorCmd(),
		newGQLCmd(),
		newEnvCmd(),
		newHelpCmd(root),
	}

	return root
}

func newVersionCmd() *command {
	return &command{
		Name:  "version",
		Short: "Print hulak version",
		Long:  "Print the current hulak version.",
		Run: func(_ []string) error {
			getVersion()
			return nil
		},
	}
}

func newInitCmd() *command {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	createEnvFlag := fs.Bool(
		"env",
		false,
		"Create specific environment files instead of the default setup",
	)

	return &command{
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
		Args: []argDef{
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

func newMigrateCmd() *command {
	return &command{
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
		Args: []argDef{
			{Name: "files", Required: true, Desc: "Postman JSON export files"},
		},
		Run: migration.CompleteMigration,
	}
}

func newDoctorCmd() *command {
	return &command{
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

func newGQLCmd() *command {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	var envFlagVal string
	fs.StringVar(&envFlagVal, "env", "", "Environment to use (skips interactive selector)")
	fs.StringVar(&envFlagVal, "environment", "", "Environment to use (skips interactive selector)")

	gqlCmd := &command{
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
		Args: []argDef{
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
		data, refreshFn, warnings := loadGraphQLOperations(args[0], envFlagVal)
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

func newEnvCmd() *command {
	envCmd := &command{
		Name:  "env",
		Short: "Manage encrypted environment secrets",
		Long: "Manage environment secrets stored in the encrypted vault (.hulak/store.age).\n\n" +
			"Secrets are organized by environment (e.g. global, staging, prod).\n" +
			"The default environment is \"global\" unless --env is specified.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak env list",
				Description: "List all key-value pairs in the default environment",
			},
			{
				Command:     "hulak env set API_KEY sk-123 --env prod",
				Description: "Set a secret in the prod environment",
			},
			{
				Command:     "hulak env get API_KEY --env staging",
				Description: "Get a secret from the staging environment",
			},
			{
				Command:     "hulak env keys --env prod",
				Description: "List all keys in the prod environment",
			},
			{
				Command:     "hulak env delete OLD_KEY",
				Description: "Delete a key from the default environment",
			},
		},
	}

	// use utils.DefaultEnvVal if user does not provide env

	// registerEnvFlag adds both -env and --environment aliases to a FlagSet,
	// pointing to the same underlying variable.
	registerEnvFlag := func(fs *flag.FlagSet) {
		var envVal string
		fs.StringVar(&envVal, "env", utils.DefaultEnvVal, "Environment to operate on")
		fs.StringVar(&envVal, "environment", utils.DefaultEnvVal, "Environment to operate on")
	}

	// set — store a key-value pair
	setFs := flag.NewFlagSet("env set", flag.ContinueOnError)
	registerEnvFlag(setFs)
	setFs.Bool("stdin", false, "Read value from stdin")

	// get — retrieve a value by key
	getFs := flag.NewFlagSet("env get", flag.ContinueOnError)
	registerEnvFlag(getFs)

	// list — show all key-value pairs
	listFs := flag.NewFlagSet("env list", flag.ContinueOnError)
	registerEnvFlag(listFs)

	// keys — list keys only
	keysFs := flag.NewFlagSet("env keys", flag.ContinueOnError)
	registerEnvFlag(keysFs)
	keysFs.Bool("show", false, "Show actual values instead of masked output")

	// delete — remove a key
	deleteFs := flag.NewFlagSet("env delete", flag.ContinueOnError)
	registerEnvFlag(deleteFs)

	// edit — interactive editor
	editFs := flag.NewFlagSet("env edit", flag.ContinueOnError)
	registerEnvFlag(editFs)

	// import-key — import an age identity
	importKeyFs := flag.NewFlagSet("env import-key", flag.ContinueOnError)
	importKeyFs.Bool("stdin", false, "Read key from stdin")

	// export-key — export the age identity
	exportKeyFs := flag.NewFlagSet("env export-key", flag.ContinueOnError)
	exportKeyFs.Bool("armor", false, "Output in ASCII-armored format")

	notImplemented := func(name string) func([]string) error {
		return func(_ []string) error {
			fmt.Printf("hulak env %s is not yet implemented\n", name)
			return nil
		}
	}

	envCmd.SubCommands = []*command{
		{
			Name:    "set",
			Aliases: []string{"add"},
			Short:   "Set a key-value pair",
			Long:    "Store a secret in the encrypted vault.\n\nUse --stdin to pipe the value from standard input.",
			Flags:   setFs,
			Args: []argDef{
				{Name: "key", Required: true, Desc: "Secret key name"},
				{Name: "value", Desc: "Secret value (omit to use --stdin)"},
			},
			Run: notImplemented("set"),
		},
		{
			Name:  "get",
			Short: "Get a value by key",
			Long:  "Retrieve a secret from the encrypted vault and print it to stdout.",
			Flags: getFs,
			Args: []argDef{
				{Name: "key", Required: true, Desc: "Secret key to retrieve"},
			},
			Run: notImplemented("get"),
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Short:   "List all key-value pairs",
			Long:    "Show all secrets in an environment. Values are masked by default.",
			Flags:   listFs,
			Run:     notImplemented("list"),
		},
		{
			Name:    "keys",
			Aliases: []string{"key"},
			Short:   "List keys only",
			Long:    "Show all secret key names in an environment without values.\n\nUse --show to display actual values.",
			Flags:   keysFs,
			Run:     notImplemented("keys"),
		},
		{
			Name:    "delete",
			Aliases: []string{"rm", "remove"},
			Short:   "Delete a key",
			Long:    "Remove a secret from the encrypted vault.",
			Flags:   deleteFs,
			Args: []argDef{
				{Name: "key", Required: true, Desc: "Secret key to delete"},
			},
			Run: notImplemented("delete"),
		},
		{
			Name:  "edit",
			Short: "Edit secrets interactively",
			Long:  "Open an interactive editor for secrets in an environment.",
			Flags: editFs,
			Run:   notImplemented("edit"),
		},
		{
			Name:  "import-key",
			Short: "Import an age identity file",
			Long:  "Import an age private key from a file or stdin into the hulak config directory.",
			Flags: importKeyFs,
			Args:  []argDef{{Name: "path", Desc: "Path to the identity file (omit to read from stdin)"}},
			Run:   notImplemented("import-key"),
		},
		{
			Name:  "export-key",
			Short: "Export the age identity file",
			Long:  "Print the age private key to stdout for backup or transfer to another machine.",
			Flags: exportKeyFs,
			Run:   notImplemented("export-key"),
		},
		{
			Name:  "add-recipient",
			Short: "Add a recipient for shared vault access",
			Long:  "Add an age public key as a recipient so another user can decrypt the vault.",
			Args:  []argDef{{Name: "public-key", Required: true, Desc: "Age public key to add"}},
			Run:   notImplemented("add-recipient"),
		},
		{
			Name:  "remove-recipient",
			Short: "Remove a recipient",
			Long:  "Remove an age public key from the recipient list.",
			Args:  []argDef{{Name: "public-key", Required: true, Desc: "Age public key to remove"}},
			Run:   notImplemented("remove-recipient"),
		},
		{
			Name:  "list-recipients",
			Short: "List all recipients",
			Long:  "Show all age public keys that can decrypt the vault.",
			Run:   notImplemented("list-recipients"),
		},
	}

	return envCmd
}

func newHelpCmd(root *command) *command {
	return &command{
		Name:  "help",
		Short: "Show help for hulak",
		Run: func(_ []string) error {
			root.printHelp()
			return nil
		},
	}
}
